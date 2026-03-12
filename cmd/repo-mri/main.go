// Command repo-mri analyzes a software repository and produces a structured
// diagnostic report in .repo-mri/analysis.json.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/brentrockwood/mri/internal/aggregation"
	"github.com/brentrockwood/mri/internal/analysis"
	"github.com/brentrockwood/mri/internal/ingestion"
	"github.com/brentrockwood/mri/internal/providers"
	"github.com/brentrockwood/mri/internal/report"
	"github.com/brentrockwood/mri/schema"
)

// version, commit, and buildDate are injected at build time via -ldflags.
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	root := &cobra.Command{
		Use:     "repo-mri",
		Short:   "Analyze a software repository and produce a diagnostic report",
		Version: fmt.Sprintf("%s (commit %s, built %s)", version, commit, buildDate),
	}

	root.AddCommand(newAnalyzeCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// newAnalyzeCmd constructs the "analyze" subcommand.
func newAnalyzeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "analyze <source>",
		Short: "Analyze a repository and write .repo-mri/analysis.json",
		Long: `analyze accepts a GitHub URL (https://github.com/org/repo) or a local path.

It clones remote repositories to a temporary directory, walks the file tree,
detects languages, parses import statements, and writes the results to
.repo-mri/analysis.json under the repository root (for cloned repos this is
the temporary clone directory).`,
		Args: cobra.ExactArgs(1),
		RunE: runAnalyze,
	}
}

// runAnalyze is the entry point for the analyze subcommand.
func runAnalyze(cmd *cobra.Command, args []string) error {
	source := args[0]
	ctx := context.Background()
	start := time.Now()

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Analyzing %s…\n", source)

	result, err := ingestion.Ingest(ctx, source)
	if err != nil {
		return fmt.Errorf("analyze: %w", err)
	}
	if result.Cleanup != nil {
		defer result.Cleanup()
	}

	if err := analysis.Analyze(ctx, result.RootDir, &result.Analysis); err != nil {
		return fmt.Errorf("analyze: static analysis: %w", err)
	}

	// Select the AI analysis provider. If no API key is configured, skip AI
	// analysis and continue with static results only.
	provider, err := providers.SelectProvider(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Notice: AI analysis unavailable (%v). Continuing with static analysis only.\n", err)
	} else {
		if n, ok := provider.(providers.Namer); ok {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Provider:   %s (%s)\n", n.Name(), n.Model())
			result.Analysis.Meta.Provider = n.Name()
			result.Analysis.Meta.ModelUsed = n.Model()
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Provider:   selected\n")
		}

		findings, skipped, passErr := analysis.RunPasses(ctx, result.RootDir, &result.Analysis, provider)
		if passErr != nil {
			return fmt.Errorf("analyze: AI passes: %w", passErr)
		}

		// Convert findings to risks (IDs assigned after dedup — see below).
		for _, f := range findings {
			result.Analysis.Risks = append(result.Analysis.Risks, schema.Risk{
				Severity:      f.Severity,
				Type:          f.Type,
				Pass:          f.Type,
				Module:        moduleForFile(f.File, result.Analysis.Modules),
				File:          f.File,
				Title:         f.Title,
				Description:   f.Description,
				Confidence:    f.Confidence,
				EvidenceLines: f.EvidenceLines,
			})
		}

		if len(skipped) > 0 {
			result.Analysis.Meta.SkippedPasses = append(result.Analysis.Meta.SkippedPasses, skipped...)
		}
	}

	// Deduplicate risks and compute risk scores for files and modules.
	aggregation.Aggregate(&result.Analysis)

	// Assign sequential IDs after deduplication so there are no gaps.
	for i := range result.Analysis.Risks {
		result.Analysis.Risks[i].ID = fmt.Sprintf("risk_%03d", i+1)
	}

	if provider != nil {
		// Print findings summary.
		high, medium, low := countBySeverity(result.Analysis.Risks)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Findings:  %d (%d high, %d medium, %d low)\n",
			len(result.Analysis.Risks), high, medium, low)
	}

	// Record total pipeline duration now that all analysis is complete.
	result.Analysis.Meta.AnalysisDurationMS = time.Since(start).Milliseconds()

	// Determine output directory: .repo-mri/ under the repo root.
	outDir := filepath.Join(result.RootDir, ".repo-mri")
	if err := os.MkdirAll(outDir, 0o750); err != nil { // #nosec G301 -- output dir, not sensitive
		return fmt.Errorf("analyze: create output dir %s: %w", outDir, err)
	}

	outPath := filepath.Join(outDir, "analysis.json")
	data, err := json.MarshalIndent(result.Analysis, "", "  ")
	if err != nil {
		return fmt.Errorf("analyze: marshal analysis: %w", err)
	}

	if err := os.WriteFile(outPath, data, 0o600); err != nil { // #nosec G306 -- analysis output file
		return fmt.Errorf("analyze: write %s: %w", outPath, err)
	}

	a := result.Analysis
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Repo:       %s\n", a.Repo.Name)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Files:      %d\n", a.Repo.FileCount)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Modules:    %d\n", a.Repo.ModuleCount)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Languages:  %s\n", strings.Join(a.Repo.Languages, ", "))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Max chain:  %d\n", a.Meta.MaxChainDepth)

	if top := analysis.MostImported(a.Modules, 3); len(top) > 0 {
		names := make([]string, len(top))
		for i, m := range top {
			names[i] = fmt.Sprintf("%s(%d)", m.ID, m.ImportCount)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Most imported: %s\n", strings.Join(names, ", "))
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Output:     %s\n", outPath)

	reportPath := filepath.Join(outDir, "report.md")
	if err := report.Generate(&result.Analysis, outDir); err != nil {
		// Non-fatal: log and continue.
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: report generation failed: %v\n", err)
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Report:     %s\n", reportPath)
	}

	return nil
}

// moduleForFile returns the ID of the module whose Path is a prefix of file,
// or whose ID matches the first path component of file. Architecture findings
// that reference the synthetic graph summary are labelled "architecture".
// Falls back to "unknown".
func moduleForFile(file string, modules []schema.Module) string {
	// Synthetic chunk names used by the architecture pass.
	if file == "graph-summary" || file == "graph" {
		return "architecture"
	}
	// Try path prefix match first.
	for _, m := range modules {
		if m.Path != "" && strings.HasPrefix(file, m.Path) {
			return m.ID
		}
	}
	// Try ID match on first path component.
	first := strings.SplitN(file, "/", 2)[0]
	for _, m := range modules {
		if m.ID == first {
			return m.ID
		}
	}
	return "unknown"
}

// countBySeverity tallies risks by severity level.
func countBySeverity(risks []schema.Risk) (high, medium, low int) {
	for _, r := range risks {
		switch r.Severity {
		case "high":
			high++
		case "medium":
			medium++
		case "low":
			low++
		}
	}
	return
}

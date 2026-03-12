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

	"github.com/spf13/cobra"

	"github.com/brentrockwood/mri/internal/analysis"
	"github.com/brentrockwood/mri/internal/ingestion"
	"github.com/brentrockwood/mri/internal/providers"
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
	// analysis and continue with static results only. Phase 4 will use the
	// provider to run passes over the ingested file chunks.
	provider, err := providers.SelectProvider(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Notice: AI analysis unavailable (%v). Continuing with static analysis only.\n", err)
	} else {
		switch p := provider.(type) {
		case *providers.AnthropicProvider:
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Provider:   Anthropic (%s)\n", p.Model())
		case *providers.OpenAIProvider:
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Provider:   OpenAI (%s)\n", p.Model())
		default:
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Provider:   selected\n")
		}
	}

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

	return nil
}

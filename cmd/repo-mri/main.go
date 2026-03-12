// Command repo-mri analyzes a software repository and produces a structured
// diagnostic report in .repo-mri/analysis.json.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/brentrockwood/mri/internal/ingestion"
)

func main() {
	root := &cobra.Command{
		Use:   "repo-mri",
		Short: "Analyze a software repository and produce a diagnostic report",
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
.repo-mri/analysis.json relative to the current working directory.`,
		Args: cobra.ExactArgs(1),
		RunE: runAnalyze,
	}
}

// runAnalyze is the entry point for the analyze subcommand.
func runAnalyze(cmd *cobra.Command, args []string) error {
	source := args[0]
	ctx := context.Background()

	fmt.Fprintf(cmd.OutOrStdout(), "Analyzing %s…\n", source)

	result, err := ingestion.Ingest(ctx, source)
	if err != nil {
		return fmt.Errorf("analyze: %w", err)
	}
	if result.Cleanup != nil {
		defer result.Cleanup()
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
	fmt.Fprintf(cmd.OutOrStdout(), "Repo:      %s\n", a.Repo.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Files:     %d\n", a.Repo.FileCount)
	fmt.Fprintf(cmd.OutOrStdout(), "Modules:   %d\n", a.Repo.ModuleCount)
	fmt.Fprintf(cmd.OutOrStdout(), "Languages: %v\n", a.Repo.Languages)
	fmt.Fprintf(cmd.OutOrStdout(), "Output:    %s\n", outPath)

	return nil
}

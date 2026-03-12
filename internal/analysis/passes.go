package analysis

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/brentrockwood/mri/internal/providers"
	"github.com/brentrockwood/mri/schema"
)

const (
	chunkSize      = 50
	chunkCharLimit = 80_000
)

// contextSetter is an optional interface implemented by providers that can
// accept repo-level context for building more accurate prompts.
type contextSetter interface {
	SetAnalysisContext(languages []string)
}

// RunPasses executes the architecture, bug, and security analysis passes using
// provider and returns the combined findings. Files are read from root. Any
// passes that fail are skipped; their names are appended to the returned
// skipped slice so the caller can record them in meta.SkippedPasses.
func RunPasses(ctx context.Context, root string, a *schema.Analysis, provider providers.AnalysisProvider) ([]providers.Finding, []string, error) {
	var allFindings []providers.Finding
	var skipped []string

	// Provide repo context to providers that support it.
	if cs, ok := provider.(contextSetter); ok {
		cs.SetAnalysisContext(a.Repo.Languages)
	}

	// --- architecture pass ---
	archChunks := buildArchChunk(a)
	archFindings, err := provider.RunPass(ctx, providers.PassArchitecture, archChunks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "analysis passes: architecture pass failed: %v\n", err)
		skipped = append(skipped, string(providers.PassArchitecture))
	} else {
		allFindings = append(allFindings, archFindings...)
	}

	// --- bug and security passes (file-level chunks) ---
	fileChunks, readErr := buildFileChunks(root, a.Files)
	if readErr != nil {
		// Non-fatal: log and continue with whatever chunks were built.
		fmt.Fprintf(os.Stderr, "analysis passes: building file chunks: %v\n", readErr)
	}

	for _, pass := range []providers.PassType{providers.PassBug, providers.PassSecurity} {
		passFindings, err := runChunkedPass(ctx, provider, pass, fileChunks)
		if err != nil {
			fmt.Fprintf(os.Stderr, "analysis passes: %s pass failed: %v\n", pass, err)
			skipped = append(skipped, string(pass))
			continue
		}
		allFindings = append(allFindings, passFindings...)
	}

	return allFindings, skipped, nil
}

// buildArchChunk constructs a single FileChunk summarising the dependency graph
// from a. The chunk has Path="graph-summary" and Language="text".
func buildArchChunk(a *schema.Analysis) []providers.FileChunk {
	var sb strings.Builder
	sb.WriteString("# Dependency Graph Summary\n\n")

	sb.WriteString("## Modules\n")
	for _, m := range a.Modules {
		fmt.Fprintf(&sb, "- %s  path=%s  import_count=%d  complexity_score=%.4f\n",
			m.ID, m.Path, m.ImportCount, m.ComplexityScore)
	}

	sb.WriteString("\n## Edges\n")
	for _, d := range a.Dependencies {
		fmt.Fprintf(&sb, "- %s -> %s  type=%s\n", d.From, d.To, d.Type)
	}

	return []providers.FileChunk{
		{
			Path:     providers.GraphSummaryPath,
			Language: "text",
			Content:  sb.String(),
		},
	}
}

// buildFileChunks reads the content of each file from disk and groups the
// results into chunks bounded by chunkSize files and chunkCharLimit characters.
// Test files (e.g. _test.go, *.spec.ts) are excluded from all chunks.
func buildFileChunks(root string, files []schema.File) ([][]providers.FileChunk, error) {
	var chunks [][]providers.FileChunk
	var current []providers.FileChunk
	var currentChars int
	var firstErr error

	for _, f := range files {
		if isTestFile(f.Path) {
			continue
		}
		absPath := filepath.Join(root, f.Path)
		data, err := os.ReadFile(absPath) // #nosec G304 -- path from ingestion walk
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		// Flush current chunk if adding this file would exceed the char limit
		// (only when current is non-empty, so a single oversized file is still included).
		if len(current) > 0 && currentChars+len(data) > chunkCharLimit {
			chunks = append(chunks, current)
			current = nil
			currentChars = 0
		}
		current = append(current, providers.FileChunk{
			Path:     f.Path,
			Language: f.Language,
			Content:  string(data),
		})
		currentChars += len(data)
		if len(current) >= chunkSize {
			chunks = append(chunks, current)
			current = nil
			currentChars = 0
		}
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks, firstErr
}

// isTestFile reports whether path matches a known test file pattern.
// Test files are excluded from AI analysis passes.
func isTestFile(path string) bool {
	if strings.HasSuffix(path, "_test.go") {
		return true
	}
	for _, suffix := range []string{".test.ts", ".test.js", ".spec.ts", ".spec.js", "_spec.rb"} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

// runChunkedPass calls provider.RunPass once per chunk and aggregates findings.
// If chunks is empty, RunPass is still called once with an empty slice so that
// provider errors surface even when no files are present.
func runChunkedPass(ctx context.Context, provider providers.AnalysisProvider, pass providers.PassType, chunks [][]providers.FileChunk) ([]providers.Finding, error) {
	if len(chunks) == 0 {
		return provider.RunPass(ctx, pass, nil)
	}
	var findings []providers.Finding
	for _, chunk := range chunks {
		f, err := provider.RunPass(ctx, pass, chunk)
		if err != nil {
			return nil, err
		}
		findings = append(findings, f...)
	}
	return findings, nil
}

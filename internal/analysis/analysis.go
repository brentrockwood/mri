// Package analysis implements Phase 2 static analysis for repo-mri.
// It computes cheap, model-free signals from the ingested file tree:
// per-file size and cyclomatic complexity, per-module complexity scores,
// module import counts, and the maximum dependency chain depth.
package analysis

import (
	"context"
	"os"
	"path/filepath"

	"github.com/brentrockwood/mri/schema"
)

// Analyze enriches a with Phase 2 static analysis signals. root is the
// absolute path to the repository root. a is modified in place.
func Analyze(ctx context.Context, root string, a *schema.Analysis) error {
	// Per-file: size and cyclomatic complexity.
	for i := range a.Files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		f := &a.Files[i]
		absPath := filepath.Join(root, f.Path)

		if sz, err := fileSize(absPath); err == nil {
			f.Size = sz
		}

		raw := cyclomaticComplexity(absPath, f.Language)
		f.Complexity = normalizeComplexity(raw)
	}

	// Per-module: complexity score as mean of file complexities.
	modCC := make(map[string][]float64, len(a.Modules))
	for _, f := range a.Files {
		modCC[f.Module] = append(modCC[f.Module], f.Complexity)
	}
	for i := range a.Modules {
		vals := modCC[a.Modules[i].ID]
		if len(vals) == 0 {
			continue
		}
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		a.Modules[i].ComplexityScore = sum / float64(len(vals))
	}

	// Graph metrics: module import counts and deepest dependency chain.
	inDeg, maxDepth := graphMetrics(a.Dependencies, a.Modules)
	for i := range a.Modules {
		a.Modules[i].ImportCount = inDeg[a.Modules[i].ID]
	}
	a.Meta.MaxChainDepth = maxDepth

	return nil
}

// fileSize returns the size in bytes of the file at path.
func fileSize(path string) (int64, error) {
	info, err := os.Stat(path) // #nosec G304 -- path sourced from ingestion walk
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// normalizeComplexity maps a raw cyclomatic complexity value to [0.0, 1.0].
// A raw value of 50 or above maps to 1.0.
func normalizeComplexity(raw int) float64 {
	const cap = 50.0
	v := float64(raw) / cap
	if v > 1.0 {
		return 1.0
	}
	return v
}

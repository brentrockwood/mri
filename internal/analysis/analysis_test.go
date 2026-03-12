package analysis

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/brentrockwood/mri/schema"
)

func TestAnalyze_PopulatesComplexityAndSize(t *testing.T) {
	// Create a minimal repo with one Go file.
	root := t.TempDir()
	src := `package foo

func f(x int) int {
	if x > 0 {
		return x
	}
	return -x
}
`
	goPath := filepath.Join(root, "foo.go")
	if err := os.WriteFile(goPath, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	a := &schema.Analysis{
		Files: []schema.File{
			{Path: "foo.go", Module: "root", Language: "go"},
		},
		Modules: []schema.Module{
			{ID: "root"},
		},
	}

	if err := Analyze(context.Background(), root, a); err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	f := a.Files[0]
	if f.Size == 0 {
		t.Error("expected Size > 0")
	}
	if f.Complexity == 0 {
		t.Error("expected Complexity > 0")
	}

	m := a.Modules[0]
	if m.ComplexityScore == 0 {
		t.Error("expected ComplexityScore > 0")
	}
}

func TestAnalyze_GraphMetricsWired(t *testing.T) {
	root := t.TempDir()

	a := &schema.Analysis{
		Files: []schema.File{
			{Path: "a.go", Module: "a", Language: "go"},
		},
		Modules: []schema.Module{
			{ID: "a"},
			{ID: "b"},
		},
		Dependencies: []schema.Dependency{
			{From: "a", To: "b"},
		},
	}

	// Write a minimal Go file so Stat doesn't fail.
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Analyze(context.Background(), root, a); err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	// b should have import_count = 1; a should have 0.
	for _, m := range a.Modules {
		switch m.ID {
		case "a":
			if m.ImportCount != 0 {
				t.Errorf("a.ImportCount: want 0, got %d", m.ImportCount)
			}
		case "b":
			if m.ImportCount != 1 {
				t.Errorf("b.ImportCount: want 1, got %d", m.ImportCount)
			}
		}
	}

	if a.Meta.MaxChainDepth != 2 {
		t.Errorf("MaxChainDepth: want 2, got %d", a.Meta.MaxChainDepth)
	}
}

func TestAnalyze_ContextCancelled(t *testing.T) {
	root := t.TempDir()
	a := &schema.Analysis{
		Files: []schema.File{
			{Path: "a.go", Module: "root", Language: "go"},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Analyze(ctx, root, a); err == nil {
		t.Error("expected error from cancelled context")
	}
}

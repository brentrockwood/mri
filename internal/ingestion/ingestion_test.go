package ingestion

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// writeTestFile creates a file at root/relPath with the given content,
// creating intermediate directories as needed.
func writeTestFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
}

// TestIngest_GoPackageLevelModules verifies that Go files are assigned to
// modules at the package (directory) level rather than the top-level directory.
func TestIngest_GoPackageLevelModules(t *testing.T) {
	root := t.TempDir()

	// cmd/main.go imports internal/analysis
	writeTestFile(t, root, "cmd/main.go", `package main

import _ "github.com/test/repo/internal/analysis"
`)

	// internal/analysis/analyzer.go imports internal/ingestion
	writeTestFile(t, root, "internal/analysis/analyzer.go", `package analysis

import _ "github.com/test/repo/internal/ingestion"
`)

	// internal/ingestion/ingest.go — no internal imports
	writeTestFile(t, root, "internal/ingestion/ingest.go", `package ingestion
`)

	result, err := Ingest(context.Background(), root)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	// Collect module IDs.
	var modIDs []string
	for _, m := range result.Analysis.Modules {
		modIDs = append(modIDs, m.ID)
	}
	sort.Strings(modIDs)

	wantMods := []string{"cmd", "internal/analysis", "internal/ingestion"}
	if len(modIDs) != len(wantMods) {
		t.Fatalf("modules: got %v, want %v", modIDs, wantMods)
	}
	for i, want := range wantMods {
		if modIDs[i] != want {
			t.Errorf("module[%d]: got %q, want %q", i, modIDs[i], want)
		}
	}

	// Collect dependency edges as "from→to".
	depSet := map[string]bool{}
	for _, d := range result.Analysis.Dependencies {
		depSet[d.From+"→"+d.To] = true
	}

	wantDeps := []string{
		"cmd→internal/analysis",
		"internal/analysis→internal/ingestion",
	}
	for _, dep := range wantDeps {
		if !depSet[dep] {
			t.Errorf("missing dependency %q; got deps: %v", dep, result.Analysis.Dependencies)
		}
	}
}

// TestIngest_NonGoTopLevelModules verifies that non-Go repos continue to use
// the top-level directory as the module ID.
func TestIngest_NonGoTopLevelModules(t *testing.T) {
	root := t.TempDir()

	writeTestFile(t, root, "src/main.py", "from payments import processor\n")
	writeTestFile(t, root, "payments/processor.py", "# processor\n")

	result, err := Ingest(context.Background(), root)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	modIDs := map[string]bool{}
	for _, m := range result.Analysis.Modules {
		modIDs[m.ID] = true
	}

	if !modIDs["src"] {
		t.Errorf("expected module %q; got %v", "src", modIDs)
	}
	if !modIDs["payments"] {
		t.Errorf("expected module %q; got %v", "payments", modIDs)
	}
	// No sub-directory module IDs should exist.
	for id := range modIDs {
		if id != "src" && id != "payments" && id != "root" {
			t.Errorf("unexpected module %q", id)
		}
	}
}

// TestModuleID verifies the moduleID helper for Go and non-Go languages.
func TestModuleID(t *testing.T) {
	tests := []struct {
		relPath  string
		language string
		want     string
	}{
		// Go: package-level granularity
		{"internal/analysis/analyzer.go", "go", "internal/analysis"},
		{"internal/ingestion/ingest.go", "go", "internal/ingestion"},
		{"cmd/repo-mri/main.go", "go", "cmd/repo-mri"},
		{"schema/analysis.go", "go", "schema"},
		{"main.go", "go", "root"},
		// Non-Go: top-level directory
		{"src/main.py", "python", "src"},
		{"payments/processor.py", "python", "payments"},
		{"ui/components/Button.tsx", "typescript", "ui"},
		{"index.js", "javascript", "root"},
	}
	for _, tt := range tests {
		got := moduleID(tt.relPath, tt.language)
		if got != tt.want {
			t.Errorf("moduleID(%q, %q) = %q, want %q", tt.relPath, tt.language, got, tt.want)
		}
	}
}

// TestImportToModule verifies suffix-based module matching with longest-match.
func TestImportToModule(t *testing.T) {
	modules := map[string]bool{
		"internal/analysis":  true,
		"internal/ingestion": true,
		"internal/providers": true,
		"cmd":                true,
		"schema":             true,
		"root":               true,
	}
	tests := []struct {
		imp  string
		want string
	}{
		{"github.com/test/repo/internal/analysis", "internal/analysis"},
		{"github.com/test/repo/internal/ingestion", "internal/ingestion"},
		{"github.com/test/repo/schema", "schema"},
		{"github.com/test/repo/cmd", "cmd"},
		// No match for stdlib or external packages.
		{"fmt", ""},
		{"github.com/external/lib", ""},
		// root is never returned.
		{"github.com/test/repo", ""},
	}
	for _, tt := range tests {
		got := importToModule(tt.imp, modules)
		if got != tt.want {
			t.Errorf("importToModule(%q) = %q, want %q", tt.imp, got, tt.want)
		}
	}
}

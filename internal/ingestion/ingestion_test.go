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

	// Local analysis should populate RootPath with the absolute repo root.
	if result.Analysis.Meta.RootPath != root {
		t.Errorf("RootPath: got %q, want %q", result.Analysis.Meta.RootPath, root)
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
		relPath        string
		language       string
		jsProjectRoots []string
		want           string
	}{
		// Go: package-level granularity
		{"internal/analysis/analyzer.go", "go", nil, "internal/analysis"},
		{"internal/ingestion/ingest.go", "go", nil, "internal/ingestion"},
		{"cmd/repo-mri/main.go", "go", nil, "cmd/repo-mri"},
		{"schema/analysis.go", "go", nil, "schema"},
		{"main.go", "go", nil, "root"},
		// Non-Go (except TS/JS): top-level directory
		{"src/main.py", "python", nil, "src"},
		{"payments/processor.py", "python", nil, "payments"},
		// TypeScript and JavaScript: directory-level granularity (no project roots)
		{"ui/src/components/Button.tsx", "typescript", nil, "ui/src/components"},
		{"ui/src/lib/risk.ts", "typescript", nil, "ui/src/lib"},
		{"ui/src/App.tsx", "typescript", nil, "ui/src"},
		{"src/app.js", "javascript", nil, "src"},
		{"index.js", "javascript", nil, "root"},
		// TypeScript with project root: all ui/** → "ui"
		{"ui/vite.config.ts", "typescript", []string{"ui"}, "ui"},
		{"ui/src/App.tsx", "typescript", []string{"ui"}, "ui"},
		{"ui/src/components/Button.tsx", "typescript", []string{"ui"}, "ui"},
		{"ui/src/lib/risk.ts", "typescript", []string{"ui"}, "ui"},
		// JavaScript with project root
		{"frontend/src/index.js", "javascript", []string{"frontend"}, "frontend"},
		// File not under any project root → directory-level fallback
		{"other/index.ts", "typescript", []string{"ui"}, "other"},
	}
	for _, tt := range tests {
		got := moduleID(tt.relPath, tt.language, tt.jsProjectRoots)
		if got != tt.want {
			t.Errorf("moduleID(%q, %q, %v) = %q, want %q", tt.relPath, tt.language, tt.jsProjectRoots, got, tt.want)
		}
	}
}

// TestIngest_TypeScriptPackageLevelModules verifies that TypeScript files are
// assigned to directory-level modules and that relative imports are resolved to
// their target module so cross-directory dependencies are recorded.
func TestIngest_TypeScriptPackageLevelModules(t *testing.T) {
	root := t.TempDir()

	// ui/src/App.tsx imports from ui/src/components and ui/src/lib
	writeTestFile(t, root, "ui/src/App.tsx", `
import { Inspector } from './components/Inspector'
import { risk } from './lib/risk'
`)
	// ui/src/components/Inspector.tsx imports from ui/src/lib
	writeTestFile(t, root, "ui/src/components/Inspector.tsx", `
import { risk } from '../lib/risk'
`)
	// ui/src/lib/risk.ts — no imports
	writeTestFile(t, root, "ui/src/lib/risk.ts", `export const risk = 1`)

	result, err := Ingest(context.Background(), root)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	modIDs := map[string]bool{}
	for _, m := range result.Analysis.Modules {
		modIDs[m.ID] = true
	}

	wantMods := []string{"ui/src", "ui/src/components", "ui/src/lib"}
	for _, want := range wantMods {
		if !modIDs[want] {
			t.Errorf("expected module %q; got %v", want, modIDs)
		}
	}

	depSet := map[string]bool{}
	for _, d := range result.Analysis.Dependencies {
		depSet[d.From+"→"+d.To] = true
	}

	wantDeps := []string{
		"ui/src→ui/src/components",
		"ui/src→ui/src/lib",
		"ui/src/components→ui/src/lib",
	}
	for _, dep := range wantDeps {
		if !depSet[dep] {
			t.Errorf("missing dependency %q; got deps: %v", dep, result.Analysis.Dependencies)
		}
	}
}

// TestIngest_TSProjectRootGrouping verifies that when a non-root package.json
// exists, all TS/JS files under that directory are assigned to a single module.
func TestIngest_TSProjectRootGrouping(t *testing.T) {
	root := t.TempDir()

	// ui/package.json marks the ui subtree as one JS project.
	writeTestFile(t, root, "ui/package.json", `{"name":"ui"}`)
	writeTestFile(t, root, "ui/vite.config.ts", `// vite config`)
	writeTestFile(t, root, "ui/src/App.tsx", `// app`)
	writeTestFile(t, root, "ui/src/components/Button.tsx", `// button`)
	writeTestFile(t, root, "ui/src/lib/risk.ts", `export const risk = 1`)

	result, err := Ingest(context.Background(), root)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	modIDs := map[string]bool{}
	for _, m := range result.Analysis.Modules {
		modIDs[m.ID] = true
	}

	// All TS files under ui/ should collapse into a single "ui" module.
	if !modIDs["ui"] {
		t.Errorf("expected module %q; got %v", "ui", modIDs)
	}
	for id := range modIDs {
		if id != "ui" {
			t.Errorf("unexpected module %q (expected only \"ui\")", id)
		}
	}

	// All files in the same module → no dependencies.
	if len(result.Analysis.Dependencies) != 0 {
		t.Errorf("expected no dependencies within single module; got %v", result.Analysis.Dependencies)
	}

	// The "ui" module should have 4 TypeScript files.
	for _, m := range result.Analysis.Modules {
		if m.ID == "ui" && m.FileCount != 4 {
			t.Errorf("module %q: FileCount = %d, want 4", m.ID, m.FileCount)
		}
	}
}

// TestIngest_LocalRepoHasNoGithubSlug verifies that analysing a local directory
// does not populate the GithubSlug field (it is only set for GitHub-sourced clones).
func TestIngest_LocalRepoHasNoGithubSlug(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "main.go", "package main\n")
	result, err := Ingest(context.Background(), root)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if result.Analysis.Repo.GithubSlug != "" {
		t.Errorf("expected empty GithubSlug for local repo, got %q", result.Analysis.Repo.GithubSlug)
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

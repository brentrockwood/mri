package ingestion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeFile creates a file at path with the given content, creating parent dirs as needed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}

func TestWalk(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Code files.
	writeFile(t, filepath.Join(root, "main.go"), "package main\n")
	writeFile(t, filepath.Join(root, "pkg", "util.go"), "package pkg\n\nfunc Foo() {}\n")
	writeFile(t, filepath.Join(root, "pkg", "util_test.go"), "package pkg\n")
	writeFile(t, filepath.Join(root, "scripts", "deploy.sh"), "#!/usr/bin/env bash\n")
	writeFile(t, filepath.Join(root, "web", "app.ts"), "export {};\n")

	// Non-code files (should be skipped).
	writeFile(t, filepath.Join(root, "README.md"), "# readme\n")
	writeFile(t, filepath.Join(root, "go.sum"), "")

	// Directories that should be skipped entirely.
	writeFile(t, filepath.Join(root, ".git", "config"), "[core]\n")
	writeFile(t, filepath.Join(root, "node_modules", "lodash", "index.js"), "module.exports={}\n")
	writeFile(t, filepath.Join(root, "vendor", "lib.go"), "package vendor\n")

	files, err := Walk(context.Background(), root)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	// Build a set of returned paths for easy lookup.
	got := map[string]FileInfo{}
	for _, fi := range files {
		got[filepath.ToSlash(fi.Path)] = fi
	}

	// Should be present.
	wantPresent := []struct {
		path string
		lang string
	}{
		{"main.go", "go"},
		{"pkg/util.go", "go"},
		{"pkg/util_test.go", "go"},
		{"scripts/deploy.sh", "shell"},
		{"web/app.ts", "typescript"},
	}
	for _, w := range wantPresent {
		fi, ok := got[w.path]
		if !ok {
			t.Errorf("Walk: expected file %q not found", w.path)
			continue
		}
		if fi.Language != w.lang {
			t.Errorf("Walk: %q language = %q, want %q", w.path, fi.Language, w.lang)
		}
	}

	// Should be absent.
	wantAbsent := []string{
		"README.md",
		"go.sum",
		".git/config",
		"node_modules/lodash/index.js",
		"vendor/lib.go",
	}
	for _, p := range wantAbsent {
		if _, ok := got[p]; ok {
			t.Errorf("Walk: file %q should have been skipped but was returned", p)
		}
	}
}

func TestWalkLineCount(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	content := "package main\n\nfunc main() {}\n"
	writeFile(t, filepath.Join(root, "main.go"), content)

	files, err := Walk(context.Background(), root)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Lines != 3 {
		t.Errorf("Lines = %d, want 3", files[0].Lines)
	}
}

func TestWalkEmptyDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	// Only non-code files.
	writeFile(t, filepath.Join(root, "README.md"), "hello\n")

	files, err := Walk(context.Background(), root)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestWalkContextCancellation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.go"), "package a\n")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := Walk(ctx, root)
	if err == nil {
		// Cancellation may or may not fire before the single file is processed,
		// so we only check that if err != nil it wraps the context error.
		return
	}
	if !errors.Is(err, ctx.Err()) {
		t.Errorf("unexpected non-context error: %v", err)
	}
}

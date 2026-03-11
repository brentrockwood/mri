package ingestion

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo holds per-file metadata collected during the walk.
type FileInfo struct {
	// Path is the file path relative to the repo root.
	Path string
	// Language is the canonical language name detected from the file extension.
	Language string
	// Lines is the number of lines in the file.
	Lines int
}

// skipDirs is the set of directory names to skip entirely during the walk.
var skipDirs = map[string]bool{
	// version control
	".git": true,
	".svn": true,
	".hg":  true,
	// dependency/build directories
	"node_modules":  true,
	"vendor":        true,
	"__pycache__":   true,
	".pytest_cache": true,
	"venv":          true,
	".venv":         true,
	"env":           true,
	".env":          true,
	"dist":          true,
	"build":         true,
	"target":        true,
	".idea":         true,
	".vscode":       true,
	// output dirs
	"bin": true,
	"out": true,
	"obj": true,
}

// Walk traverses the directory tree rooted at root and returns metadata
// for all recognized code files.
func Walk(ctx context.Context, root string) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("ingestion walk: accessing %s: %w", path, err)
		}

		// Respect context cancellation.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		name := d.Name()

		if d.IsDir() {
			// Skip hidden directories and known non-code directories.
			if strings.HasPrefix(name, ".") || skipDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}

		lang := DetectLanguage(name)
		if lang == "" {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("ingestion walk: relative path for %s: %w", path, err)
		}

		lines, err := countLines(path)
		if err != nil {
			// Non-fatal: record 0 lines.
			lines = 0
		}

		files = append(files, FileInfo{
			Path:     rel,
			Language: lang,
			Lines:    lines,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ingestion walk: %w", err)
	}

	return files, nil
}

// countLines returns the number of newline-delimited lines in the file at path.
func countLines(path string) (int, error) {
	f, err := os.Open(path) //nolint:gosec // path comes from filepath.WalkDir
	if err != nil {
		return 0, err
	}
	defer f.Close() //nolint:errcheck

	n := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		n++
	}
	return n, sc.Err()
}

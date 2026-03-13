// Package ingestion implements the Phase 1 ingestion pipeline for repo-mri.
// It accepts a GitHub URL or local path, walks the file tree, detects languages,
// parses import statements, and returns a partially-populated schema.Analysis.
package ingestion

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/brentrockwood/mri/schema"
)

// Result holds the output of the ingestion phase.
type Result struct {
	// RootDir is the absolute path to the repo root (temp dir if cloned).
	RootDir string
	// Cleanup removes the temporary directory created during cloning.
	// It is nil when the source is a local path.
	Cleanup func()
	// Analysis is the partially-populated analysis (no risks yet).
	Analysis schema.Analysis
}

// Ingest runs the full ingestion pipeline for the given source.
// source may be a GitHub URL (https://...) or a local path.
func Ingest(ctx context.Context, source string) (*Result, error) {
	start := time.Now()

	var root string
	var cleanup func()

	if isRemoteURL(source) {
		var err error
		root, cleanup, err = Clone(ctx, source)
		if err != nil {
			return nil, err
		}
	} else {
		abs, err := filepath.Abs(source)
		if err != nil {
			return nil, fmt.Errorf("ingestion ingest: resolve path %s: %w", source, err)
		}
		root = abs
		cleanup = nil
	}

	files, err := Walk(ctx, root)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, fmt.Errorf("ingestion ingest: walk: %w", err)
	}

	if len(files) == 0 {
		if cleanup != nil {
			cleanup()
		}
		return nil, fmt.Errorf("ingestion ingest: no code files found in %s", source)
	}

	jsProjectRoots := findJSProjectRoots(root)

	// Build module map: top-level subdirectory name → module.
	// Files directly in root get module ID "root".
	moduleFiles := map[string][]FileInfo{}
	moduleLang := map[string]map[string]int{} // module → language → count

	for _, fi := range files {
		mod := moduleID(fi.Path, fi.Language, jsProjectRoots)
		moduleFiles[mod] = append(moduleFiles[mod], fi)
		if moduleLang[mod] == nil {
			moduleLang[mod] = map[string]int{}
		}
		moduleLang[mod][fi.Language]++
	}

	// Collect unique languages across all files.
	langSet := map[string]bool{}
	for _, fi := range files {
		langSet[fi.Language] = true
	}
	languages := sortedKeys(langSet)

	// Build schema.Module slice.
	var modules []schema.Module
	for modID, fis := range moduleFiles {
		lang := dominantLanguage(moduleLang[modID])
		modules = append(modules, schema.Module{
			ID:        modID,
			Path:      modulePath(root, modID),
			Language:  lang,
			FileCount: len(fis),
		})
	}
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].ID < modules[j].ID
	})

	// Build schema.File slice and parse imports.
	moduleSet := map[string]bool{}
	for _, m := range modules {
		moduleSet[m.ID] = true
	}

	depSet := map[string]bool{} // "from→to" dedup key
	var dependencies []schema.Dependency
	var schemaFiles []schema.File

	for _, fi := range files {
		mod := moduleID(fi.Path, fi.Language, jsProjectRoots)

		schemaFiles = append(schemaFiles, schema.File{
			Path:     fi.Path,
			Module:   mod,
			Language: fi.Language,
			Lines:    fi.Lines,
		})

		absPath := filepath.Join(root, fi.Path)
		imps, err := ParseImports(ctx, absPath, fi.Language)
		if err != nil {
			// Non-fatal: skip import parsing for this file.
			continue
		}

		for _, imp := range imps {
			var toMod string
			if (fi.Language == "javascript" || fi.Language == "typescript") && strings.HasPrefix(imp, ".") {
				// Resolve relative import (e.g. "../lib/risk") to its module directory.
				fileDir := path.Dir(filepath.ToSlash(fi.Path))
				candidate := path.Dir(path.Clean(path.Join(fileDir, imp)))
				if moduleSet[candidate] {
					toMod = candidate
				}
			} else {
				toMod = importToModule(imp, moduleSet)
			}
			if toMod == "" || toMod == mod {
				continue
			}
			key := mod + "→" + toMod
			if !depSet[key] {
				depSet[key] = true
				dependencies = append(dependencies, schema.Dependency{
					From: mod,
					To:   toMod,
					Type: "import",
				})
			}
		}
	}

	sort.Slice(schemaFiles, func(i, j int) bool {
		return schemaFiles[i].Path < schemaFiles[j].Path
	})
	sort.Slice(dependencies, func(i, j int) bool {
		if dependencies[i].From != dependencies[j].From {
			return dependencies[i].From < dependencies[j].From
		}
		return dependencies[i].To < dependencies[j].To
	})

	repoName := filepath.Base(root)
	var githubSlug string
	if isRemoteURL(source) {
		if u, err := url.Parse(source); err == nil {
			slug := strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"), "/")
			slug = strings.TrimSuffix(slug, ".git")
			repoName = path.Base(slug)
			// Capture the full "owner/repo" slug for GitHub deep links.
			if strings.Count(slug, "/") == 1 {
				githubSlug = slug
			}
		}
	}

	// RootPath is only meaningful for local analyses; for remote repos the
	// clone lands in a temp directory that has no value to the end user.
	var rootPath string
	if !isRemoteURL(source) {
		rootPath = root
	}

	analysis := schema.Analysis{
		Meta: schema.Meta{
			SchemaVersion:      schema.SchemaVersion,
			CLIVersion:         schema.CLIVersion,
			AnalysisDurationMS: time.Since(start).Milliseconds(),
			RootPath:           rootPath,
		},
		Repo: schema.Repo{
			Name:         repoName,
			GithubSlug:   githubSlug,
			Languages:    languages,
			FileCount:    len(files),
			ModuleCount:  len(modules),
			AnalysisTime: time.Now().UTC(),
		},
		Modules:      modules,
		Dependencies: dependencies,
		Risks:        []schema.Risk{},
		Files:        schemaFiles,
	}

	return &Result{
		RootDir:  root,
		Cleanup:  cleanup,
		Analysis: analysis,
	}, nil
}

// isRemoteURL reports whether source looks like a remote URL.
func isRemoteURL(source string) bool {
	return strings.HasPrefix(source, "https://") ||
		strings.HasPrefix(source, "http://") ||
		strings.HasPrefix(source, "git@")
}

// moduleID returns the module ID for a file at the given relative path.
// For Go files, the module ID is the repo-relative directory path (directory-level granularity).
// For TypeScript and JavaScript files, if the file lives under a JS project root
// (a directory containing a non-root package.json), the project root directory is returned.
// Otherwise TS/JS falls back to directory-level granularity.
// For all other languages the top-level directory heuristic is used.
// Files directly in the repo root get module ID "root".
func moduleID(relPath, language string, jsProjectRoots []string) string {
	slashPath := filepath.ToSlash(relPath)
	if language == "go" {
		idx := strings.LastIndex(slashPath, "/")
		if idx == -1 {
			return "root"
		}
		return slashPath[:idx]
	}
	if language == "typescript" || language == "javascript" {
		for _, proj := range jsProjectRoots {
			if strings.HasPrefix(slashPath, proj+"/") {
				return proj
			}
		}
		// Fallback: directory-level granularity.
		idx := strings.LastIndex(slashPath, "/")
		if idx == -1 {
			return "root"
		}
		return slashPath[:idx]
	}
	idx := strings.Index(slashPath, "/")
	if idx == -1 {
		return "root"
	}
	return slashPath[:idx]
}

// findJSProjectRoots walks the tree and returns repo-relative slash paths of
// directories that contain a package.json (excluding the repo root itself).
// It respects the same skipDirs and hidden-directory rules as the file walker.
func findJSProjectRoots(root string) []string {
	var roots []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "package.json" {
			dir := filepath.Dir(p)
			if dir == root {
				return nil // skip repo-root package.json
			}
			if rel, err := filepath.Rel(root, dir); err == nil {
				roots = append(roots, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	return roots
}

// modulePath returns a display path for the module.
func modulePath(root, modID string) string {
	if modID == "root" {
		return filepath.Base(root)
	}
	return modID
}

// dominantLanguage returns the language with the most files in the map.
func dominantLanguage(counts map[string]int) string {
	best, bestCount := "", 0
	for lang, n := range counts {
		if n > bestCount || (n == bestCount && lang < best) {
			best, bestCount = lang, n
		}
	}
	return best
}

// importToModule maps an import path to the longest-matching module ID in the
// set. It returns the module ID whose value equals imp or appears as a
// slash-delimited suffix of imp (e.g. "internal/analysis" matches
// "github.com/user/repo/internal/analysis"). Longest match wins so that
// "internal/analysis" is preferred over "internal" when both are present.
func importToModule(imp string, modules map[string]bool) string {
	imp = filepath.ToSlash(imp)
	best := ""
	for modID := range modules {
		if modID == "root" {
			continue
		}
		if imp == modID || strings.HasSuffix(imp, "/"+modID) {
			if len(modID) > len(best) {
				best = modID
			}
		}
	}
	return best
}

// sortedKeys returns the keys of a bool map in sorted order.
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

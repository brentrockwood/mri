package ingestion

import (
	"bufio"
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ParseImports returns the list of import paths found in the given source file.
// The language parameter is the canonical language name from DetectLanguage.
// Returns nil, nil if the language is not supported for import parsing.
func ParseImports(ctx context.Context, path, language string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	switch language {
	case "go":
		return parseGoImports(path)
	case "python":
		return parsePythonImports(path)
	case "javascript", "typescript":
		return parseJSImports(path)
	case "java":
		return parseJavaImports(path)
	default:
		return nil, nil
	}
}

// parseGoImports uses the Go AST to extract import paths from a .go file.
func parseGoImports(path string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("ingestion imports go parse %s: %w", path, err)
	}

	var imports []string
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		imports = append(imports, p)
	}
	return imports, nil
}

// rePythonImport matches "import foo.bar" and "from foo.bar import ...".
var rePythonImport = regexp.MustCompile(`^(?:from\s+([\w.]+)\s+import|import\s+([\w.,\s]+))`)

// parsePythonImports extracts top-level import statements from a Python file.
func parsePythonImports(path string) ([]string, error) {
	lines, err := readLines(path)
	if err != nil {
		return nil, fmt.Errorf("ingestion imports python read %s: %w", path, err)
	}

	seen := map[string]bool{}
	var imports []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		m := rePythonImport.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if m[1] != "" {
			// from X import ...
			pkg := strings.TrimSpace(m[1])
			if !seen[pkg] {
				seen[pkg] = true
				imports = append(imports, pkg)
			}
		} else {
			// import X, Y, Z
			for _, part := range strings.Split(m[2], ",") {
				fields := strings.Fields(part)
				if len(fields) == 0 {
					continue
				}
				pkg := fields[0]
				if !seen[pkg] {
					seen[pkg] = true
					imports = append(imports, pkg)
				}
			}
		}
	}
	return imports, nil
}

// reJSImport matches ES6 import statements and require() calls.
var reJSImport = regexp.MustCompile(
	`(?:import\s+[^'"]*\s+from\s+['"]([^'"]+)['"]|` +
		`import\s+['"]([^'"]+)['"]|` +
		`require\s*\(\s*['"]([^'"]+)['"]\s*\))`,
)

// parseJSImports extracts import/require paths from a JS or TS file.
func parseJSImports(path string) ([]string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path sourced from Walk
	if err != nil {
		return nil, fmt.Errorf("ingestion imports js read %s: %w", path, err)
	}

	seen := map[string]bool{}
	var imports []string

	for _, m := range reJSImport.FindAllStringSubmatch(string(data), -1) {
		var pkg string
		switch {
		case m[1] != "":
			pkg = m[1]
		case m[2] != "":
			pkg = m[2]
		case m[3] != "":
			pkg = m[3]
		}
		if pkg != "" && !seen[pkg] {
			seen[pkg] = true
			imports = append(imports, pkg)
		}
	}
	return imports, nil
}

// reJavaImport matches "import com.example.Foo;" and wildcard "import java.util.*;" statements.
var reJavaImport = regexp.MustCompile(`^import\s+(static\s+)?([\w.*]+)\s*;`)

// parseJavaImports extracts import statements from a Java file.
func parseJavaImports(path string) ([]string, error) {
	lines, err := readLines(path)
	if err != nil {
		return nil, fmt.Errorf("ingestion imports java read %s: %w", path, err)
	}

	seen := map[string]bool{}
	var imports []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		m := reJavaImport.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		pkg := m[2]
		if !seen[pkg] {
			seen[pkg] = true
			imports = append(imports, pkg)
		}
	}
	return imports, nil
}

// readLines reads a file and returns its lines as a slice of strings.
func readLines(path string) ([]string, error) {
	f, err := os.Open(path) // #nosec G304 -- path sourced from Walk
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

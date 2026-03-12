package analysis

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
)

// cyclomaticComplexity computes a best-effort cyclomatic complexity for the
// file at path. For Go files it uses the AST; for other languages it uses a
// keyword-counting heuristic. Returns 1 (the minimum) on error or for
// unsupported languages.
func cyclomaticComplexity(path, language string) int {
	switch language {
	case "go":
		return goComplexity(path)
	case "python":
		return keywordComplexity(path, rePythonCC)
	case "javascript", "typescript":
		return keywordComplexity(path, reJSCC)
	case "java", "csharp", "c", "cpp", "rust", "ruby", "shell":
		return keywordComplexity(path, reCLikeCC)
	default:
		return 1
	}
}

// goComplexity counts cyclomatic complexity for a Go source file via the AST.
// It counts one decision point per: if, for, range, select case, switch case,
// && and || binary expressions.
func goComplexity(path string) int {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return 1
	}
	v := &ccVisitor{count: 1}
	ast.Walk(v, f)
	return v.count
}

// ccVisitor accumulates cyclomatic complexity while walking a Go AST.
type ccVisitor struct {
	count int
}

// Visit implements ast.Visitor.
func (v *ccVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.IfStmt:
		v.count++
	case *ast.ForStmt:
		v.count++
	case *ast.RangeStmt:
		v.count++
	case *ast.CaseClause:
		// Count non-default switch cases. The default clause has a nil List.
		if n.List != nil {
			v.count++
		}
	case *ast.CommClause:
		// All select cases including default.
		v.count++
	case *ast.BinaryExpr:
		if n.Op == token.LAND || n.Op == token.LOR {
			v.count++
		}
	}
	return v
}

// rePythonCC matches Python branching keywords and logical operators.
var rePythonCC = regexp.MustCompile(`\b(if|elif|for|while|except|with|and|or)\b`)

// reJSCC matches JavaScript/TypeScript branching keywords and logical operators.
// \?\? matches nullish coalescing; (?:\?(?:[^.?]|$)) matches ternary ? while
// excluding optional chaining ?. and nullish coalescing ??.
var reJSCC = regexp.MustCompile(`\b(if|else\s+if|for|while|do|switch|catch)\b|&&|\|\||\?\?|\?(?:[^.?]|$)`)

// reCLikeCC matches C-like language branching keywords and logical operators.
var reCLikeCC = regexp.MustCompile(`\b(if|else\s+if|for|while|do|switch|catch|case)\b|&&|\|\|`)

// keywordComplexity estimates cyclomatic complexity for non-Go files by
// counting branching keyword matches in the raw source text.
func keywordComplexity(path string, re *regexp.Regexp) int {
	data, err := os.ReadFile(path) // #nosec G304 -- path sourced from ingestion walk
	if err != nil {
		return 1
	}
	return 1 + len(re.FindAllIndex(data, -1))
}

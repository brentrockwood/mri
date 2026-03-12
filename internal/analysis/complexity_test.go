package analysis

import (
	"os"
	"testing"
)

func TestGoComplexity_Simple(t *testing.T) {
	src := `package main
func f() {}
`
	if got := goComplexityFromSrc(t, src); got != 1 {
		t.Errorf("simple function: want 1, got %d", got)
	}
}

func TestGoComplexity_IfFor(t *testing.T) {
	src := `package main
func f(x int) int {
	if x > 0 {
		for i := 0; i < x; i++ {
			_ = i
		}
	}
	return x
}
`
	// 1 base + 1 if + 1 for = 3
	if got := goComplexityFromSrc(t, src); got != 3 {
		t.Errorf("if+for: want 3, got %d", got)
	}
}

func TestGoComplexity_SwitchCase(t *testing.T) {
	src := `package main
func f(x int) string {
	switch x {
	case 1:
		return "one"
	case 2:
		return "two"
	default:
		return "other"
	}
}
`
	// 1 base + 2 non-default cases = 3
	if got := goComplexityFromSrc(t, src); got != 3 {
		t.Errorf("switch: want 3, got %d", got)
	}
}

func TestGoComplexity_LogicalOps(t *testing.T) {
	src := `package main
func f(a, b, c bool) bool {
	return a && b || c
}
`
	// 1 base + 1 && + 1 || = 3
	if got := goComplexityFromSrc(t, src); got != 3 {
		t.Errorf("logical ops: want 3, got %d", got)
	}
}

func TestGoComplexity_Range(t *testing.T) {
	src := `package main
func f(xs []int) {
	for _, x := range xs {
		_ = x
	}
}
`
	// 1 base + 1 range = 2
	if got := goComplexityFromSrc(t, src); got != 2 {
		t.Errorf("range: want 2, got %d", got)
	}
}

func TestNormalizeComplexity(t *testing.T) {
	cases := []struct {
		raw  int
		want float64
	}{
		{1, 0.02},
		{25, 0.5},
		{50, 1.0},
		{100, 1.0},
	}
	for _, c := range cases {
		got := normalizeComplexity(c.raw)
		if got != c.want {
			t.Errorf("normalizeComplexity(%d) = %f, want %f", c.raw, got, c.want)
		}
	}
}

func TestKeywordComplexity_Python(t *testing.T) {
	src := `def f(x):
    if x > 0:
        for i in range(x):
            if i > 0 and i < x:
                pass
`
	path := writeTempFile(t, src, "*.py")
	// 1 base + if + for + if + and = 5
	if got := keywordComplexity(path, rePythonCC); got != 5 {
		t.Errorf("python keyword: want 5, got %d", got)
	}
}

func TestKeywordComplexity_JS(t *testing.T) {
	src := `function f(x) {
  if (x > 0 && x < 10) {
    for (let i = 0; i < x; i++) {}
  }
}
`
	path := writeTempFile(t, src, "*.js")
	// 1 base + if + && + for = 4
	if got := keywordComplexity(path, reJSCC); got != 4 {
		t.Errorf("js keyword: want 4, got %d", got)
	}
}

// goComplexityFromSrc writes src to a temp Go file and calls goComplexity.
func goComplexityFromSrc(t *testing.T, src string) int {
	t.Helper()
	path := writeTempFile(t, src, "*.go")
	return goComplexity(path)
}

// writeTempFile creates a temp file with the given content and name pattern.
func writeTempFile(t *testing.T, content, pattern string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), pattern)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

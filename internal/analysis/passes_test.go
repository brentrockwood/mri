package analysis

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brentrockwood/mri/internal/providers"
	"github.com/brentrockwood/mri/schema"
)

// mockProvider records every RunPass call and optionally implements
// contextSetter so tests can verify analysis context propagation.
type mockProvider struct {
	calls      []mockCall
	errFor     map[providers.PassType]error
	contextSet bool
	languages  []string
}

type mockCall struct {
	pass   providers.PassType
	chunks []providers.FileChunk
}

func (m *mockProvider) RunPass(_ context.Context, pass providers.PassType, chunks []providers.FileChunk) ([]providers.Finding, error) {
	m.calls = append(m.calls, mockCall{pass: pass, chunks: chunks})
	if err, ok := m.errFor[pass]; ok {
		return nil, err
	}
	return nil, nil
}

func (m *mockProvider) SetAnalysisContext(languages []string) {
	m.contextSet = true
	m.languages = languages
}

// writeTempFiles creates files under a temp directory and returns the dir path.
func writeTempFiles(t *testing.T, n int) (root string, files []schema.File) {
	t.Helper()
	root = t.TempDir()
	for i := 0; i < n; i++ {
		name := filepath.Join(root, filepath.FromSlash("src/file"+string(rune('A'+i%26))+".go"))
		if err := os.MkdirAll(filepath.Dir(name), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(name, []byte("package src\n"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		rel, _ := filepath.Rel(root, name)
		files = append(files, schema.File{
			Path:     filepath.ToSlash(rel),
			Module:   "src",
			Language: "Go",
		})
	}
	return root, files
}

func TestRunPasses_ChunkingBugAndSecurity(t *testing.T) {
	const fileCount = 110 // produces ceil(110/50) = 3 chunks per file-based pass

	root, files := writeTempFiles(t, fileCount)
	a := &schema.Analysis{Files: files}
	mp := &mockProvider{}

	_, skipped, err := RunPasses(context.Background(), root, a, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("expected no skipped passes, got %v", skipped)
	}

	expectedChunks := (fileCount + chunkSize - 1) / chunkSize // ceil(110/50) = 3

	bugCalls := callsFor(mp.calls, providers.PassBug)
	if len(bugCalls) != expectedChunks {
		t.Errorf("bug pass: got %d calls, want %d", len(bugCalls), expectedChunks)
	}

	secCalls := callsFor(mp.calls, providers.PassSecurity)
	if len(secCalls) != expectedChunks {
		t.Errorf("security pass: got %d calls, want %d", len(secCalls), expectedChunks)
	}
}

func TestRunPasses_ArchitectureChunk(t *testing.T) {
	root := t.TempDir()
	a := &schema.Analysis{
		Modules: []schema.Module{
			{ID: "cmd", Path: "cmd"},
			{ID: "internal/analysis", Path: "internal/analysis"},
		},
		Dependencies: []schema.Dependency{
			{From: "cmd", To: "internal/analysis", Type: "import"},
		},
	}
	mp := &mockProvider{}

	_, _, err := RunPasses(context.Background(), root, a, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	archCalls := callsFor(mp.calls, providers.PassArchitecture)
	if len(archCalls) != 1 {
		t.Fatalf("architecture pass: got %d calls, want 1", len(archCalls))
	}
	if len(archCalls[0].chunks) != 1 {
		t.Fatalf("architecture pass: got %d chunks, want 1", len(archCalls[0].chunks))
	}
	if archCalls[0].chunks[0].Path != providers.GraphSummaryPath {
		t.Errorf("architecture chunk path: got %q, want %q", archCalls[0].chunks[0].Path, providers.GraphSummaryPath)
	}
}

func TestRunPasses_SkippedOnError(t *testing.T) {
	root := t.TempDir()
	a := &schema.Analysis{}
	mp := &mockProvider{
		errFor: map[providers.PassType]error{
			providers.PassBug: errors.New("boom"),
		},
	}

	_, skipped, err := RunPasses(context.Background(), root, a, mp)
	if err != nil {
		t.Fatalf("unexpected hard error: %v", err)
	}
	if len(skipped) != 1 || skipped[0] != string(providers.PassBug) {
		t.Errorf("skipped: got %v, want [%s]", skipped, providers.PassBug)
	}
}

func TestRunPasses_AllPassesSkipped(t *testing.T) {
	root := t.TempDir()
	a := &schema.Analysis{}
	boom := errors.New("api error")
	mp := &mockProvider{
		errFor: map[providers.PassType]error{
			providers.PassArchitecture: boom,
			providers.PassBug:          boom,
			providers.PassSecurity:     boom,
		},
	}

	findings, skipped, err := RunPasses(context.Background(), root, a, mp)
	if err != nil {
		t.Fatalf("unexpected hard error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
	if len(skipped) != 3 {
		t.Errorf("expected 3 skipped passes, got %v", skipped)
	}
}

func TestBuildFileChunks_ExcludesTestFiles(t *testing.T) {
	root := t.TempDir()

	write := func(name, content string) schema.File {
		abs := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		return schema.File{Path: name, Language: "Go"}
	}

	files := []schema.File{
		write("src/app.go", "package src\n"),
		write("src/app_test.go", "package src\n"),
		write("ui/Button.test.ts", "// test\n"),
		write("ui/Button.spec.js", "// test\n"),
		write("spec/model_spec.rb", "# test\n"),
		write("cmd/main.go", "package main\n"),
	}

	chunks, err := buildFileChunks(root, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, chunk := range chunks {
		for _, fc := range chunk {
			if isTestFile(fc.Path) {
				t.Errorf("test file %q found in chunk; should be excluded", fc.Path)
			}
		}
	}

	// Confirm the two non-test files are present.
	var paths []string
	for _, chunk := range chunks {
		for _, fc := range chunk {
			paths = append(paths, fc.Path)
		}
	}
	if len(paths) != 2 {
		t.Errorf("expected 2 non-test files in chunks, got %d: %v", len(paths), paths)
	}
}

func TestBuildFileChunks_CharLimitSplits(t *testing.T) {
	// Create files whose combined size exceeds chunkCharLimit but file count
	// stays well below chunkSize, so only the char limit triggers the split.
	root := t.TempDir()

	// Each file is just over chunkCharLimit/2 bytes so two files exceed the limit.
	halfPlus := chunkCharLimit/2 + 1
	content := strings.Repeat("x", halfPlus)

	write := func(name string) schema.File {
		abs := filepath.Join(root, name)
		if err := os.WriteFile(abs, []byte(content), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		return schema.File{Path: name, Language: "Go"}
	}

	files := []schema.File{
		write("a.go"),
		write("b.go"),
		write("c.go"),
	}

	chunks, err := buildFileChunks(root, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With each file just over 40 000 chars, a.go and b.go cannot share a chunk
	// (combined > 80 000). Expected layout: [a], [b], [c].
	if len(chunks) != 3 {
		t.Errorf("expected 3 chunks (char limit splits), got %d", len(chunks))
	}
	for i, chunk := range chunks {
		if len(chunk) != 1 {
			t.Errorf("chunk %d: expected 1 file, got %d", i, len(chunk))
		}
	}
}

func TestRunPasses_SetsAnalysisContext(t *testing.T) {
	root := t.TempDir()
	a := &schema.Analysis{
		Repo: schema.Repo{Languages: []string{"go", "shell"}},
	}
	mp := &mockProvider{}

	_, _, err := RunPasses(context.Background(), root, a, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mp.contextSet {
		t.Error("expected SetAnalysisContext to be called on provider")
	}
	if len(mp.languages) != 2 || mp.languages[0] != "go" {
		t.Errorf("unexpected languages: %v", mp.languages)
	}
}

// callsFor returns the subset of calls matching the given pass type.
func callsFor(calls []mockCall, pass providers.PassType) []mockCall {
	var out []mockCall
	for _, c := range calls {
		if c.pass == pass {
			out = append(out, c)
		}
	}
	return out
}

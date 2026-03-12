package analysis

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/brentrockwood/mri/internal/providers"
	"github.com/brentrockwood/mri/schema"
)

// mockProvider records every RunPass call for test inspection.
type mockProvider struct {
	calls  []mockCall
	errFor map[providers.PassType]error
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
	if archCalls[0].chunks[0].Path != "graph-summary" {
		t.Errorf("architecture chunk path: got %q, want %q", archCalls[0].chunks[0].Path, "graph-summary")
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

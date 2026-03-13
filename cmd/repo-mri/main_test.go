package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brentrockwood/mri/schema"
)

// TestModuleForFile_LongestPrefixMatch verifies that overlapping module paths
// resolve to the most specific (longest) match rather than the first match,
// and that path boundary checking prevents "src/pay" from matching
// files under "src/payment".
func TestModuleForFile_LongestPrefixMatch(t *testing.T) {
	modules := []schema.Module{
		{ID: "src/pay", Path: "src/pay"},
		{ID: "src/payment", Path: "src/payment"},
		{ID: "src", Path: "src"},
	}

	tests := []struct {
		file string
		want string
	}{
		{"src/payment/processor.go", "src/payment"},
		{"src/pay/util.go", "src/pay"},
		{"src/other/file.go", "src"},
		{"cmd/main.go", "unknown"},
	}

	for _, tt := range tests {
		got := moduleForFile(tt.file, modules)
		if got != tt.want {
			t.Errorf("moduleForFile(%q) = %q, want %q", tt.file, got, tt.want)
		}
	}
}

// TestModuleForFile_PathBoundary verifies that a module path is not matched
// when the prefix ends mid-component (e.g. "src/pay" must not match
// "src/payment/file.go" when "src/payment" is not a known module).
func TestModuleForFile_PathBoundary(t *testing.T) {
	modules := []schema.Module{
		{ID: "src/pay", Path: "src/pay"},
	}
	// "src/payment/file.go" starts with "src/pay" but the next char is 'm',
	// not '/', so it must not be attributed to the src/pay module.
	if got := moduleForFile("src/payment/file.go", modules); got != "unknown" {
		t.Errorf("moduleForFile(%q) = %q, want %q", "src/payment/file.go", got, "unknown")
	}
}

// TestModuleForFile_GraphSummary verifies that both the canonical and the
// AI-provider alias for the synthetic architecture chunk are attributed to
// "architecture".
func TestModuleForFile_GraphSummary(t *testing.T) {
	modules := []schema.Module{
		{ID: "internal", Path: "internal"},
	}
	for _, path := range []string{"graph-summary", "graph"} {
		if got := moduleForFile(path, modules); got != "architecture" {
			t.Errorf("moduleForFile(%q) = %q, want %q", path, got, "architecture")
		}
	}
}

// TestFindingTarget verifies TargetType and TargetID assignment for all pass types.
func TestFindingTarget(t *testing.T) {
	tests := []struct {
		typ      string
		file     string
		repoName string
		wantType string
		wantID   string
	}{
		// Architecture findings target the repository.
		{"architecture", "graph-summary", "my-repo", "repository", "my-repo"},
		// Bug and security findings target the specific file.
		{"bug", "internal/analysis/passes.go", "my-repo", "file", "internal/analysis/passes.go"},
		{"security", "cmd/main.go", "my-repo", "file", "cmd/main.go"},
	}
	for _, tt := range tests {
		gotType, gotID := findingTarget(tt.typ, tt.file, tt.repoName)
		if gotType != tt.wantType {
			t.Errorf("findingTarget(%q,...) targetType = %q, want %q", tt.typ, gotType, tt.wantType)
		}
		if gotID != tt.wantID {
			t.Errorf("findingTarget(%q,...) targetID = %q, want %q", tt.typ, gotID, tt.wantID)
		}
	}
}

// TestRunAnalyze_CancelledContext verifies that a pre-cancelled context
// surfaces as an error from runAnalyze rather than silently succeeding.
func TestRunAnalyze_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel before runAnalyze is called

	cmd := newAnalyzeCmd()
	cmd.SetContext(ctx)

	err := runAnalyze(cmd, []string{"."}, 5*time.Minute)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled in error chain, got: %v", err)
	}
}

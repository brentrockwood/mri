package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brentrockwood/mri/schema"
)

// TestModuleForFile_LongestPrefixMatch verifies that overlapping module paths
// resolve to the most specific (longest) match rather than the first match.
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

// TestModuleForFile_GraphSummary verifies that the synthetic graph-summary
// chunk is attributed to "architecture" rather than any real module.
func TestModuleForFile_GraphSummary(t *testing.T) {
	modules := []schema.Module{
		{ID: "internal", Path: "internal"},
	}
	if got := moduleForFile("graph-summary", modules); got != "architecture" {
		t.Errorf("moduleForFile(%q) = %q, want %q", "graph-summary", got, "architecture")
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

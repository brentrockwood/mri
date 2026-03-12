package providers_test

import (
	"context"
	"testing"

	"github.com/brentrockwood/mri/internal/providers"
)

func TestSelectProvider_BothKeys_PrefersAnthropic(t *testing.T) {
	t.Setenv("REPO_MRI_ANTHROPIC_KEY", "test-anthropic-key")
	t.Setenv("REPO_MRI_OPENAI_KEY", "test-openai-key")

	p, err := providers.SelectProvider(context.Background())
	if err != nil {
		t.Fatalf("SelectProvider() returned unexpected error: %v", err)
	}
	if _, ok := p.(*providers.AnthropicProvider); !ok {
		t.Fatalf("expected *AnthropicProvider, got %T", p)
	}
}

func TestSelectProvider_OnlyAnthropicKey(t *testing.T) {
	t.Setenv("REPO_MRI_ANTHROPIC_KEY", "test-anthropic-key")
	t.Setenv("REPO_MRI_OPENAI_KEY", "")

	p, err := providers.SelectProvider(context.Background())
	if err != nil {
		t.Fatalf("SelectProvider() returned unexpected error: %v", err)
	}
	if _, ok := p.(*providers.AnthropicProvider); !ok {
		t.Fatalf("expected *AnthropicProvider, got %T", p)
	}
}

func TestSelectProvider_OnlyOpenAIKey(t *testing.T) {
	t.Setenv("REPO_MRI_ANTHROPIC_KEY", "")
	t.Setenv("REPO_MRI_OPENAI_KEY", "test-openai-key")

	p, err := providers.SelectProvider(context.Background())
	if err != nil {
		t.Fatalf("SelectProvider() returned unexpected error: %v", err)
	}
	if _, ok := p.(*providers.OpenAIProvider); !ok {
		t.Fatalf("expected *OpenAIProvider, got %T", p)
	}
}

func TestSelectProvider_NoKeys_ReturnsError(t *testing.T) {
	t.Setenv("REPO_MRI_ANTHROPIC_KEY", "")
	t.Setenv("REPO_MRI_OPENAI_KEY", "")

	p, err := providers.SelectProvider(context.Background())
	if err == nil {
		t.Fatalf("SelectProvider() expected error, got provider: %T", p)
	}
	if p != nil {
		t.Errorf("SelectProvider() expected nil provider on error, got %T", p)
	}
}

// Compile-time checks: both concrete provider types must satisfy AnalysisProvider.
var (
	_ providers.AnalysisProvider = (*providers.AnthropicProvider)(nil)
	_ providers.AnalysisProvider = (*providers.OpenAIProvider)(nil)
)

// TestProviderTypes verifies that SelectProvider returns the expected concrete
// types for Anthropic and OpenAI keys.
func TestProviderTypes(t *testing.T) {
	t.Setenv("REPO_MRI_ANTHROPIC_KEY", "test-key")
	t.Setenv("REPO_MRI_OPENAI_KEY", "")

	ap, err := providers.SelectProvider(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := ap.(*providers.AnthropicProvider); !ok {
		t.Errorf("expected *AnthropicProvider, got %T", ap)
	}

	t.Setenv("REPO_MRI_ANTHROPIC_KEY", "")
	t.Setenv("REPO_MRI_OPENAI_KEY", "test-key")

	op, err := providers.SelectProvider(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := op.(*providers.OpenAIProvider); !ok {
		t.Errorf("expected *OpenAIProvider, got %T", op)
	}
}

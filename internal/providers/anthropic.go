package providers

import (
	"context"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const anthropicModel = "claude-sonnet-4-20250514"

// AnthropicProvider implements AnalysisProvider using the Anthropic API.
type AnthropicProvider struct {
	client *anthropic.Client
}

// NewAnthropicProvider constructs an AnthropicProvider authenticated with key.
func NewAnthropicProvider(key string) *AnthropicProvider {
	c := anthropic.NewClient(option.WithAPIKey(key))
	return &AnthropicProvider{client: &c}
}

// RunPass implements AnalysisProvider. It is a stub — Phase 4 fills in the
// actual API call. It returns nil, nil unconditionally.
func (p *AnthropicProvider) RunPass(_ context.Context, _ PassType, _ []FileChunk) ([]Finding, error) {
	return nil, nil
}

// Model returns the Anthropic model identifier used by this provider.
func (p *AnthropicProvider) Model() string {
	return anthropicModel
}

package providers

import (
	"context"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const openaiModel = "gpt-4o"

// OpenAIProvider implements AnalysisProvider using the OpenAI API.
type OpenAIProvider struct {
	client *openai.Client
}

// NewOpenAIProvider constructs an OpenAIProvider authenticated with key.
func NewOpenAIProvider(key string) *OpenAIProvider {
	c := openai.NewClient(option.WithAPIKey(key))
	return &OpenAIProvider{client: &c}
}

// RunPass implements AnalysisProvider. It is a stub — Phase 4 fills in the
// actual API call. It returns nil, nil unconditionally.
func (p *OpenAIProvider) RunPass(_ context.Context, _ PassType, _ []FileChunk) ([]Finding, error) {
	return nil, nil
}

// Model returns the OpenAI model identifier used by this provider.
func (p *OpenAIProvider) Model() string {
	return openaiModel
}

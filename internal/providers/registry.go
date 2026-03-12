package providers

import (
	"context"
	"errors"
	"os"
)

const (
	envAnthropicKey = "REPO_MRI_ANTHROPIC_KEY"
	envOpenAIKey    = "REPO_MRI_OPENAI_KEY"
)

// ErrNoProviderKey is returned by SelectProvider when neither
// REPO_MRI_ANTHROPIC_KEY nor REPO_MRI_OPENAI_KEY is set.
var ErrNoProviderKey = errors.New("providers: no API key set; set REPO_MRI_ANTHROPIC_KEY or REPO_MRI_OPENAI_KEY")

// SelectProvider reads environment variables to choose an AnalysisProvider.
// If REPO_MRI_ANTHROPIC_KEY is set, Anthropic is used regardless of whether
// REPO_MRI_OPENAI_KEY is also set. If only REPO_MRI_OPENAI_KEY is set, OpenAI
// is used. If neither key is set, ErrNoProviderKey is returned.
func SelectProvider(_ context.Context) (AnalysisProvider, error) {
	if key := os.Getenv(envAnthropicKey); key != "" {
		return NewAnthropicProvider(key), nil
	}
	if key := os.Getenv(envOpenAIKey); key != "" {
		return NewOpenAIProvider(key), nil
	}
	return nil, ErrNoProviderKey
}

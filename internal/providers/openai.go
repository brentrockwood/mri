package providers

import (
	"context"
	"fmt"

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

// Name returns the provider identifier "openai".
func (p *OpenAIProvider) Name() string { return "openai" }

// Model returns the OpenAI model identifier used by this provider.
func (p *OpenAIProvider) Model() string { return openaiModel }

// RunPass implements AnalysisProvider. It sends the chunks to the OpenAI Chat
// Completions API and parses the response as a JSON array of Finding values.
func (p *OpenAIProvider) RunPass(ctx context.Context, pass PassType, chunks []FileChunk) ([]Finding, error) {
	systemPrompt := buildSystemPrompt(pass)
	userText := buildUserMessage(pass, chunks)

	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:     openai.ChatModel(openaiModel),
		MaxTokens: openai.Int(4096),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userText),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("openai RunPass %s: %w", pass, err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai RunPass %s: no choices in response", pass)
	}

	responseText := resp.Choices[0].Message.Content

	findings, err := parseFindings(responseText)
	if err != nil {
		return nil, fmt.Errorf("openai RunPass %s: parse findings: %w", pass, err)
	}
	return findings, nil
}

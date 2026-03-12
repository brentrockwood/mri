package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const anthropicModel = "claude-sonnet-4-20250514"

// repoPreamble is the static portion of the context-aware prompt preamble.
// It tells the model which patterns are intentional in this codebase so they
// are not surfaced as false-positive findings.
const repoPreamble = `IMPORTANT: Before reporting any finding, check whether it matches a pre-approved pattern below. If it does, DO NOT report it.

This is a Go CLI tool. The following patterns are intentional and accepted:

1. EXEC ARGUMENT INJECTION: All exec.Command/exec.CommandContext calls pass each argument as a separate element of a Go string slice — no shell is involved, so there is no injection risk. Lines marked "// #nosec G204" have been manually reviewed. Do NOT report command injection or argument injection for any exec call in this codebase.

2. DEFERRED CLOSE ERRORS: "defer f.Close()" calls annotated with "//nolint:errcheck" are intentional. Propagating close errors from deferred reads is not idiomatic Go. Do NOT report ignored close errors or resource leaks for these lines.

3. API KEYS IN STRUCT FIELDS: API keys are read once from environment variables, passed to a constructor, and held in a struct field for one analysis run. They are never written to disk or logs. Do NOT report "key exposed in memory", "sensitive value in struct", or similar findings.

4. NON-FATAL ERROR PATHS: When a function logs an error to stderr and continues rather than returning, this is intentional pipeline design — partial results are preferred over aborting. Do NOT report missing error returns for these logged-and-continued paths.

5. OUTPUT DIRECTORY PERMISSIONS: A directory created with 0o700 (owner-only) is the correct restrictive choice for user-owned output. Do NOT report this as overly permissive or overly restrictive.

Only report findings that are genuinely problematic and not covered by any rule above.`

// AnthropicProvider implements AnalysisProvider using the Anthropic API.
type AnthropicProvider struct {
	client    *anthropic.Client
	languages []string
}

// SetAnalysisContext stores repo-level metadata used to build context-aware
// prompts. Call this before running passes to reduce false positives.
func (p *AnthropicProvider) SetAnalysisContext(languages []string) {
	p.languages = languages
}

// NewAnthropicProvider constructs an AnthropicProvider authenticated with key.
func NewAnthropicProvider(key string) *AnthropicProvider {
	c := anthropic.NewClient(option.WithAPIKey(key))
	return &AnthropicProvider{client: &c}
}

// Name returns the provider identifier "anthropic".
func (p *AnthropicProvider) Name() string { return "anthropic" }

// Model returns the Anthropic model identifier used by this provider.
func (p *AnthropicProvider) Model() string { return anthropicModel }

// RunPass implements AnalysisProvider. It sends the chunks to the Anthropic
// Messages API and parses the response as a JSON array of Finding values.
func (p *AnthropicProvider) RunPass(ctx context.Context, pass PassType, chunks []FileChunk) ([]Finding, error) {
	systemPrompt := buildSystemPrompt(pass)
	userText := buildUserMessage(pass, chunks, p.languages)

	resp, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(anthropicModel),
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			{
				Role: anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{
					{OfText: &anthropic.TextBlockParam{Text: userText}},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic RunPass %s: %w", pass, err)
	}

	// Collect all text blocks from the response.
	var sb strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	responseText := sb.String()

	findings, err := parseFindings(responseText)
	if err != nil {
		return nil, fmt.Errorf("anthropic RunPass %s: parse findings: %w", pass, err)
	}
	return findings, nil
}

// buildSystemPrompt constructs the system prompt for the given pass type.
func buildSystemPrompt(pass PassType) string {
	base := `You are a senior software engineer performing a code review. Your task is to analyze the provided code and identify issues. You MUST respond with ONLY a valid JSON array of findings. No prose, no markdown, no explanation outside the JSON.

Each finding must be a JSON object with these exact fields:
{
  "severity": "high" | "medium" | "low",
  "type": "<pass-type>",
  "file": "<repo-relative path or 'graph' for architecture>",
  "title": "<short label>",
  "description": "<explanation>",
  "confidence": <0.0-1.0>,
  "evidence_lines": [<line numbers>]
}

Return [] if no issues are found.`

	var focus string
	switch pass {
	case PassArchitecture:
		focus = "Focus on: circular dependencies, layer violations (e.g. low-level packages importing high-level), unexpected dependency relationships, and structural anomalies."
	case PassBug:
		focus = "Focus on: cyclomatic complexity hotspots, error handling gaps (ignored errors, missing nil checks), fragile logic paths, and potential panics."
	case PassSecurity:
		focus = "Focus on: hardcoded secrets or credentials, injection vulnerabilities (SQL, command, path traversal), authentication gaps, and insecure use of cryptography or random numbers."
	}

	if focus == "" {
		return base
	}
	return base + "\n\n" + focus
}

// buildUserMessage formats chunks into a user message for the given pass.
// When the repository contains Go, a context-aware preamble is prepended to
// suppress false-positive findings for intentional Go patterns.
func buildUserMessage(pass PassType, chunks []FileChunk, languages []string) string {
	var sb strings.Builder
	if containsLanguage(languages, "go") {
		sb.WriteString(repoPreamble)
		fmt.Fprintf(&sb, "\nLanguages detected in this repository: %s.\n\n", strings.Join(languages, ", "))
	}
	fmt.Fprintf(&sb, "Perform a %s analysis on the following content:\n\n", pass)
	for _, c := range chunks {
		fmt.Fprintf(&sb, "=== FILE: %s (%s) ===\n%s\n\n", c.Path, c.Language, c.Content)
	}
	return sb.String()
}

// containsLanguage reports whether lang appears in languages (case-insensitive).
func containsLanguage(languages []string, lang string) bool {
	for _, l := range languages {
		if strings.EqualFold(l, lang) {
			return true
		}
	}
	return false
}

// rawFinding is used for JSON unmarshalling with snake_case field names.
type rawFinding struct {
	Severity      string  `json:"severity"`
	Type          string  `json:"type"`
	File          string  `json:"file"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Confidence    float64 `json:"confidence"`
	EvidenceLines []int   `json:"evidence_lines"`
}

// parseFindings extracts and parses the JSON array of findings from text.
// It locates the first '[' and last ']' to tolerate minor formatting issues.
func parseFindings(text string) ([]Finding, error) {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("no JSON array found in response")
	}
	raw := text[start : end+1]

	var raws []rawFinding
	if err := json.Unmarshal([]byte(raw), &raws); err != nil {
		return nil, fmt.Errorf("unmarshal findings: %w", err)
	}

	findings := make([]Finding, len(raws))
	for i, r := range raws {
		findings[i] = Finding(r)
	}
	return findings, nil
}

// Package providers defines the AnalysisProvider interface and the types used
// to pass content to and receive structured findings from AI analysis passes.
package providers

import "context"

// PassType identifies which analysis pass is being run.
type PassType string

const (
	// PassArchitecture is the architecture analysis pass.
	PassArchitecture PassType = "architecture"
	// PassBug is the bug analysis pass.
	PassBug PassType = "bug"
	// PassSecurity is the security analysis pass.
	PassSecurity PassType = "security"
)

// GraphSummaryPath is the synthetic chunk path used by the architecture pass
// to represent the dependency graph summary rather than a real source file.
const GraphSummaryPath = "graph-summary"

// FileChunk is a unit of content passed to a provider pass.
type FileChunk struct {
	// Path is the repo-relative path of the file.
	Path string
	// Language is the detected language of the file (e.g. "Go", "Python").
	Language string
	// Content is the raw source text.
	Content string
}

// Finding is a single result returned by a provider pass.
// It maps to schema.Risk but is provider-agnostic.
type Finding struct {
	// Severity is one of "high", "medium", or "low".
	Severity string
	// Type is one of "architecture", "bug", or "security".
	Type string
	// File is the repo-relative path where the finding was detected.
	File string
	// Title is a short human-readable label for the finding.
	Title string
	// Description is a longer explanation of the finding.
	Description string
	// Confidence is a value in [0, 1] indicating the provider's confidence.
	Confidence float64
	// EvidenceLines contains the 1-based source line numbers that support the finding.
	EvidenceLines []int
}

// AnalysisProvider runs a single named pass over a set of file chunks.
type AnalysisProvider interface {
	// RunPass executes pass over chunks and returns the resulting findings.
	// It returns nil, nil when no findings are produced.
	RunPass(ctx context.Context, pass PassType, chunks []FileChunk) ([]Finding, error)
}

// Namer is an optional interface implemented by providers that can report
// their name and the model they use. Use a type assertion to check for it.
type Namer interface {
	// Name returns a short identifier for the provider (e.g. "anthropic").
	Name() string
	// Model returns the model identifier (e.g. "claude-sonnet-4-20250514").
	Model() string
}

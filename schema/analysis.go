// Package schema defines the canonical data types for the repo-mri analysis output.
package schema

import "time"

// SchemaVersion is the current version of the analysis.json schema.
const SchemaVersion = "1.1"

// CLIVersion is the current version of the CLI.
const CLIVersion = "0.1.0"

// Analysis is the top-level output artifact written to .repo-mri/analysis.json.
type Analysis struct {
	Meta         Meta         `json:"meta"`
	Repo         Repo         `json:"repo"`
	Modules      []Module     `json:"modules"`
	Dependencies []Dependency `json:"dependencies"`
	Risks        []Risk       `json:"risks"`
	Files        []File       `json:"files"`
	Subsystems   []Subsystem  `json:"subsystems,omitempty"`
}

// Meta holds metadata about the analysis run.
type Meta struct {
	SchemaVersion      string   `json:"schema_version"`
	CLIVersion         string   `json:"cli_version"`
	ModelUsed          string   `json:"model_used,omitempty"`
	Provider           string   `json:"provider,omitempty"`
	AnalysisDurationMS int64    `json:"analysis_duration_ms"`
	SkippedPasses      []string `json:"skipped_passes,omitempty"`
	// MaxChainDepth is the length of the longest module dependency chain. Set by Phase 2.
	MaxChainDepth int `json:"max_chain_depth,omitempty"`
}

// Repo holds repository-level summary data.
type Repo struct {
	Name         string    `json:"name"`
	Languages    []string  `json:"languages"`
	FileCount    int       `json:"file_count"`
	ModuleCount  int       `json:"module_count"`
	AnalysisTime time.Time `json:"analysis_time"`
}

// Module represents a logical grouping of files (typically a package/directory).
type Module struct {
	ID              string  `json:"id"`
	Path            string  `json:"path"`
	Language        string  `json:"language"`
	FileCount       int     `json:"file_count"`
	RiskScore       float64 `json:"risk_score"`
	ComplexityScore float64 `json:"complexity_score"`
	// ImportCount is the number of other modules that import this module. Set by Phase 2.
	ImportCount int `json:"import_count,omitempty"`
}

// Dependency represents a directed import relationship between modules.
type Dependency struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

// Risk represents a single finding from an analysis pass.
type Risk struct {
	ID            string  `json:"id"`
	Severity      string  `json:"severity"`
	Type          string  `json:"type"`
	Pass          string  `json:"pass"`
	Module        string  `json:"module"`
	File          string  `json:"file"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Confidence    float64 `json:"confidence"`
	EvidenceLines []int   `json:"evidence_lines,omitempty"`
}

// File holds per-file metrics.
type File struct {
	Path     string `json:"path"`
	Module   string `json:"module"`
	Language string `json:"language"`
	Lines    int    `json:"lines"`
	// Size is the file size in bytes. Set by Phase 2.
	Size       int64   `json:"size,omitempty"`
	Complexity float64 `json:"complexity"`
	RiskScore  float64 `json:"risk_score"`
}

// Subsystem groups related modules.
type Subsystem struct {
	ID      string   `json:"id"`
	Modules []string `json:"modules"`
}

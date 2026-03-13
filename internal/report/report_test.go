package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brentrockwood/mri/schema"
)

// minimalAnalysis returns a schema.Analysis with minimal required fields populated.
func minimalAnalysis() schema.Analysis {
	return schema.Analysis{
		Meta: schema.Meta{
			CLIVersion:         "0.1.0",
			AnalysisDurationMS: 42,
		},
		Repo: schema.Repo{
			Name:         "test-repo",
			Languages:    []string{"Go", "Python"},
			FileCount:    10,
			ModuleCount:  3,
			AnalysisTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Modules: []schema.Module{
			{ID: "pkg/alpha", RiskScore: 0.8, ComplexityScore: 0.6, FileCount: 3},
			{ID: "pkg/beta", RiskScore: 0.4, ComplexityScore: 0.3, FileCount: 2},
		},
	}
}

// TestHealthScore verifies health score calculation across different mean risk values.
func TestHealthScore(t *testing.T) {
	tests := []struct {
		name    string
		modules []schema.Module
		want    int
	}{
		{
			name:    "no modules returns 100",
			modules: nil,
			want:    100,
		},
		{
			name: "all zero risk",
			modules: []schema.Module{
				{RiskScore: 0.0},
				{RiskScore: 0.0},
			},
			want: 100,
		},
		{
			name: "all max risk",
			modules: []schema.Module{
				{RiskScore: 1.0},
				{RiskScore: 1.0},
			},
			want: 0,
		},
		{
			name: "mean risk 0.5",
			modules: []schema.Module{
				{RiskScore: 0.5},
			},
			want: 50,
		},
		{
			name: "mean risk 0.1 → score 90",
			modules: []schema.Module{
				{RiskScore: 0.1},
			},
			want: 90,
		},
		{
			name: "mixed modules average",
			modules: []schema.Module{
				{RiskScore: 0.2},
				{RiskScore: 0.8},
			},
			want: 50,
		},
		{
			name: "rounding up",
			modules: []schema.Module{
				{RiskScore: 0.005}, // mean=0.005, 0.995*100=99.5 → rounds to 100
			},
			want: 100,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := HealthScore(tc.modules)
			if got != tc.want {
				t.Errorf("HealthScore() = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestScoreBand verifies the correct narrative is returned for each band.
func TestScoreBand(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{100, "Excellent"},
		{90, "Excellent"},
		{89, "Good"},
		{70, "Good"},
		{69, "Fair"},
		{50, "Fair"},
		{49, "Poor"},
		{0, "Poor"},
	}

	for _, tc := range tests {
		t.Run("score "+strings.ToLower(tc.want), func(t *testing.T) {
			got := ScoreBand(tc.score)
			if !strings.HasPrefix(got, tc.want) {
				t.Errorf("ScoreBand(%d) = %q, want prefix %q", tc.score, got, tc.want)
			}
		})
	}
}

// TestReportContainsExpectedSections checks that the report includes key sections.
func TestReportContainsExpectedSections(t *testing.T) {
	a := minimalAnalysis()
	content := buildReport(&a)

	sections := []string{
		"# Repo MRI Report: test-repo",
		"## Repository Overview",
		"## Health Score:",
		"## Top Modules by Risk",
		"static analysis only",
		"Go, Python",
	}

	for _, s := range sections {
		if !strings.Contains(content, s) {
			t.Errorf("report missing expected content %q", s)
		}
	}
}

// TestReportModuleTable verifies module data appears in the top-modules table.
func TestReportModuleTable(t *testing.T) {
	a := minimalAnalysis()
	content := buildReport(&a)

	if !strings.Contains(content, "pkg/alpha") {
		t.Error("report missing module pkg/alpha in table")
	}
	if !strings.Contains(content, "pkg/beta") {
		t.Error("report missing module pkg/beta in table")
	}
}

// TestHighSeveritySectionSkippedWhenNone verifies the high-severity section is
// omitted when there are no high-severity findings.
func TestHighSeveritySectionSkippedWhenNone(t *testing.T) {
	a := minimalAnalysis()
	a.Risks = []schema.Risk{
		{Severity: "low", Type: "style", TargetType: "file", Module: "pkg/alpha", File: "a.go", Title: "low finding"},
	}
	content := buildReport(&a)

	if strings.Contains(content, "## High-Severity Findings") {
		t.Error("report should not contain high-severity section when no high findings")
	}
}

// TestSecuritySectionSkippedWhenNone verifies the security section is omitted
// when there are no security-type findings.
func TestSecuritySectionSkippedWhenNone(t *testing.T) {
	a := minimalAnalysis()
	a.Risks = []schema.Risk{
		{Severity: "high", Type: "complexity", TargetType: "file", Module: "pkg/alpha", File: "a.go",
			Title: "complex fn", Confidence: 0.9},
	}
	content := buildReport(&a)

	if strings.Contains(content, "## Security Findings") {
		t.Error("report should not contain security section when no security findings")
	}
}

// TestBothSectionsPresentWhenApplicable verifies both finding sections appear
// when the analysis contains both high-severity and security findings.
func TestBothSectionsPresentWhenApplicable(t *testing.T) {
	a := minimalAnalysis()
	a.Risks = []schema.Risk{
		{
			Severity: "high", Type: "security", TargetType: "file", Module: "pkg/alpha", File: "auth.go",
			Title: "SQL injection", Description: "User input unsanitized.",
			Confidence: 0.95,
		},
	}
	content := buildReport(&a)

	if !strings.Contains(content, "## High-Severity Findings") {
		t.Error("report missing High-Severity Findings section")
	}
	if !strings.Contains(content, "## Security Findings") {
		t.Error("report missing Security Findings section")
	}
}

// TestHighSeveritySectionGroupsByModule ensures high findings are grouped under
// their module heading.
func TestHighSeveritySectionGroupsByModule(t *testing.T) {
	a := minimalAnalysis()
	a.Risks = []schema.Risk{
		{
			Severity: "high", Type: "complexity", TargetType: "file", Module: "pkg/alpha", File: "a.go",
			Title: "High complexity", Description: "Too complex.", Confidence: 0.8,
		},
	}
	content := buildReport(&a)

	if !strings.Contains(content, "### pkg/alpha") {
		t.Error("report missing module heading in high-severity section")
	}
	if !strings.Contains(content, "High complexity") {
		t.Error("report missing finding title in high-severity section")
	}
}

// TestProviderLine verifies the provider line is rendered correctly when set.
func TestProviderLine(t *testing.T) {
	a := minimalAnalysis()
	a.Meta.Provider = "anthropic"
	a.Meta.ModelUsed = "claude-3-opus"
	content := buildReport(&a)

	if !strings.Contains(content, "anthropic (claude-3-opus)") {
		t.Errorf("report missing provider line, got:\n%s", content[:200])
	}
}

// TestGenerateWritesToCorrectPath verifies Generate writes report.md and that
// the file has correct permissions (0600).
func TestGenerateWritesToCorrectPath(t *testing.T) {
	dir := t.TempDir()
	a := minimalAnalysis()

	if err := Generate(&a, dir); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	outPath := filepath.Join(dir, "report.md")
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("report.md not found: %v", err)
	}

	// Check permissions: mode bits should be 0600.
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("report.md permissions = %04o, want 0600", perm)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read report.md: %v", err)
	}
	if !strings.Contains(string(data), "# Repo MRI Report:") {
		t.Error("report.md missing expected header")
	}
}

// TestArchitectureSectionAppearsForRepositoryTarget verifies the Architecture
// Findings section is rendered when a risk has TargetType "repository".
func TestArchitectureSectionAppearsForRepositoryTarget(t *testing.T) {
	a := minimalAnalysis()
	a.Risks = []schema.Risk{
		{
			Severity:    "high",
			Type:        "architecture",
			TargetType:  "repository",
			TargetID:    "test-repo",
			Module:      "architecture",
			File:        "graph-summary",
			Title:       "Circular dependency",
			Description: "pkg/alpha imports pkg/beta and vice versa.",
			Confidence:  0.9,
		},
	}
	content := buildReport(&a)

	if !strings.Contains(content, "## Architecture Findings") {
		t.Error("report missing Architecture Findings section for repository-target risk")
	}
	if !strings.Contains(content, "Circular dependency") {
		t.Error("report missing architecture finding title")
	}
}

// TestArchitectureSectionSkippedWhenNone verifies the Architecture Findings
// section is omitted when no risks have TargetType "repository".
func TestArchitectureSectionSkippedWhenNone(t *testing.T) {
	a := minimalAnalysis()
	a.Risks = []schema.Risk{
		{Severity: "high", Type: "bug", TargetType: "file", Module: "pkg/alpha", File: "a.go",
			Title: "nil deref", Confidence: 0.8},
	}
	content := buildReport(&a)

	if strings.Contains(content, "## Architecture Findings") {
		t.Error("report should not contain Architecture Findings section when no repository-target risks")
	}
}

// TestTargetTypeRoutingAllThree verifies the three TargetType values each route
// to the correct report section.
func TestTargetTypeRoutingAllThree(t *testing.T) {
	a := minimalAnalysis()
	a.Risks = []schema.Risk{
		// repository-target: architecture section
		{
			Severity: "medium", Type: "architecture", TargetType: "repository", TargetID: "test-repo",
			Module: "architecture", File: "graph-summary",
			Title: "Layer violation", Description: "cmd imports internal directly.", Confidence: 0.7,
		},
		// file-target, high-severity: high section
		{
			Severity: "high", Type: "bug", TargetType: "file", TargetID: "pkg/alpha/a.go",
			Module: "pkg/alpha", File: "pkg/alpha/a.go",
			Title: "Nil pointer", Description: "ptr dereferenced before check.", Confidence: 0.85,
		},
		// file-target, security: security section
		{
			Severity: "medium", Type: "security", TargetType: "file", TargetID: "pkg/alpha/auth.go",
			Module: "pkg/alpha", File: "pkg/alpha/auth.go",
			Title: "Hardcoded secret", Description: "API key in source.", Confidence: 0.95,
		},
	}
	content := buildReport(&a)

	tests := []struct {
		section string
		title   string
	}{
		{"## Architecture Findings", "Layer violation"},
		{"## High-Severity Findings", "Nil pointer"},
		{"## Security Findings", "Hardcoded secret"},
	}
	for _, tc := range tests {
		if !strings.Contains(content, tc.section) {
			t.Errorf("report missing section %q", tc.section)
		}
		if !strings.Contains(content, tc.title) {
			t.Errorf("report missing finding %q", tc.title)
		}
	}

	// Architecture finding must NOT appear in the file-level high-severity section.
	// Locate high section and verify "Layer violation" is not in it.
	highIdx := strings.Index(content, "## High-Severity Findings")
	secIdx := strings.Index(content, "## Security Findings")
	if highIdx != -1 && secIdx != -1 {
		highSection := content[highIdx:secIdx]
		if strings.Contains(highSection, "Layer violation") {
			t.Error("architecture finding should not appear in High-Severity Findings section")
		}
	}
}

// TestTopModulesSortOrder verifies the tie-breaking sort: risk desc, complexity desc,
// file count desc, module name asc.
func TestTopModulesSortOrder(t *testing.T) {
	a := minimalAnalysis()
	a.Modules = []schema.Module{
		{ID: "pkg/z", RiskScore: 0.0, ComplexityScore: 0.5, FileCount: 3},
		{ID: "pkg/a", RiskScore: 0.0, ComplexityScore: 0.5, FileCount: 3},
		{ID: "pkg/b", RiskScore: 0.0, ComplexityScore: 0.8, FileCount: 1},
		{ID: "pkg/c", RiskScore: 0.5, ComplexityScore: 0.1, FileCount: 1},
	}
	content := buildReport(&a)

	// pkg/c has the highest risk score and must appear first (rank 1)
	if !strings.Contains(content, "| 1 | pkg/c |") {
		t.Error("expected pkg/c (highest risk) to be rank 1")
	}
	// pkg/b has higher complexity than pkg/a and pkg/z — must be rank 2
	if !strings.Contains(content, "| 2 | pkg/b |") {
		t.Error("expected pkg/b (highest complexity among tied risk) to be rank 2")
	}
	// pkg/a < pkg/z alphabetically — must appear before pkg/z
	aIdx := strings.Index(content, "pkg/a")
	zIdx := strings.Index(content, "pkg/z")
	if aIdx == -1 || zIdx == -1 || aIdx > zIdx {
		t.Error("expected pkg/a (alphabetically first) to appear before pkg/z in table")
	}
	// Footnote explaining sort order must be present
	if !strings.Contains(content, "risk score (desc)") {
		t.Error("expected sort-order footnote in report")
	}
}

// TestTopModulesLimitedToTen verifies that only up to 10 modules appear in the table.
func TestTopModulesLimitedToTen(t *testing.T) {
	a := minimalAnalysis()
	a.Modules = nil
	for i := 0; i < 15; i++ {
		a.Modules = append(a.Modules, schema.Module{
			ID:        "mod" + strings.Repeat("x", i),
			RiskScore: float64(15-i) / 15.0,
			FileCount: 1,
		})
	}
	content := buildReport(&a)

	// The 11th module (index 10) should NOT appear.
	if strings.Contains(content, "| 11 |") {
		t.Error("report table should not have rank 11 entry")
	}
	if !strings.Contains(content, "| 10 |") {
		t.Error("report table should have rank 10 entry")
	}
}

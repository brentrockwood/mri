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
		{Severity: "low", Type: "style", Module: "pkg/alpha", File: "a.go", Title: "low finding"},
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
		{Severity: "high", Type: "complexity", Module: "pkg/alpha", File: "a.go",
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
			Severity: "high", Type: "security", Module: "pkg/alpha", File: "auth.go",
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
			Severity: "high", Type: "complexity", Module: "pkg/alpha", File: "a.go",
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

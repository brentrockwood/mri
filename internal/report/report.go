// Package report generates a human-readable Markdown report from a completed
// repo-mri analysis.
package report

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/brentrockwood/mri/schema"
)

//go:embed static/report.html
var reportHTML []byte

// GenerateHTML writes report.html to outDir. It injects the analysis JSON into
// the embedded HTML template as window.__MRI_DATA__ so the report can be opened
// directly from the file system without a web server.
// outDir must already exist. The report file is written with mode 0600.
func GenerateHTML(a *schema.Analysis, outDir string) error {
	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("report: marshal analysis for HTML: %w", err)
	}

	// Inject <script>window.__MRI_DATA__ = <json>;</script> just before </head>.
	injection := append([]byte("<script>window.__MRI_DATA__ = "), data...)
	injection = append(injection, []byte(";</script>")...)

	html := bytes.Replace(reportHTML, []byte("</head>"), append(injection, []byte("</head>")...), 1)

	outPath := filepath.Join(outDir, "report.html")
	if err := os.WriteFile(outPath, html, 0o600); err != nil {
		return fmt.Errorf("report: write %s: %w", outPath, err)
	}
	return nil
}

// Generate writes report.md to outDir based on the completed analysis a.
// outDir must already exist. The report file is written with mode 0600.
func Generate(a *schema.Analysis, outDir string) error {
	content := buildReport(a)
	outPath := filepath.Join(outDir, "report.md")
	if err := os.WriteFile(outPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("report: write %s: %w", outPath, err)
	}
	return nil
}

// HealthScore computes the overall health score (0–100) from the mean risk
// across all modules. Returns 100 if there are no modules.
func HealthScore(modules []schema.Module) int {
	if len(modules) == 0 {
		return 100
	}
	var sum float64
	for _, m := range modules {
		sum += m.RiskScore
	}
	mean := sum / float64(len(modules))
	return int(math.Round((1.0 - mean) * 100))
}

// ScoreBand returns a narrative description for a given health score.
func ScoreBand(score int) string {
	switch {
	case score >= 90:
		return "Excellent — no significant issues detected."
	case score >= 70:
		return "Good — minor issues present, no critical concerns."
	case score >= 50:
		return "Fair — moderate issues require attention."
	default:
		return "Poor — significant issues detected, immediate review recommended."
	}
}

// buildReport assembles the full Markdown report string.
func buildReport(a *schema.Analysis) string {
	var sb strings.Builder

	// Header
	fmt.Fprintf(&sb, "# Repo MRI Report: %s\n\n", a.Repo.Name)
	fmt.Fprintf(&sb, "**Generated:** %s\n", a.Repo.AnalysisTime.Format(time.RFC3339))
	fmt.Fprintf(&sb, "**CLI version:** %s\n", a.Meta.CLIVersion)

	if a.Meta.Provider != "" {
		fmt.Fprintf(&sb, "**Provider:** %s (%s)\n", a.Meta.Provider, a.Meta.ModelUsed)
	} else {
		fmt.Fprintf(&sb, "**Provider:** static analysis only\n")
	}

	fmt.Fprintf(&sb, "\n---\n\n")

	// Repository Overview table
	fmt.Fprintf(&sb, "## Repository Overview\n\n")
	fmt.Fprintf(&sb, "| Field | Value |\n")
	fmt.Fprintf(&sb, "|---|---|\n")
	fmt.Fprintf(&sb, "| Name | %s |\n", a.Repo.Name)
	fmt.Fprintf(&sb, "| Languages | %s |\n", strings.Join(a.Repo.Languages, ", "))
	fmt.Fprintf(&sb, "| Files | %d |\n", a.Repo.FileCount)
	fmt.Fprintf(&sb, "| Modules | %d |\n", a.Repo.ModuleCount)
	fmt.Fprintf(&sb, "| Analysis time | %dms |\n", a.Meta.AnalysisDurationMS)
	fmt.Fprintf(&sb, "\n")

	// Health Score
	score := HealthScore(a.Modules)
	fmt.Fprintf(&sb, "## Health Score: %d/100\n\n", score)
	fmt.Fprintf(&sb, "> %s\n\n", ScoreBand(score))

	// Top Modules by Risk — sorted: risk desc, complexity desc, file count desc, name asc
	fmt.Fprintf(&sb, "## Top Modules by Risk\n\n")
	fmt.Fprintf(&sb, "| Rank | Module | Risk Score | Complexity | Files |\n")
	fmt.Fprintf(&sb, "|---|---|---|---|---|\n")
	top := make([]schema.Module, len(a.Modules))
	copy(top, a.Modules)
	sort.Slice(top, func(i, j int) bool {
		mi, mj := top[i], top[j]
		if mi.RiskScore != mj.RiskScore {
			return mi.RiskScore > mj.RiskScore
		}
		if mi.ComplexityScore != mj.ComplexityScore {
			return mi.ComplexityScore > mj.ComplexityScore
		}
		if mi.FileCount != mj.FileCount {
			return mi.FileCount > mj.FileCount
		}
		return mi.ID < mj.ID
	})
	if len(top) > 10 {
		top = top[:10]
	}
	for i, m := range top {
		fmt.Fprintf(&sb, "| %d | %s | %.2f | %.2f | %d |\n",
			i+1, m.ID, m.RiskScore, m.ComplexityScore, m.FileCount)
	}
	fmt.Fprintf(&sb, "\n_Ranked by risk score (desc), complexity (desc), file count (desc), module name (asc)._\n\n")

	// Architecture Findings (repository-level, routed by TargetType)
	archFindings := filterRisks(a.Risks, func(r schema.Risk) bool {
		return r.TargetType == "repository"
	})
	if len(archFindings) > 0 {
		fmt.Fprintf(&sb, "## Architecture Findings\n\n")
		for _, r := range archFindings {
			fmt.Fprintf(&sb, "- **[%s]** %s\n", r.Severity, r.Title)
			fmt.Fprintf(&sb, "  %s\n", r.Description)
			fmt.Fprintf(&sb, "  *Confidence: %.0f%%*\n\n", r.Confidence*100)
		}
	}

	// High-Severity Findings (file-level)
	highByModule := groupRisksByModule(a.Risks, func(r schema.Risk) bool {
		return r.Severity == "high" && r.TargetType != "repository"
	})
	if len(highByModule) > 0 {
		fmt.Fprintf(&sb, "## High-Severity Findings\n\n")
		for _, moduleID := range sortedKeys(highByModule) {
			fmt.Fprintf(&sb, "### %s\n\n", moduleID)
			for _, r := range highByModule[moduleID] {
				fmt.Fprintf(&sb, "- **[%s]** `%s` — %s\n", r.Type, r.File, r.Title)
				fmt.Fprintf(&sb, "  %s\n", r.Description)
				fmt.Fprintf(&sb, "  *Confidence: %.0f%%*\n\n", r.Confidence*100)
			}
		}
	}

	// Security Findings (file-level, routed by TargetType)
	secByModule := groupRisksByModule(a.Risks, func(r schema.Risk) bool {
		return r.TargetType == "file" && r.Type == "security"
	})
	if len(secByModule) > 0 {
		fmt.Fprintf(&sb, "## Security Findings\n\n")
		for _, moduleID := range sortedKeys(secByModule) {
			fmt.Fprintf(&sb, "### %s\n\n", moduleID)
			for _, r := range secByModule[moduleID] {
				fmt.Fprintf(&sb, "- **[%s]** `%s` — %s\n", r.Severity, r.File, r.Title)
				fmt.Fprintf(&sb, "  %s\n", r.Description)
				fmt.Fprintf(&sb, "  *Confidence: %.0f%%*\n\n", r.Confidence*100)
			}
		}
	}

	fmt.Fprintf(&sb, "---\n")
	fmt.Fprintf(&sb, "*Generated by repo-mri %s*\n", a.Meta.CLIVersion)

	return sb.String()
}

// filterRisks returns risks that satisfy predicate, in original order.
func filterRisks(risks []schema.Risk, predicate func(schema.Risk) bool) []schema.Risk {
	var out []schema.Risk
	for _, r := range risks {
		if predicate(r) {
			out = append(out, r)
		}
	}
	return out
}

// groupRisksByModule groups risks that satisfy predicate by their Module field.
// The returned map is keyed by module ID.
func groupRisksByModule(risks []schema.Risk, predicate func(schema.Risk) bool) map[string][]schema.Risk {
	result := make(map[string][]schema.Risk)
	for _, r := range risks {
		if predicate(r) {
			result[r.Module] = append(result[r.Module], r)
		}
	}
	return result
}

// sortedKeys returns the keys of m in ascending alphabetical order.
func sortedKeys(m map[string][]schema.Risk) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort — small N.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

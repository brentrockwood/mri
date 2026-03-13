package depaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/brentrockwood/mri/schema"
)

// auditJS runs npm audit for each JS/TS project root found during ingestion.
// If npm is not on PATH the "npm-audit" pass is skipped (non-fatal).
func auditJS(ctx context.Context, root string, jsProjectRoots []string) ([]schema.Risk, []string) {
	if len(jsProjectRoots) == 0 {
		return nil, nil
	}

	npmPath, err := exec.LookPath("npm")
	if err != nil {
		return nil, []string{"npm-audit"}
	}

	var risks []schema.Risk
	for _, proj := range jsProjectRoots {
		projAbs := filepath.Join(root, filepath.FromSlash(proj))
		pkgJSONPath := proj + "/package.json"
		r := runNPMAudit(ctx, npmPath, projAbs, proj, pkgJSONPath)
		risks = append(risks, r...)
	}
	return risks, nil
}

// runNPMAudit executes npm audit --json in dir and parses the output into risks.
// The npm process exits with a non-zero code when vulnerabilities are found;
// we ignore the exit code and parse whatever was written to stdout.
func runNPMAudit(ctx context.Context, npmPath, dir, moduleID, pkgJSONPath string) []schema.Risk {
	cmd := exec.CommandContext(ctx, npmPath, "audit", "--json") // #nosec G204 -- npmPath from LookPath
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	// Intentionally ignoring exit code — non-zero means vulnerabilities found.
	_ = cmd.Run()

	if out.Len() == 0 {
		return nil
	}
	return parseNPMAuditJSON(out.Bytes(), moduleID, pkgJSONPath)
}

// npmAuditV2 is the npm v7+ audit report shape (auditReportVersion == 2).
type npmAuditV2 struct {
	AuditReportVersion int                       `json:"auditReportVersion"`
	Vulnerabilities    map[string]npmVulnV2Entry `json:"vulnerabilities"`
}

// npmVulnV2Entry is a single entry in the npm v7+ vulnerabilities map.
type npmVulnV2Entry struct {
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Range    string `json:"range"`
}

// npmAuditV1 is the npm v6 audit report shape (advisories map).
type npmAuditV1 struct {
	Advisories map[string]npmAdvisoryV1 `json:"advisories"`
}

// npmAdvisoryV1 is a single entry in the npm v6 advisories map.
type npmAdvisoryV1 struct {
	ModuleName string `json:"module_name"`
	Title      string `json:"title"`
	Overview   string `json:"overview"`
	Severity   string `json:"severity"`
	URL        string `json:"url"`
}

// parseNPMAuditJSON detects whether data is a v7+ or v6 npm audit report and
// converts vulnerabilities to schema.Risk entries. Unrecognised formats are
// silently ignored so that future npm output changes are non-fatal.
func parseNPMAuditJSON(data []byte, moduleID, pkgJSONPath string) []schema.Risk {
	// Probe for v2 format first: it has auditReportVersion == 2.
	var probe struct {
		AuditReportVersion int `json:"auditReportVersion"`
	}
	if err := json.Unmarshal(data, &probe); err == nil && probe.AuditReportVersion == 2 {
		var v2 npmAuditV2
		if err := json.Unmarshal(data, &v2); err == nil {
			return v2ToRisks(v2, moduleID, pkgJSONPath)
		}
	}

	// Fall back to v1 (advisories map).
	var v1 npmAuditV1
	if err := json.Unmarshal(data, &v1); err == nil && len(v1.Advisories) > 0 {
		return v1ToRisks(v1, moduleID, pkgJSONPath)
	}
	return nil
}

// mapNPMSeverity converts npm audit severity strings to schema severity levels.
func mapNPMSeverity(s string) string {
	switch s {
	case "critical", "high":
		return "high"
	case "moderate":
		return "medium"
	default:
		return "low"
	}
}

// v2ToRisks converts an npm v7+ audit report to schema.Risk entries.
func v2ToRisks(v npmAuditV2, moduleID, pkgJSONPath string) []schema.Risk {
	risks := make([]schema.Risk, 0, len(v.Vulnerabilities))
	for _, vuln := range v.Vulnerabilities {
		title := fmt.Sprintf("Vulnerable dependency: %s", vuln.Name)
		desc := fmt.Sprintf("Package %s has a known vulnerability.", vuln.Name)
		if vuln.Range != "" {
			desc += fmt.Sprintf(" Affected range: %s.", vuln.Range)
		}
		risks = append(risks, schema.Risk{
			Severity:    mapNPMSeverity(vuln.Severity),
			Type:        "dep-vuln",
			Pass:        "npm-audit",
			Module:      moduleID,
			File:        pkgJSONPath,
			Title:       title,
			Description: desc,
			Confidence:  1.0,
			TargetType:  "file",
			TargetID:    pkgJSONPath,
		})
	}
	return risks
}

// v1ToRisks converts an npm v6 audit report to schema.Risk entries.
func v1ToRisks(v npmAuditV1, moduleID, pkgJSONPath string) []schema.Risk {
	risks := make([]schema.Risk, 0, len(v.Advisories))
	for _, adv := range v.Advisories {
		desc := adv.Overview
		if adv.URL != "" {
			desc += " See: " + adv.URL
		}
		risks = append(risks, schema.Risk{
			Severity:    mapNPMSeverity(adv.Severity),
			Type:        "dep-vuln",
			Pass:        "npm-audit",
			Module:      moduleID,
			File:        pkgJSONPath,
			Title:       adv.Title,
			Description: desc,
			Confidence:  1.0,
			TargetType:  "file",
			TargetID:    pkgJSONPath,
		})
	}
	return risks
}

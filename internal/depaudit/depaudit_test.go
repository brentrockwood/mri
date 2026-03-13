package depaudit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- mapNPMSeverity ---

func TestMapNPMSeverity(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"critical", "high"},
		{"high", "high"},
		{"moderate", "medium"},
		{"low", "low"},
		{"info", "low"},
		{"", "low"},
		{"unknown", "low"},
	}
	for _, tt := range tests {
		if got := mapNPMSeverity(tt.input); got != tt.want {
			t.Errorf("mapNPMSeverity(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- parseNPMAuditJSON ---

func TestParseNPMAuditJSON_V2(t *testing.T) {
	data := []byte(`{
		"auditReportVersion": 2,
		"vulnerabilities": {
			"lodash": {
				"name": "lodash",
				"severity": "high",
				"range": "<4.17.21"
			},
			"minimist": {
				"name": "minimist",
				"severity": "moderate",
				"range": "<1.2.6"
			}
		}
	}`)

	risks := parseNPMAuditJSON(data, "ui", "ui/package.json")
	if len(risks) != 2 {
		t.Fatalf("got %d risks, want 2", len(risks))
	}

	sev := map[string]string{}
	for _, r := range risks {
		if r.Type != "dep-vuln" {
			t.Errorf("risk.Type = %q, want \"dep-vuln\"", r.Type)
		}
		if r.Pass != "npm-audit" {
			t.Errorf("risk.Pass = %q, want \"npm-audit\"", r.Pass)
		}
		if r.Module != "ui" {
			t.Errorf("risk.Module = %q, want \"ui\"", r.Module)
		}
		if r.File != "ui/package.json" {
			t.Errorf("risk.File = %q, want \"ui/package.json\"", r.File)
		}
		if r.Confidence != 1.0 {
			t.Errorf("risk.Confidence = %v, want 1.0", r.Confidence)
		}
		sev[r.Title] = r.Severity
	}
	if sev["Vulnerable dependency: lodash"] != "high" {
		t.Errorf("lodash severity = %q, want \"high\"", sev["Vulnerable dependency: lodash"])
	}
	if sev["Vulnerable dependency: minimist"] != "medium" {
		t.Errorf("minimist severity = %q, want \"medium\"", sev["Vulnerable dependency: minimist"])
	}
}

func TestParseNPMAuditJSON_V1(t *testing.T) {
	data := []byte(`{
		"advisories": {
			"755": {
				"module_name": "lodash",
				"title": "Prototype Pollution in lodash",
				"overview": "Lodash versions prior to 4.17.21 are vulnerable to prototype pollution.",
				"severity": "high",
				"url": "https://npmjs.com/advisories/755"
			}
		}
	}`)

	risks := parseNPMAuditJSON(data, "ui", "ui/package.json")
	if len(risks) != 1 {
		t.Fatalf("got %d risks, want 1", len(risks))
	}
	r := risks[0]
	if r.Severity != "high" {
		t.Errorf("severity = %q, want \"high\"", r.Severity)
	}
	if r.Title != "Prototype Pollution in lodash" {
		t.Errorf("title = %q, want \"Prototype Pollution in lodash\"", r.Title)
	}
	if !strings.Contains(r.Description, "https://npmjs.com/advisories/755") {
		t.Errorf("description missing URL: %q", r.Description)
	}
	if r.Type != "dep-vuln" {
		t.Errorf("type = %q, want \"dep-vuln\"", r.Type)
	}
	if r.Pass != "npm-audit" {
		t.Errorf("pass = %q, want \"npm-audit\"", r.Pass)
	}
}

func TestParseNPMAuditJSON_NoVulns_V2(t *testing.T) {
	data := []byte(`{"auditReportVersion": 2, "vulnerabilities": {}}`)
	risks := parseNPMAuditJSON(data, "ui", "ui/package.json")
	if len(risks) != 0 {
		t.Errorf("got %d risks, want 0", len(risks))
	}
}

func TestParseNPMAuditJSON_Unrecognised(t *testing.T) {
	data := []byte(`{"something": "else"}`)
	risks := parseNPMAuditJSON(data, "ui", "ui/package.json")
	if risks != nil {
		t.Errorf("got %d risks from unrecognised format, want nil", len(risks))
	}
}

func TestParseNPMAuditJSON_Invalid(t *testing.T) {
	risks := parseNPMAuditJSON([]byte(`not json`), "ui", "ui/package.json")
	if risks != nil {
		t.Errorf("got risks from invalid JSON, want nil")
	}
}

// --- parseGovulncheckJSON ---

func TestParseGovulncheckJSON(t *testing.T) {
	// Simulate a govulncheck -json stream with two findings for the same OSV ID
	// (dedup) and one distinct finding.
	data := `{"message":{"config":{"protocol_version":"v1.0.0","handler":"govulncheck","analysis":"source"}}}
{"message":{"finding":{"osv":"GO-2021-0113","fixed_version":"go1.17.2","trace":[{"module":"stdlib","version":"go1.17","packages":[{"path":"crypto/tls"}]}]}}}
{"message":{"finding":{"osv":"GO-2021-0113","fixed_version":"go1.17.2","trace":[{"module":"stdlib","version":"go1.17"}]}}}
{"message":{"finding":{"osv":"GO-2022-0536","fixed_version":"v0.13.0","trace":[{"module":"golang.org/x/net","version":"v0.5.0"}]}}}
`

	risks := parseGovulncheckJSON([]byte(data))
	if len(risks) != 2 {
		t.Fatalf("got %d risks, want 2 (dedup should collapse duplicate OSV ID)", len(risks))
	}

	osvIDs := map[string]bool{}
	for _, r := range risks {
		if r.Type != "dep-vuln" {
			t.Errorf("type = %q, want \"dep-vuln\"", r.Type)
		}
		if r.Pass != "govulncheck" {
			t.Errorf("pass = %q, want \"govulncheck\"", r.Pass)
		}
		if r.Severity != "high" {
			t.Errorf("severity = %q, want \"high\"", r.Severity)
		}
		if r.File != "go.mod" {
			t.Errorf("file = %q, want \"go.mod\"", r.File)
		}
		if r.Module != "root" {
			t.Errorf("module = %q, want \"root\"", r.Module)
		}
		if r.Confidence != 1.0 {
			t.Errorf("confidence = %v, want 1.0", r.Confidence)
		}
		osvIDs[r.Title] = true
	}
	if !osvIDs["Vulnerability in Go dependency: GO-2021-0113"] {
		t.Errorf("missing GO-2021-0113 finding")
	}
	if !osvIDs["Vulnerability in Go dependency: GO-2022-0536"] {
		t.Errorf("missing GO-2022-0536 finding")
	}
}

func TestParseGovulncheckJSON_FixedVersionInDescription(t *testing.T) {
	data := `{"message":{"finding":{"osv":"GO-2022-1234","fixed_version":"v1.2.3","trace":[{"module":"example.com/foo","version":"v1.0.0"}]}}}
`
	risks := parseGovulncheckJSON([]byte(data))
	if len(risks) != 1 {
		t.Fatalf("got %d risks, want 1", len(risks))
	}
	if !strings.Contains(risks[0].Description, "v1.2.3") {
		t.Errorf("description missing fixed version: %q", risks[0].Description)
	}
}

func TestParseGovulncheckJSON_NoFindings(t *testing.T) {
	data := `{"message":{"config":{"protocol_version":"v1.0.0"}}}
{"message":{"progress":{"message":"No vulnerabilities found."}}}
`
	risks := parseGovulncheckJSON([]byte(data))
	if len(risks) != 0 {
		t.Errorf("got %d risks, want 0", len(risks))
	}
}

func TestParseGovulncheckJSON_MalformedLines(t *testing.T) {
	// A mix of valid and invalid lines; valid findings should still be parsed.
	data := `not json at all
{"message":{"finding":{"osv":"GO-2023-0001","fixed_version":"","trace":[]}}}
{broken
`
	risks := parseGovulncheckJSON([]byte(data))
	if len(risks) != 1 {
		t.Fatalf("got %d risks, want 1 (malformed lines should be skipped)", len(risks))
	}
}

// --- auditGo integration: skipped when no go.sum ---

func TestAuditGo_NoGoSum(t *testing.T) {
	dir := t.TempDir()
	risks, skipped := auditGo(context.Background(), dir)
	if len(risks) != 0 {
		t.Errorf("got %d risks in dir without go.sum, want 0", len(risks))
	}
	if len(skipped) != 0 {
		t.Errorf("got skipped %v in dir without go.sum, want none", skipped)
	}
}

// --- auditJS integration: skipped when no project roots ---

func TestAuditJS_NoProjectRoots(t *testing.T) {
	risks, skipped := auditJS(context.Background(), t.TempDir(), nil)
	if len(risks) != 0 {
		t.Errorf("got %d risks with no project roots, want 0", len(risks))
	}
	if len(skipped) != 0 {
		t.Errorf("got skipped %v with no project roots, want none", skipped)
	}
}

// --- Audit: skipped_passes reported when govulncheck absent ---

func TestAudit_GovulncheckSkippedWhenGoSumPresent(t *testing.T) {
	if _, err := exec.LookPath("govulncheck"); err == nil {
		t.Skip("govulncheck is present on PATH; skipped-pass behaviour not testable")
	}

	dir := t.TempDir()
	// Create a go.sum so auditGo attempts to run govulncheck.
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	_, skipped := Audit(context.Background(), dir, nil)
	found := false
	for _, s := range skipped {
		if s == "govulncheck" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected \"govulncheck\" in skipped passes when binary absent; got %v", skipped)
	}
}

package staticanalysis

import (
	"context"
	"testing"
)

// --- mapSemgrepSeverity ---

func TestMapSemgrepSeverity(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ERROR", "high"},
		{"WARNING", "medium"},
		{"INFO", "low"},
		{"", "low"},
	}
	for _, tt := range tests {
		if got := mapSemgrepSeverity(tt.input); got != tt.want {
			t.Errorf("mapSemgrepSeverity(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- mapSemgrepConfidence ---

func TestMapSemgrepConfidence(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"HIGH", 0.9},
		{"MEDIUM", 0.6},
		{"LOW", 0.3},
		{"", 0.3},
	}
	for _, tt := range tests {
		if got := mapSemgrepConfidence(tt.input); got != tt.want {
			t.Errorf("mapSemgrepConfidence(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- parseSemgrepJSON ---

func TestParseSemgrepJSON(t *testing.T) {
	data := []byte(`{
		"results": [
			{
				"check_id": "go.lang.security.audit.dangerous-exec-command.dangerous-exec-command",
				"path": "internal/depaudit/js.go",
				"start": {"line": 40, "col": 9, "offset": 0},
				"end":   {"line": 40, "col": 61, "offset": 0},
				"extra": {
					"message": "Detected non-static command inside Command.",
					"severity": "ERROR",
					"metadata": {
						"confidence": "LOW",
						"category": "security"
					}
				}
			},
			{
				"check_id": "bash.curl.security.curl-pipe-bash.curl-pipe-bash",
				"path": ".devcontainer/postCreate.sh",
				"start": {"line": 62, "col": 1, "offset": 0},
				"end":   {"line": 62, "col": 87, "offset": 0},
				"extra": {
					"message": "Data is being piped into bash from curl.",
					"severity": "WARNING",
					"metadata": {
						"confidence": "LOW",
						"category": "security"
					}
				}
			}
		]
	}`)

	risks := parseSemgrepJSON(data)
	if len(risks) != 2 {
		t.Fatalf("got %d risks, want 2", len(risks))
	}

	r0 := risks[0]
	if r0.Severity != "high" {
		t.Errorf("risks[0].Severity = %q, want \"high\"", r0.Severity)
	}
	if r0.Type != "static" {
		t.Errorf("risks[0].Type = %q, want \"static\"", r0.Type)
	}
	if r0.Pass != "semgrep" {
		t.Errorf("risks[0].Pass = %q, want \"semgrep\"", r0.Pass)
	}
	if r0.File != "internal/depaudit/js.go" {
		t.Errorf("risks[0].File = %q", r0.File)
	}
	if len(r0.EvidenceLines) != 1 || r0.EvidenceLines[0] != 40 {
		t.Errorf("risks[0].EvidenceLines = %v, want [40]", r0.EvidenceLines)
	}
	if r0.Confidence != 0.3 {
		t.Errorf("risks[0].Confidence = %v, want 0.3 (LOW)", r0.Confidence)
	}

	r1 := risks[1]
	if r1.Severity != "medium" {
		t.Errorf("risks[1].Severity = %q, want \"medium\"", r1.Severity)
	}
}

func TestParseSemgrepJSON_Empty(t *testing.T) {
	data := []byte(`{"results": []}`)
	risks := parseSemgrepJSON(data)
	if len(risks) != 0 {
		t.Errorf("got %d risks for empty results, want 0", len(risks))
	}
}

func TestParseSemgrepJSON_Invalid(t *testing.T) {
	risks := parseSemgrepJSON([]byte(`not json`))
	if risks != nil {
		t.Errorf("got risks from invalid JSON, want nil")
	}
}

// --- parseTrufflehogJSON ---

func TestParseTrufflehogJSON_Verified(t *testing.T) {
	data := []byte(`{"DetectorName":"AWS","DetectorDescription":"AWS credentials detector.","Verified":true,"Redacted":"AKIA***","SourceMetadata":{"Data":{"Filesystem":{"file":"config/deploy.sh","line":12}}}}`)

	risks := parseTrufflehogJSON(data)
	if len(risks) != 1 {
		t.Fatalf("got %d risks, want 1", len(risks))
	}
	r := risks[0]
	if r.Severity != "high" {
		t.Errorf("severity = %q, want \"high\" for verified finding", r.Severity)
	}
	if r.Confidence != 1.0 {
		t.Errorf("confidence = %v, want 1.0 for verified finding", r.Confidence)
	}
	if r.Type != "secret" {
		t.Errorf("type = %q, want \"secret\"", r.Type)
	}
	if r.Pass != "trufflehog" {
		t.Errorf("pass = %q, want \"trufflehog\"", r.Pass)
	}
	if r.File != "config/deploy.sh" {
		t.Errorf("file = %q, want \"config/deploy.sh\"", r.File)
	}
	if len(r.EvidenceLines) != 1 || r.EvidenceLines[0] != 12 {
		t.Errorf("evidence_lines = %v, want [12]", r.EvidenceLines)
	}
}

func TestParseTrufflehogJSON_Unverified(t *testing.T) {
	data := []byte(`{"DetectorName":"GitHub","DetectorDescription":"GitHub token.","Verified":false,"Redacted":"ghp_***","SourceMetadata":{"Data":{"Filesystem":{"file":"scripts/deploy.sh","line":5}}}}`)

	risks := parseTrufflehogJSON(data)
	if len(risks) != 1 {
		t.Fatalf("got %d risks, want 1", len(risks))
	}
	r := risks[0]
	if r.Severity != "medium" {
		t.Errorf("severity = %q, want \"medium\" for unverified finding", r.Severity)
	}
	if r.Confidence != 0.5 {
		t.Errorf("confidence = %v, want 0.5 for unverified finding", r.Confidence)
	}
}

func TestParseTrufflehogJSON_MultipleFindings(t *testing.T) {
	data := []byte(`{"DetectorName":"AWS","DetectorDescription":"","Verified":true,"Redacted":"AKIA***","SourceMetadata":{"Data":{"Filesystem":{"file":"a.sh","line":1}}}}
{"DetectorName":"Slack","DetectorDescription":"","Verified":false,"Redacted":"xoxb-***","SourceMetadata":{"Data":{"Filesystem":{"file":"b.sh","line":3}}}}
`)

	risks := parseTrufflehogJSON(data)
	if len(risks) != 2 {
		t.Fatalf("got %d risks, want 2", len(risks))
	}
}

func TestParseTrufflehogJSON_SkipsEmptyFilePath(t *testing.T) {
	data := []byte(`{"DetectorName":"AWS","DetectorDescription":"","Verified":false,"Redacted":"","SourceMetadata":{"Data":{"Filesystem":{"file":"","line":0}}}}`)
	risks := parseTrufflehogJSON(data)
	if len(risks) != 0 {
		t.Errorf("got %d risks for empty file path, want 0", len(risks))
	}
}

func TestParseTrufflehogJSON_MalformedLine(t *testing.T) {
	data := []byte(`not json
{"DetectorName":"AWS","DetectorDescription":"","Verified":true,"Redacted":"AKIA***","SourceMetadata":{"Data":{"Filesystem":{"file":"ok.sh","line":1}}}}
`)
	risks := parseTrufflehogJSON(data)
	if len(risks) != 1 {
		t.Fatalf("got %d risks, want 1 (malformed line should be skipped)", len(risks))
	}
}

// --- Run: skipped when tools absent ---

func TestRun_BothToolsSkippedInEmptyDir(t *testing.T) {
	// In a temp dir with no recognisable code, both tools either skip or find
	// nothing. This test just verifies Run doesn't panic.
	_, _ = Run(context.Background(), t.TempDir())
}

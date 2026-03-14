package staticanalysis

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"

	"github.com/brentrockwood/mri/schema"
)

// semgrepResult is the top-level JSON object written by semgrep --json.
type semgrepResult struct {
	Results []semgrepFinding `json:"results"`
}

// semgrepFinding is a single finding in the semgrep JSON output.
type semgrepFinding struct {
	CheckID string `json:"check_id"`
	Path    string `json:"path"`
	Start   struct {
		Line int `json:"line"`
	} `json:"start"`
	Extra struct {
		Message  string `json:"message"`
		Severity string `json:"severity"`
		Metadata struct {
			Confidence string `json:"confidence"`
			Category   string `json:"category"`
		} `json:"metadata"`
	} `json:"extra"`
}

// runSemgrep runs semgrep with the auto ruleset against root and converts
// findings to schema.Risk entries. Non-fatal if semgrep is unavailable.
func runSemgrep(ctx context.Context, root string) ([]schema.Risk, []string) {
	semgrepPath, err := exec.LookPath("semgrep")
	if err != nil {
		return nil, []string{"semgrep"}
	}

	// --config auto selects rules based on the languages present in the repo.
	// --quiet suppresses progress output so only JSON goes to stdout.
	// Exit code 1 means findings were found; we parse regardless.
	cmd := exec.CommandContext(ctx, semgrepPath, // #nosec G204 -- semgrepPath from LookPath
		"--json", "--config", "auto", "--quiet", ".")
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run()

	if ctx.Err() != nil || out.Len() == 0 {
		return nil, nil
	}
	return parseSemgrepJSON(out.Bytes()), nil
}

// parseSemgrepJSON converts semgrep --json output to schema.Risk entries.
func parseSemgrepJSON(data []byte) []schema.Risk {
	var result semgrepResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}

	risks := make([]schema.Risk, 0, len(result.Results))
	for _, f := range result.Results {
		risks = append(risks, schema.Risk{
			Severity:      mapSemgrepSeverity(f.Extra.Severity),
			Type:          "static",
			Pass:          "semgrep",
			Module:        "", // assigned by caller via moduleForFile
			File:          f.Path,
			Title:         f.CheckID,
			Description:   f.Extra.Message,
			Confidence:    mapSemgrepConfidence(f.Extra.Metadata.Confidence),
			EvidenceLines: []int{f.Start.Line},
			TargetType:    "file",
			TargetID:      f.Path,
		})
	}
	return risks
}

// mapSemgrepSeverity converts semgrep severity strings to schema severity levels.
func mapSemgrepSeverity(s string) string {
	switch s {
	case "ERROR":
		return "high"
	case "WARNING":
		return "medium"
	default:
		return "low"
	}
}

// mapSemgrepConfidence converts semgrep confidence strings to a [0, 1] float.
func mapSemgrepConfidence(c string) float64 {
	switch c {
	case "HIGH":
		return 0.9
	case "MEDIUM":
		return 0.6
	default: // LOW or absent
		return 0.3
	}
}

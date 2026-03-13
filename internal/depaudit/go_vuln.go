package depaudit

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/brentrockwood/mri/schema"
)

// auditGo runs govulncheck against the Go module rooted at root.
// If go.sum is absent (not a Go module) the pass is silently skipped.
// If govulncheck is not on PATH the "govulncheck" pass is skipped (non-fatal).
func auditGo(ctx context.Context, root string) ([]schema.Risk, []string) {
	// Only run for repositories that have a Go module.
	if _, err := os.Stat(filepath.Join(root, "go.sum")); err != nil {
		return nil, nil
	}

	govulncheckPath, err := exec.LookPath("govulncheck")
	if err != nil {
		return nil, []string{"govulncheck"}
	}

	cmd := exec.CommandContext(ctx, govulncheckPath, "-json", "./...") // #nosec G204 -- govulncheckPath from LookPath
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	// govulncheck exits 3 when vulnerabilities are found; ignore exit code.
	_ = cmd.Run()

	if out.Len() == 0 {
		return nil, nil
	}
	return parseGovulncheckJSON(out.Bytes()), nil
}

// govulncheckEnvelope wraps a single newline-delimited JSON object emitted by
// govulncheck -json. Only the "finding" message type is consumed; all others
// are skipped.
type govulncheckEnvelope struct {
	Message struct {
		Finding *govulncheckFinding `json:"finding"`
	} `json:"message"`
}

// govulncheckFinding is the payload of a govulncheck "finding" message.
type govulncheckFinding struct {
	OSV          string `json:"osv"`
	FixedVersion string `json:"fixed_version"`
	Trace        []struct {
		Module  string `json:"module"`
		Version string `json:"version"`
	} `json:"trace"`
}

// parseGovulncheckJSON reads the newline-delimited JSON stream produced by
// govulncheck -json and returns one schema.Risk per unique OSV ID.
// Unrecognised lines are silently skipped so future format changes are
// non-fatal.
func parseGovulncheckJSON(data []byte) []schema.Risk {
	const goModFile = "go.mod"
	seen := make(map[string]bool)
	var risks []schema.Risk

	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Increase the scanner buffer for long lines (some govulncheck OSV payloads
	// can be large).
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var env govulncheckEnvelope
		if err := json.Unmarshal(scanner.Bytes(), &env); err != nil {
			continue
		}
		f := env.Message.Finding
		if f == nil || f.OSV == "" {
			continue
		}
		if seen[f.OSV] {
			continue
		}
		seen[f.OSV] = true

		// Best-effort: use the first trace entry's module and version for context.
		var modDesc string
		if len(f.Trace) > 0 {
			modDesc = f.Trace[0].Module
			if f.Trace[0].Version != "" {
				modDesc += "@" + f.Trace[0].Version
			}
		}

		title := fmt.Sprintf("Vulnerability in Go dependency: %s", f.OSV)
		desc := fmt.Sprintf("Known vulnerability %s", f.OSV)
		if modDesc != "" {
			desc += fmt.Sprintf(" in %s", modDesc)
		}
		if f.FixedVersion != "" {
			desc += fmt.Sprintf(". Fixed in %s.", f.FixedVersion)
		} else {
			desc += "."
		}

		risks = append(risks, schema.Risk{
			// govulncheck only reports confirmed reachable vulnerabilities;
			// all findings are treated as high severity.
			Severity:    "high",
			Type:        "dep-vuln",
			Pass:        "govulncheck",
			Module:      "root",
			File:        goModFile,
			Title:       title,
			Description: desc,
			Confidence:  1.0,
			TargetType:  "file",
			TargetID:    goModFile,
		})
	}
	return risks
}

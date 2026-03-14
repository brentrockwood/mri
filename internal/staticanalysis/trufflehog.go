package staticanalysis

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/brentrockwood/mri/schema"
)

// trufflehogFinding is a single JSON object from the trufflehog --json stream.
type trufflehogFinding struct {
	DetectorName        string `json:"DetectorName"`
	DetectorDescription string `json:"DetectorDescription"`
	Verified            bool   `json:"Verified"`
	Redacted            string `json:"Redacted"`
	SourceMetadata      struct {
		Data struct {
			Filesystem struct {
				File string `json:"file"`
				Line int    `json:"line"`
			} `json:"Filesystem"`
		} `json:"Data"`
	} `json:"SourceMetadata"`
}

// runTrufflehog scans root for secrets using trufflehog. Non-fatal if
// trufflehog is unavailable. node_modules, .git, dist, bin, and vendor trees
// are excluded to suppress false positives from vendored type definitions.
func runTrufflehog(ctx context.Context, root string) ([]schema.Risk, []string) {
	trufflehogPath, err := exec.LookPath("trufflehog")
	if err != nil {
		return nil, []string{"trufflehog"}
	}

	// Write the exclusion pattern file to a temp location.
	excludeFile, err := writeTrufflehogExcludes()
	if err != nil {
		// Non-fatal: if we can't write the temp file, skip the pass.
		return nil, []string{"trufflehog"}
	}
	defer os.Remove(excludeFile) // #nosec G104 -- best-effort cleanup of temp file

	cmd := exec.CommandContext(ctx, trufflehogPath, // #nosec G204 -- trufflehogPath from LookPath
		"filesystem",
		"--json",
		"--no-update",
		"--exclude-paths", excludeFile,
		".",
	)
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	// trufflehog exits non-zero when findings are present; ignore exit code.
	_ = cmd.Run()

	if ctx.Err() != nil || out.Len() == 0 {
		return nil, nil
	}
	return parseTrufflehogJSON(out.Bytes()), nil
}

// writeTrufflehogExcludes writes a temp file containing glob patterns for
// directories that should not be scanned. Returns the file path.
func writeTrufflehogExcludes() (string, error) {
	f, err := os.CreateTemp("", "trufflehog-excludes-*.txt")
	if err != nil {
		return "", err
	}
	defer f.Close() // #nosec G307 -- temp file, write then close
	excludes := []string{
		"node_modules",
		".git",
		"dist",
		"bin",
		"vendor",
	}
	for _, pattern := range excludes {
		if _, err := fmt.Fprintln(f, pattern); err != nil {
			return "", err
		}
	}
	return f.Name(), nil
}

// parseTrufflehogJSON reads the newline-delimited JSON stream from trufflehog
// and returns one schema.Risk per finding. Unverified findings are emitted at
// medium severity; verified findings at high.
func parseTrufflehogJSON(data []byte) []schema.Risk {
	var risks []schema.Risk
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var f trufflehogFinding
		if err := json.Unmarshal(scanner.Bytes(), &f); err != nil {
			continue
		}
		filePath := f.SourceMetadata.Data.Filesystem.File
		if filePath == "" {
			continue
		}

		severity := "medium"
		if f.Verified {
			severity = "high"
		}

		title := fmt.Sprintf("Potential secret: %s", f.DetectorName)
		desc := f.DetectorDescription
		if f.Redacted != "" {
			desc += fmt.Sprintf(" Value (redacted): %s", f.Redacted)
		}
		if !f.Verified {
			desc += " (unverified)"
		}

		var evidenceLines []int
		if f.SourceMetadata.Data.Filesystem.Line > 0 {
			evidenceLines = []int{f.SourceMetadata.Data.Filesystem.Line}
		}

		risks = append(risks, schema.Risk{
			Severity:      severity,
			Type:          "secret",
			Pass:          "trufflehog",
			Module:        "",
			File:          filePath,
			Title:         title,
			Description:   desc,
			Confidence:    trufflehogConfidence(f.Verified),
			EvidenceLines: evidenceLines,
			TargetType:    "file",
			TargetID:      filePath,
		})
	}
	return risks
}

// trufflehogConfidence returns a confidence score based on whether the finding
// was verified against an external service.
func trufflehogConfidence(verified bool) float64 {
	if verified {
		return 1.0
	}
	return 0.5
}

// Package staticanalysis implements Phase 9 deterministic static analysis
// passes. Each pass wraps an external CLI tool (semgrep, trufflehog, …) and
// converts its output to schema.Risk entries. All passes are non-fatal: if the
// required tool is unavailable its name is appended to the skipped slice.
package staticanalysis

import (
	"context"

	"github.com/brentrockwood/mri/schema"
)

// Run executes all static analysis passes against the repository at root and
// returns the combined risks and the names of any passes that were skipped
// because the required tool was not available.
func Run(ctx context.Context, root string) (risks []schema.Risk, skipped []string) {
	semgrepRisks, semgrepSkipped := runSemgrep(ctx, root)
	risks = append(risks, semgrepRisks...)
	skipped = append(skipped, semgrepSkipped...)

	trufflehogRisks, trufflehogSkipped := runTrufflehog(ctx, root)
	risks = append(risks, trufflehogRisks...)
	skipped = append(skipped, trufflehogSkipped...)

	return risks, skipped
}

// Package depaudit implements Phase 8 dependency vulnerability auditing.
// It runs npm audit for JS/TS projects and govulncheck for Go modules,
// converting findings to schema.Risk entries with type "dep-vuln".
// All passes are non-fatal: if a required tool is unavailable the pass name
// is appended to the skipped slice so the caller can record it in
// meta.skipped_passes.
package depaudit

import (
	"context"

	"github.com/brentrockwood/mri/schema"
)

// Audit runs dependency vulnerability audits for all detected project types
// in the repository at root. jsProjectRoots are repo-relative slash paths of
// directories containing a non-root package.json (as produced by ingestion).
//
// It returns the combined risks and the names of any passes that were skipped
// because the required tool was not available.
func Audit(ctx context.Context, root string, jsProjectRoots []string) (risks []schema.Risk, skipped []string) {
	jsRisks, jsSkipped := auditJS(ctx, root, jsProjectRoots)
	risks = append(risks, jsRisks...)
	skipped = append(skipped, jsSkipped...)

	goRisks, goSkipped := auditGo(ctx, root)
	risks = append(risks, goRisks...)
	skipped = append(skipped, goSkipped...)

	return risks, skipped
}

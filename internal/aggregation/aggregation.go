// Package aggregation deduplicates risks and computes risk scores for files
// and modules in a repo-mri analysis.
package aggregation

import (
	"sort"

	"github.com/brentrockwood/mri/schema"
)

// severityWeight maps severity labels to their scoring weight.
var severityWeight = map[string]float64{
	"high":   1.0,
	"medium": 0.5,
	"low":    0.25,
}

// Aggregate deduplicates risks, then computes and populates risk scores
// on all files and modules in a. Modules are re-sorted by risk score
// descending after scoring.
func Aggregate(a *schema.Analysis) {
	a.Risks = deduplicate(a.Risks)
	scoreFiles(a)
	scoreModules(a)
	sortModules(a)
}

// deduplicate removes duplicate risks. A duplicate is any two risks with the
// same File and Type. When duplicates exist, the one with the higher
// Confidence is kept. Original order is preserved for non-duplicates (first
// occurrence wins unless a later one has higher confidence).
func deduplicate(risks []schema.Risk) []schema.Risk {
	type key struct{ file, typ string }

	// Map from dedup key to index in result slice.
	index := make(map[key]int, len(risks))
	result := make([]schema.Risk, 0, len(risks))

	for _, r := range risks {
		k := key{r.File, r.Type}
		if i, exists := index[k]; exists {
			// Replace if the new one has strictly higher confidence.
			if r.Confidence > result[i].Confidence {
				result[i] = r
			}
		} else {
			index[k] = len(result)
			result = append(result, r)
		}
	}

	return result
}

// scoreFiles computes and sets RiskScore on every File in a.
//
// Formula:
//
//	per_finding_score = severity_weight[severity] * confidence
//	file_risk_score   = min(1.0, sum of per_finding_score for all findings in file)
func scoreFiles(a *schema.Analysis) {
	// Build a map of file path → sum of per-finding scores.
	sums := make(map[string]float64, len(a.Files))
	for _, r := range a.Risks {
		w := severityWeight[r.Severity] // zero if unknown, intentional
		sums[r.File] += w * r.Confidence
	}

	for i := range a.Files {
		s := sums[a.Files[i].Path]
		if s > 1.0 {
			s = 1.0
		}
		a.Files[i].RiskScore = s
	}
}

// scoreModules computes and sets RiskScore on every Module in a.
//
// The score is the mean RiskScore across ALL files that belong to the module.
// Files with no findings contribute 0.0 to the mean.
func scoreModules(a *schema.Analysis) {
	// Build a map of module ID → (sum, count).
	type stat struct {
		sum   float64
		count int
	}
	stats := make(map[string]*stat, len(a.Modules))
	for i := range a.Modules {
		stats[a.Modules[i].ID] = &stat{}
	}

	for _, f := range a.Files {
		if s, ok := stats[f.Module]; ok {
			s.sum += f.RiskScore
			s.count++
		}
	}

	for i := range a.Modules {
		s := stats[a.Modules[i].ID]
		if s != nil && s.count > 0 {
			a.Modules[i].RiskScore = s.sum / float64(s.count)
		}
	}
}

// sortModules sorts a.Modules descending by RiskScore. Ties are broken
// alphabetically by ID.
func sortModules(a *schema.Analysis) {
	sort.SliceStable(a.Modules, func(i, j int) bool {
		mi, mj := a.Modules[i], a.Modules[j]
		if mi.RiskScore != mj.RiskScore {
			return mi.RiskScore > mj.RiskScore
		}
		return mi.ID < mj.ID
	})
}

package aggregation

import (
	"testing"

	"github.com/brentrockwood/mri/schema"
)

const epsilon = 1e-9

// approxEqual reports whether a and b differ by less than epsilon.
func approxEqual(a, b float64) bool {
	d := a - b
	return d < epsilon && d > -epsilon
}

// risk is a helper to build a schema.Risk concisely.
func risk(file, typ, severity string, confidence float64) schema.Risk {
	return schema.Risk{
		File:       file,
		Type:       typ,
		Severity:   severity,
		Confidence: confidence,
	}
}

func TestDeduplicate_SameFileAndType_KeepsHigherConfidence(t *testing.T) {
	input := []schema.Risk{
		risk("a.go", "lint", "high", 0.6),
		risk("a.go", "lint", "high", 0.9), // higher confidence — should win
	}
	got := deduplicate(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 risk, got %d", len(got))
	}
	if got[0].Confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", got[0].Confidence)
	}
}

func TestDeduplicate_SameFileAndType_FirstWinsIfLowerConfidenceComesLater(t *testing.T) {
	// First occurrence has higher confidence — it should be kept.
	input := []schema.Risk{
		risk("a.go", "lint", "high", 0.9),
		risk("a.go", "lint", "high", 0.3),
	}
	got := deduplicate(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 risk, got %d", len(got))
	}
	if got[0].Confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", got[0].Confidence)
	}
}

func TestDeduplicate_SameFileDifferentType_BothKept(t *testing.T) {
	input := []schema.Risk{
		risk("a.go", "lint", "high", 0.8),
		risk("a.go", "security", "medium", 0.7),
	}
	got := deduplicate(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 risks, got %d", len(got))
	}
}

func TestDeduplicate_PreservesOrder(t *testing.T) {
	input := []schema.Risk{
		risk("b.go", "lint", "low", 0.5),
		risk("a.go", "lint", "low", 0.5),
		risk("c.go", "lint", "low", 0.5),
	}
	got := deduplicate(input)
	if len(got) != 3 {
		t.Fatalf("expected 3 risks, got %d", len(got))
	}
	files := []string{got[0].File, got[1].File, got[2].File}
	want := []string{"b.go", "a.go", "c.go"}
	for i := range want {
		if files[i] != want[i] {
			t.Errorf("position %d: got %s, want %s", i, files[i], want[i])
		}
	}
}

func TestScoreFiles_SingleHighConfidenceHighSeverity(t *testing.T) {
	a := &schema.Analysis{
		Files: []schema.File{{Path: "a.go", Module: "mod"}},
		Risks: []schema.Risk{risk("a.go", "lint", "high", 1.0)},
	}
	scoreFiles(a)
	if a.Files[0].RiskScore != 1.0 {
		t.Errorf("expected 1.0, got %f", a.Files[0].RiskScore)
	}
}

func TestScoreFiles_MultipleFindingsCappedAt1(t *testing.T) {
	a := &schema.Analysis{
		Files: []schema.File{{Path: "a.go", Module: "mod"}},
		Risks: []schema.Risk{
			risk("a.go", "lint", "high", 1.0),
			risk("a.go", "security", "high", 1.0),
		},
	}
	scoreFiles(a)
	if a.Files[0].RiskScore != 1.0 {
		t.Errorf("expected cap at 1.0, got %f", a.Files[0].RiskScore)
	}
}

func TestScoreFiles_NoFindings_ZeroScore(t *testing.T) {
	a := &schema.Analysis{
		Files: []schema.File{{Path: "a.go", Module: "mod"}},
		Risks: nil,
	}
	scoreFiles(a)
	if a.Files[0].RiskScore != 0.0 {
		t.Errorf("expected 0.0, got %f", a.Files[0].RiskScore)
	}
}

func TestScoreFiles_MediumSeverity(t *testing.T) {
	a := &schema.Analysis{
		Files: []schema.File{{Path: "a.go", Module: "mod"}},
		Risks: []schema.Risk{risk("a.go", "lint", "medium", 0.8)},
	}
	scoreFiles(a)
	want := 0.5 * 0.8 // 0.4
	if !approxEqual(a.Files[0].RiskScore, want) {
		t.Errorf("expected %f, got %f", want, a.Files[0].RiskScore)
	}
}

func TestScoreModules_MeanAcrossAllFilesIncludingZero(t *testing.T) {
	// Module has 3 files: scores 0.8, 0.4, 0.0 → mean = 0.4
	a := &schema.Analysis{
		Modules: []schema.Module{{ID: "mod"}},
		Files: []schema.File{
			{Path: "a.go", Module: "mod", RiskScore: 0.8},
			{Path: "b.go", Module: "mod", RiskScore: 0.4},
			{Path: "c.go", Module: "mod", RiskScore: 0.0},
		},
	}
	scoreModules(a)
	want := (0.8 + 0.4 + 0.0) / 3.0
	if !approxEqual(a.Modules[0].RiskScore, want) {
		t.Errorf("expected %f, got %f", want, a.Modules[0].RiskScore)
	}
}

func TestScoreModules_EmptyModule_ZeroScore(t *testing.T) {
	a := &schema.Analysis{
		Modules: []schema.Module{{ID: "mod"}},
		Files:   nil,
	}
	scoreModules(a)
	if a.Modules[0].RiskScore != 0.0 {
		t.Errorf("expected 0.0, got %f", a.Modules[0].RiskScore)
	}
}

func TestSortModules_DescendingByRiskScore(t *testing.T) {
	a := &schema.Analysis{
		Modules: []schema.Module{
			{ID: "low", RiskScore: 0.1},
			{ID: "high", RiskScore: 0.9},
			{ID: "med", RiskScore: 0.5},
		},
	}
	sortModules(a)
	order := []string{a.Modules[0].ID, a.Modules[1].ID, a.Modules[2].ID}
	want := []string{"high", "med", "low"}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("position %d: got %s, want %s", i, order[i], want[i])
		}
	}
}

func TestSortModules_TiesAlphabeticalByID(t *testing.T) {
	a := &schema.Analysis{
		Modules: []schema.Module{
			{ID: "zebra", RiskScore: 0.5},
			{ID: "alpha", RiskScore: 0.5},
			{ID: "mango", RiskScore: 0.5},
		},
	}
	sortModules(a)
	order := []string{a.Modules[0].ID, a.Modules[1].ID, a.Modules[2].ID}
	want := []string{"alpha", "mango", "zebra"}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("position %d: got %s, want %s", i, order[i], want[i])
		}
	}
}

func TestAggregate_EndToEnd(t *testing.T) {
	a := &schema.Analysis{
		Modules: []schema.Module{
			{ID: "pkg/a"},
			{ID: "pkg/b"},
		},
		Files: []schema.File{
			{Path: "pkg/a/main.go", Module: "pkg/a"},
			{Path: "pkg/b/util.go", Module: "pkg/b"},
		},
		Risks: []schema.Risk{
			// Two risks for same file+type — lower confidence first.
			risk("pkg/a/main.go", "lint", "high", 0.5),
			risk("pkg/a/main.go", "lint", "high", 0.8), // should win
			// Different type on same file.
			risk("pkg/a/main.go", "security", "medium", 0.6),
		},
	}

	Aggregate(a)

	// Deduplication: lint risk on pkg/a/main.go should be confidence 0.8.
	var lintRisk *schema.Risk
	for i := range a.Risks {
		if a.Risks[i].File == "pkg/a/main.go" && a.Risks[i].Type == "lint" {
			lintRisk = &a.Risks[i]
		}
	}
	if lintRisk == nil {
		t.Fatal("lint risk for pkg/a/main.go not found")
	}
	if lintRisk.Confidence != 0.8 {
		t.Errorf("expected lint confidence 0.8, got %f", lintRisk.Confidence)
	}

	// Total risks: 2 (lint deduped, security kept).
	if len(a.Risks) != 2 {
		t.Errorf("expected 2 risks after dedup, got %d", len(a.Risks))
	}

	// File score for pkg/a/main.go: min(1, 1.0*0.8 + 0.5*0.6) = min(1, 1.1) = 1.0
	var fileA *schema.File
	for i := range a.Files {
		if a.Files[i].Path == "pkg/a/main.go" {
			fileA = &a.Files[i]
		}
	}
	if fileA == nil {
		t.Fatal("file pkg/a/main.go not found")
	}
	if fileA.RiskScore != 1.0 {
		t.Errorf("expected file risk 1.0 (capped), got %f", fileA.RiskScore)
	}

	// Module ranking: pkg/a should be first (higher risk score than pkg/b which has 0).
	if a.Modules[0].ID != "pkg/a" {
		t.Errorf("expected pkg/a first, got %s", a.Modules[0].ID)
	}
}

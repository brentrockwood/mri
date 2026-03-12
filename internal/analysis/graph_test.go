package analysis

import (
	"testing"

	"github.com/brentrockwood/mri/schema"
)

func TestGraphMetrics_Linear(t *testing.T) {
	// a → b → c: chain depth 3, in-degrees: a=0, b=1, c=1
	deps := []schema.Dependency{
		{From: "a", To: "b"},
		{From: "b", To: "c"},
	}
	mods := []schema.Module{{ID: "a"}, {ID: "b"}, {ID: "c"}}

	inDeg, maxDepth := graphMetrics(deps, mods)

	if maxDepth != 3 {
		t.Errorf("maxDepth: want 3, got %d", maxDepth)
	}
	wantInDeg := map[string]int{"a": 0, "b": 1, "c": 1}
	for id, want := range wantInDeg {
		if inDeg[id] != want {
			t.Errorf("inDeg[%s]: want %d, got %d", id, want, inDeg[id])
		}
	}
}

func TestGraphMetrics_Diamond(t *testing.T) {
	// a → b, a → c, b → d, c → d
	// maxDepth: a→b→d = 3 or a→c→d = 3
	// in-degrees: a=0, b=1, c=1, d=2
	deps := []schema.Dependency{
		{From: "a", To: "b"},
		{From: "a", To: "c"},
		{From: "b", To: "d"},
		{From: "c", To: "d"},
	}
	mods := []schema.Module{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}

	inDeg, maxDepth := graphMetrics(deps, mods)

	if maxDepth != 3 {
		t.Errorf("maxDepth: want 3, got %d", maxDepth)
	}
	if inDeg["d"] != 2 {
		t.Errorf("inDeg[d]: want 2, got %d", inDeg["d"])
	}
	if inDeg["a"] != 0 {
		t.Errorf("inDeg[a]: want 0, got %d", inDeg["a"])
	}
}

func TestGraphMetrics_Cycle(t *testing.T) {
	// a → b → a (cycle) — must not hang, maxDepth must be finite
	deps := []schema.Dependency{
		{From: "a", To: "b"},
		{From: "b", To: "a"},
	}
	mods := []schema.Module{{ID: "a"}, {ID: "b"}}

	inDeg, maxDepth := graphMetrics(deps, mods)

	// Both nodes import each other, so in-degrees are 1.
	if inDeg["a"] != 1 || inDeg["b"] != 1 {
		t.Errorf("inDeg: want a=1 b=1, got a=%d b=%d", inDeg["a"], inDeg["b"])
	}
	// The cycle is broken, depth is finite (≤ 3 for this tiny graph).
	if maxDepth < 1 || maxDepth > 3 {
		t.Errorf("maxDepth with cycle: want 1–3, got %d", maxDepth)
	}
}

func TestGraphMetrics_Empty(t *testing.T) {
	_, maxDepth := graphMetrics(nil, []schema.Module{{ID: "a"}})
	if maxDepth != 1 {
		t.Errorf("isolated module: want depth 1, got %d", maxDepth)
	}
}

func TestMostImported(t *testing.T) {
	mods := []schema.Module{
		{ID: "a", ImportCount: 0},
		{ID: "b", ImportCount: 3},
		{ID: "c", ImportCount: 1},
		{ID: "d", ImportCount: 5},
	}
	top := MostImported(mods, 2)
	if len(top) != 2 {
		t.Fatalf("want 2, got %d", len(top))
	}
	if top[0].ID != "d" || top[1].ID != "b" {
		t.Errorf("want d,b got %s,%s", top[0].ID, top[1].ID)
	}
}

func TestMostImported_FewerThanN(t *testing.T) {
	mods := []schema.Module{
		{ID: "a", ImportCount: 2},
	}
	top := MostImported(mods, 10)
	if len(top) != 1 {
		t.Errorf("want 1, got %d", len(top))
	}
}

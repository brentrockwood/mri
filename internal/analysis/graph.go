package analysis

import (
	"github.com/brentrockwood/mri/schema"
)

// graphMetrics computes two signals from the module dependency graph:
//
//   - inDeg: how many distinct modules import each module (in-degree).
//   - maxDepth: the length of the longest dependency chain in the graph.
//
// Cycles are handled gracefully: back edges are ignored so the traversal
// always terminates and depth is computed over acyclic sub-paths.
func graphMetrics(deps []schema.Dependency, modules []schema.Module) (inDeg map[string]int, maxDepth int) {
	// Initialize every known module with in-degree 0.
	inDeg = make(map[string]int, len(modules))
	for _, m := range modules {
		inDeg[m.ID] = 0
	}

	// Build adjacency list and count in-degrees, deduplicating edges.
	adj := make(map[string][]string, len(modules))
	seen := make(map[string]bool, len(deps))
	for _, d := range deps {
		key := d.From + "\x00" + d.To
		if seen[key] {
			continue
		}
		seen[key] = true
		adj[d.From] = append(adj[d.From], d.To)
		inDeg[d.To]++
	}

	// Compute per-node depth (longest path from node to any leaf) using DFS
	// with three-colour marking to detect and skip back edges in cycles.
	const (
		white = 0 // unvisited
		grey  = 1 // currently on the DFS stack (potential back edge)
		black = 2 // fully processed
	)
	color := make(map[string]int, len(modules))
	depth := make(map[string]int, len(modules))

	var dfs func(node string) int
	dfs = func(node string) int {
		if color[node] == black {
			return depth[node]
		}
		if color[node] == grey {
			// Back edge — break the cycle by returning 0.
			return 0
		}
		color[node] = grey
		maxChild := 0
		for _, to := range adj[node] {
			if d := dfs(to); d > maxChild {
				maxChild = d
			}
		}
		depth[node] = 1 + maxChild
		color[node] = black
		return depth[node]
	}

	for _, m := range modules {
		if color[m.ID] == white {
			dfs(m.ID)
		}
	}

	for _, d := range depth {
		if d > maxDepth {
			maxDepth = d
		}
	}

	return inDeg, maxDepth
}

// MostImported returns up to n modules sorted by import count (descending).
// Modules with zero imports are excluded. Returns nil for n <= 0.
func MostImported(modules []schema.Module, n int) []schema.Module {
	if n <= 0 {
		return nil
	}
	// Copy to avoid mutating the original slice order.
	ranked := make([]schema.Module, 0, len(modules))
	for _, m := range modules {
		if m.ImportCount > 0 {
			ranked = append(ranked, m)
		}
	}
	// Insertion-sort is fine for the small n we care about.
	for i := 1; i < len(ranked); i++ {
		for j := i; j > 0 && ranked[j].ImportCount > ranked[j-1].ImportCount; j-- {
			ranked[j], ranked[j-1] = ranked[j-1], ranked[j]
		}
	}
	if n > len(ranked) {
		n = len(ranked)
	}
	return ranked[:n]
}

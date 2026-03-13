import { describe, it, expect } from 'vitest'
import { aggregateEdges, computeLayout, groupByDepth, pathDepth, topSegment } from './layered'
import type { Dependency, Module } from '../types/analysis'

function mod(id: string, fileCount = 1): Module {
  return { id, path: id, language: 'Go', file_count: fileCount, risk_score: 0, complexity_score: 0 }
}

// ── pathDepth ─────────────────────────────────────────────────────────────────

describe('pathDepth', () => {
  it('returns 0 for a single-segment path', () => expect(pathDepth('schema')).toBe(0))
  it('returns 1 for a two-segment path', () => expect(pathDepth('internal/analysis')).toBe(1))
  it('returns 2 for a three-segment path', () => expect(pathDepth('a/b/c')).toBe(2))
})

// ── topSegment ────────────────────────────────────────────────────────────────

describe('topSegment', () => {
  it('returns the single segment unchanged', () => expect(topSegment('cmd')).toBe('cmd'))
  it('returns the first segment of a multi-segment path', () =>
    expect(topSegment('internal/analysis')).toBe('internal'))
})

// ── aggregateEdges ────────────────────────────────────────────────────────────

describe('aggregateEdges', () => {
  it('counts unique from→to pairs', () => {
    const deps: Dependency[] = [
      { from: 'a', to: 'b', type: 'import' },
      { from: 'a', to: 'b', type: 'import' },
      { from: 'b', to: 'c', type: 'import' },
    ]
    const edges = aggregateEdges(deps)
    expect(edges).toHaveLength(2)
    const ab = edges.find((e) => e.fromId === 'a' && e.toId === 'b')
    expect(ab?.weight).toBe(2)
  })

  it('returns an empty array for no dependencies', () => {
    expect(aggregateEdges([])).toEqual([])
  })
})

// ── groupByDepth ──────────────────────────────────────────────────────────────

describe('groupByDepth', () => {
  const modules = [
    mod('schema'),
    mod('internal/analysis'),
    mod('internal/ingestion'),
    mod('cmd/repo-mri'),
  ]

  it('places depth-0 modules in group 0', () => {
    const groups = groupByDepth(modules)
    expect(groups.get(0)?.map((m) => m.id)).toEqual(['schema'])
  })

  it('groups depth-1 modules together', () => {
    const groups = groupByDepth(modules)
    const ids = groups.get(1)?.map((m) => m.id) ?? []
    expect(ids).toContain('internal/analysis')
    expect(ids).toContain('internal/ingestion')
    expect(ids).toContain('cmd/repo-mri')
  })

  it('sorts siblings lexicographically within each group', () => {
    const groups = groupByDepth(modules)
    const ids = groups.get(1)?.map((m) => m.id) ?? []
    expect(ids).toEqual([...ids].sort())
  })
})

// ── computeLayout Level 2 (Modules) ──────────────────────────────────────────

describe('computeLayout — Level 2 (Modules)', () => {
  const modules = [
    mod('schema', 2),
    mod('internal/analysis', 5),
    mod('internal/ingestion', 3),
    mod('cmd/repo-mri', 1),
  ]
  const deps: Dependency[] = [
    { from: 'cmd/repo-mri', to: 'internal/analysis', type: 'import' },
    { from: 'internal/analysis', to: 'schema', type: 'import' },
  ]

  it('assigns depth-0 nodes a smaller y than depth-1 nodes', () => {
    const layout = computeLayout(modules, deps, [], undefined, 2, null)
    const schemaY = layout.nodes.find((n) => n.id === 'schema')!.y
    const analysisY = layout.nodes.find((n) => n.id === 'internal/analysis')!.y
    expect(schemaY).toBeLessThan(analysisY)
  })

  it('assigns the same y to siblings at the same depth', () => {
    const layout = computeLayout(modules, deps, [], undefined, 2, null)
    const analysisY = layout.nodes.find((n) => n.id === 'internal/analysis')!.y
    const ingestionY = layout.nodes.find((n) => n.id === 'internal/ingestion')!.y
    expect(analysisY).toBe(ingestionY)
  })

  it('orders siblings left-to-right lexicographically', () => {
    const layout = computeLayout(modules, deps, [], undefined, 2, null)
    const analysisX = layout.nodes.find((n) => n.id === 'internal/analysis')!.x
    const ingestionX = layout.nodes.find((n) => n.id === 'internal/ingestion')!.x
    // 'analysis' < 'ingestion' lexicographically → analysis is further left
    expect(analysisX).toBeLessThan(ingestionX)
  })

  it('sizes nodes proportionally to file_count', () => {
    const layout = computeLayout(modules, deps, [], undefined, 2, null)
    const bigW = layout.nodes.find((n) => n.id === 'internal/analysis')!.width // file_count: 5
    const smallW = layout.nodes.find((n) => n.id === 'cmd/repo-mri')!.width // file_count: 1
    expect(bigW).toBeGreaterThan(smallW)
  })

  it('produces edges from dependencies', () => {
    const layout = computeLayout(modules, deps, [], undefined, 2, null)
    expect(layout.edges).toHaveLength(2)
  })

  it('returns positive canvas dimensions', () => {
    const layout = computeLayout(modules, deps, [], undefined, 2, null)
    expect(layout.canvasWidth).toBeGreaterThan(0)
    expect(layout.canvasHeight).toBeGreaterThan(0)
  })
})

// ── computeLayout Level 1 (Architecture) ─────────────────────────────────────

describe('computeLayout — Level 1 (Architecture)', () => {
  const modules = [
    mod('internal/analysis', 5),
    mod('internal/ingestion', 3),
    mod('cmd/repo-mri', 1),
    mod('schema', 2),
  ]
  const deps: Dependency[] = [{ from: 'cmd/repo-mri', to: 'internal/analysis', type: 'import' }]

  it('collapses modules to their top-level segment', () => {
    const layout = computeLayout(modules, deps, [], undefined, 1, null)
    const ids = layout.nodes.map((n) => n.id)
    expect(ids).toContain('internal')
    expect(ids).toContain('cmd')
    expect(ids).toContain('schema')
    expect(ids).not.toContain('internal/analysis')
  })

  it('orders segments lexicographically', () => {
    const layout = computeLayout(modules, deps, [], undefined, 1, null)
    const ids = layout.nodes.map((n) => n.id)
    expect(ids).toEqual([...ids].sort())
  })

  it('produces cross-segment edges only', () => {
    const layout = computeLayout(modules, deps, [], undefined, 1, null)
    expect(layout.edges).toHaveLength(1)
    expect(layout.edges[0].fromId).toBe('cmd')
    expect(layout.edges[0].toId).toBe('internal')
  })
})

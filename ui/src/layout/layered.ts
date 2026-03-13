import type { Dependency, File, Module } from '../types/analysis'
import type { LayoutEdge, LayoutNode, LayoutResult, ZoomLevel } from './types'

// ── Layout constants ──────────────────────────────────────────────────────────

const NODE_MIN_W = 80
const NODE_MAX_W = 200
const NODE_H = 48
const H_GAP = 24
const V_GAP = 100
const CANVAS_PAD = 60

// ── Pure helpers ──────────────────────────────────────────────────────────────

/** Number of path segments minus 1. e.g. "internal/analysis" → 1 */
export function pathDepth(id: string): number {
  return id.split('/').length - 1
}

/** First path segment. e.g. "internal/analysis" → "internal" */
export function topSegment(id: string): string {
  return id.split('/')[0]
}

/** Normalise a count to [NODE_MIN_W, NODE_MAX_W]. */
function scaleWidth(value: number, min: number, max: number): number {
  if (max === min) return (NODE_MIN_W + NODE_MAX_W) / 2
  return NODE_MIN_W + ((value - min) / (max - min)) * (NODE_MAX_W - NODE_MIN_W)
}

/**
 * Aggregates dependency declarations into weighted directed edges.
 * Multiple declarations between the same module pair are counted as one edge
 * with the combined count as weight.
 */
export function aggregateEdges(dependencies: Dependency[]): LayoutEdge[] {
  const weights = new Map<string, number>()
  for (const dep of dependencies) {
    const key = `${dep.from}\u2192${dep.to}`
    weights.set(key, (weights.get(key) ?? 0) + 1)
  }
  return Array.from(weights.entries()).map(([key, weight]) => {
    const arrow = key.indexOf('\u2192')
    return { fromId: key.slice(0, arrow), toId: key.slice(arrow + 1), weight }
  })
}

// ── Level 2: Modules ──────────────────────────────────────────────────────────

/** Groups modules by path depth, sorted lexicographically within each group. */
export function groupByDepth(modules: Module[]): Map<number, Module[]> {
  const groups = new Map<number, Module[]>()
  for (const mod of modules) {
    const d = pathDepth(mod.id)
    if (!groups.has(d)) groups.set(d, [])
    groups.get(d)!.push(mod)
  }
  for (const group of groups.values()) {
    group.sort((a, b) => a.id.localeCompare(b.id))
  }
  return groups
}

function layoutModules(modules: Module[], dependencies: Dependency[]): LayoutResult {
  if (modules.length === 0) return empty()

  const counts = modules.map((m) => m.file_count)
  const minCount = Math.min(...counts)
  const maxCount = Math.max(...counts)

  const groups = groupByDepth(modules)
  const depths = Array.from(groups.keys()).sort((a, b) => a - b)

  const nodeWidths = new Map(
    modules.map((m) => [m.id, scaleWidth(m.file_count, minCount, maxCount)]),
  )

  const maxRowW = Math.max(
    ...depths.map((d) => {
      const row = groups.get(d)!
      return row.reduce((s, m) => s + nodeWidths.get(m.id)!, 0) + H_GAP * (row.length - 1)
    }),
  )

  const canvasWidth = maxRowW + CANVAS_PAD * 2
  const canvasHeight = depths.length * (NODE_H + V_GAP) - V_GAP + CANVAS_PAD * 2

  const nodes: LayoutNode[] = []
  depths.forEach((depth, rowIndex) => {
    const row = groups.get(depth)!
    const rowW = row.reduce((s, m) => s + nodeWidths.get(m.id)!, 0) + H_GAP * (row.length - 1)
    let x = (canvasWidth - rowW) / 2
    const y = CANVAS_PAD + rowIndex * (NODE_H + V_GAP)
    for (const mod of row) {
      const w = nodeWidths.get(mod.id)!
      nodes.push({ id: mod.id, x, y, width: w, height: NODE_H })
      x += w + H_GAP
    }
  })

  const visibleIds = new Set(modules.map((m) => m.id))
  const edges = aggregateEdges(
    dependencies.filter((d) => visibleIds.has(d.from) && visibleIds.has(d.to)),
  )

  return { nodes, edges, canvasWidth, canvasHeight }
}

// ── Level 1: Architecture ────────────────────────────────────────────────────

function layoutArchitecture(modules: Module[], dependencies: Dependency[]): LayoutResult {
  if (modules.length === 0) return empty()

  const segGroups = new Map<string, Module[]>()
  for (const mod of modules) {
    const seg = topSegment(mod.id)
    if (!segGroups.has(seg)) segGroups.set(seg, [])
    segGroups.get(seg)!.push(mod)
  }

  const segments = Array.from(segGroups.keys()).sort()
  const segFileCounts = new Map(
    segments.map((s) => [s, segGroups.get(s)!.reduce((n, m) => n + m.file_count, 0)]),
  )

  const counts = Array.from(segFileCounts.values())
  const minCount = Math.min(...counts)
  const maxCount = Math.max(...counts)

  const nodeWidths = new Map(
    segments.map((s) => [s, scaleWidth(segFileCounts.get(s)!, minCount, maxCount)]),
  )

  const rowW =
    segments.reduce((s, seg) => s + nodeWidths.get(seg)!, 0) + H_GAP * (segments.length - 1)
  const canvasWidth = rowW + CANVAS_PAD * 2
  const canvasHeight = NODE_H + CANVAS_PAD * 2

  let x = CANVAS_PAD
  const y = CANVAS_PAD
  const nodes: LayoutNode[] = segments.map((seg) => {
    const w = nodeWidths.get(seg)!
    const node: LayoutNode = { id: seg, x, y, width: w, height: NODE_H }
    x += w + H_GAP
    return node
  })

  const edgeWeights = new Map<string, number>()
  for (const dep of dependencies) {
    const from = topSegment(dep.from)
    const to = topSegment(dep.to)
    if (from === to) continue
    const key = `${from}\u2192${to}`
    edgeWeights.set(key, (edgeWeights.get(key) ?? 0) + 1)
  }
  const edges: LayoutEdge[] = Array.from(edgeWeights.entries()).map(([key, weight]) => {
    const arrow = key.indexOf('\u2192')
    return { fromId: key.slice(0, arrow), toId: key.slice(arrow + 1), weight }
  })

  return { nodes, edges, canvasWidth, canvasHeight }
}

// ── Level 3: Files ────────────────────────────────────────────────────────────

function layoutFiles(files: File[], selectedModuleId: string): LayoutResult {
  const moduleFiles = files
    .filter((f) => f.module === selectedModuleId)
    .sort((a, b) => b.risk_score - a.risk_score)

  if (moduleFiles.length === 0) return empty()

  const locs = moduleFiles.map((f) => f.lines)
  const minLoc = Math.min(...locs)
  const maxLoc = Math.max(...locs)

  const COLS = Math.max(1, Math.ceil(Math.sqrt(moduleFiles.length)))
  const nodes: LayoutNode[] = moduleFiles.map((f, i) => {
    const col = i % COLS
    const row = Math.floor(i / COLS)
    return {
      id: f.path,
      x: CANVAS_PAD + col * (NODE_MAX_W + H_GAP),
      y: CANVAS_PAD + row * (NODE_H + V_GAP),
      width: scaleWidth(f.lines, minLoc, maxLoc),
      height: NODE_H,
    }
  })

  const cols = Math.min(COLS, moduleFiles.length)
  const rows = Math.ceil(moduleFiles.length / COLS)
  const canvasWidth = CANVAS_PAD * 2 + cols * (NODE_MAX_W + H_GAP) - H_GAP
  const canvasHeight = CANVAS_PAD * 2 + rows * (NODE_H + V_GAP) - V_GAP

  return { nodes, edges: [], canvasWidth, canvasHeight }
}

// ── Public API ────────────────────────────────────────────────────────────────

function empty(): LayoutResult {
  return { nodes: [], edges: [], canvasWidth: 400, canvasHeight: 300 }
}

/**
 * Computes a layout for the given zoom level.
 * Level 1 (Architecture): top-level path segments.
 * Level 2 (Modules): all modules, grouped by path depth.
 * Level 3 (Files): files within the selected module (falls back to Level 2 if none selected).
 *
 * At Level 3, `selectedId` may be either a module ID or a file path.
 * When it is a file path, the layout shows that file's parent module's files
 * so the canvas is never blank and the selected file node is highlighted.
 */
export function computeLayout(
  modules: Module[],
  dependencies: Dependency[],
  files: File[],
  zoomLevel: ZoomLevel,
  selectedId: string | null,
): LayoutResult {
  switch (zoomLevel) {
    case 1:
      return layoutArchitecture(modules, dependencies)
    case 2:
      return layoutModules(modules, dependencies)
    case 3: {
      if (selectedId === null) return layoutModules(modules, dependencies)
      // If selectedId is a file path, use its parent module for the layout.
      const fileEntry = files.find((f) => f.path === selectedId)
      const moduleId = fileEntry ? fileEntry.module : selectedId
      return layoutFiles(files, moduleId)
    }
  }
}

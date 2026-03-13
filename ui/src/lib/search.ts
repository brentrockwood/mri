import type { Analysis } from '../types/analysis'
import type { ZoomLevel } from '../layout/types'

const MAX_HITS_PER_KIND = 5

// ── Hit types ─────────────────────────────────────────────────────────────────

export type SearchHit =
  | { kind: 'module'; id: string; label: string; moduleId: string; zoomLevel: ZoomLevel }
  | {
      kind: 'file'
      path: string
      label: string
      detail: string
      moduleId: string
      zoomLevel: ZoomLevel
    }
  | {
      kind: 'finding'
      id: string
      label: string
      detail: string
      moduleId: string
      zoomLevel: ZoomLevel
    }

/** Stable unique key for a search hit (for React lists). */
export function hitKey(hit: SearchHit): string {
  if (hit.kind === 'module') return `m:${hit.id}`
  if (hit.kind === 'file') return `f:${hit.path}`
  return `r:${hit.id}`
}

// ── Search ────────────────────────────────────────────────────────────────────

/**
 * Returns matching hits across modules, files, and findings.
 * Case-insensitive substring match. At most MAX_HITS_PER_KIND per category.
 * Returns an empty array when the query is blank.
 */
export function search(query: string, analysis: Analysis): SearchHit[] {
  const q = query.trim().toLowerCase()
  if (!q) return []

  const hits: SearchHit[] = []

  let moduleCount = 0
  for (const m of analysis.modules) {
    if (moduleCount >= MAX_HITS_PER_KIND) break
    if (m.id.toLowerCase().includes(q)) {
      hits.push({ kind: 'module', id: m.id, label: m.id, moduleId: m.id, zoomLevel: 2 })
      moduleCount++
    }
  }

  let fileCount = 0
  for (const f of analysis.files) {
    if (fileCount >= MAX_HITS_PER_KIND) break
    if (f.path.toLowerCase().includes(q)) {
      const name = f.path.split('/').pop() ?? f.path
      hits.push({
        kind: 'file',
        path: f.path,
        label: name,
        detail: f.path,
        moduleId: f.module,
        zoomLevel: 3,
      })
      fileCount++
    }
  }

  let findingCount = 0
  for (const r of analysis.risks) {
    if (findingCount >= MAX_HITS_PER_KIND) break
    if (
      r.title.toLowerCase().includes(q) ||
      r.description.toLowerCase().includes(q)
    ) {
      hits.push({
        kind: 'finding',
        id: r.id,
        label: r.title,
        detail: r.file,
        moduleId: r.module,
        zoomLevel: 2,
      })
      findingCount++
    }
  }

  return hits
}

/**
 * Returns the set of module IDs that contain at least one search hit.
 * Used to highlight matching nodes and dim the rest in the map.
 * Returns null when the query is blank (no active filter).
 *
 * Performs a full scan (no per-kind limit) so no matching module is omitted
 * due to MAX_HITS_PER_KIND truncation in search().
 */
export function matchingModuleIds(
  query: string,
  analysis: Analysis,
): Set<string> | null {
  const q = query.trim().toLowerCase()
  if (!q) return null

  const ids = new Set<string>()

  for (const m of analysis.modules) {
    if (m.id.toLowerCase().includes(q)) ids.add(m.id)
  }
  for (const f of analysis.files) {
    if (f.path.toLowerCase().includes(q)) ids.add(f.module)
  }
  for (const r of analysis.risks) {
    if (r.title.toLowerCase().includes(q) || r.description.toLowerCase().includes(q)) {
      ids.add(r.module)
    }
  }

  return ids
}

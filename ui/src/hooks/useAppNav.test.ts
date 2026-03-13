import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest'
import { parseHash, buildHash } from './useAppNav'

// ── parseHash ────────────────────────────────────────────────────────────────

describe('parseHash', () => {
  it('parses zoom level and selected ID', () => {
    expect(parseHash('#z=2&s=internal%2Fanalysis')).toEqual({
      zoomLevel: 2,
      selectedId: 'internal/analysis',
    })
  })

  it('returns null selectedId when s param is absent', () => {
    expect(parseHash('#z=1')).toEqual({ zoomLevel: 1, selectedId: null })
  })

  it('returns null selectedId when s param is empty string', () => {
    expect(parseHash('#z=3&s=')).toEqual({ zoomLevel: 3, selectedId: null })
  })

  it('defaults to zoomLevel 2 for empty hash', () => {
    expect(parseHash('')).toEqual({ zoomLevel: 2, selectedId: null })
  })

  it('defaults to zoomLevel 2 for malformed z value', () => {
    expect(parseHash('#z=banana')).toEqual({ zoomLevel: 2, selectedId: null })
  })

  it('defaults to zoomLevel 2 for out-of-range z value', () => {
    expect(parseHash('#z=5')).toEqual({ zoomLevel: 2, selectedId: null })
  })

  it('handles hash without leading #', () => {
    expect(parseHash('z=2&s=cmd%2Fmain')).toEqual({
      zoomLevel: 2,
      selectedId: 'cmd/main',
    })
  })

  it('decodes percent-encoded slashes in module ID', () => {
    expect(parseHash('#z=3&s=internal%2Freport%2Freport.go')).toEqual({
      zoomLevel: 3,
      selectedId: 'internal/report/report.go',
    })
  })
})

// ── buildHash ────────────────────────────────────────────────────────────────

describe('buildHash', () => {
  it('encodes zoom level without selection', () => {
    expect(buildHash(1, null)).toBe('#z=1')
  })

  it('encodes zoom level and selected ID', () => {
    expect(buildHash(2, 'internal/analysis')).toBe('#z=2&s=internal%2Fanalysis')
  })

  it('percent-encodes slashes in module IDs', () => {
    expect(buildHash(3, 'cmd/repo-mri')).toBe('#z=3&s=cmd%2Frepo-mri')
  })
})

// ── round-trip ───────────────────────────────────────────────────────────────

describe('round-trip', () => {
  it('parseHash(buildHash(...)) returns original values', () => {
    const cases: Array<[import('../layout/types').ZoomLevel, string | null]> = [
      [1, null],
      [2, 'internal/analysis'],
      [3, 'internal/report/report.go'],
    ]
    for (const [level, id] of cases) {
      expect(parseHash(buildHash(level, id))).toEqual({
        zoomLevel: level,
        selectedId: id,
      })
    }
  })
})

// ── popstate sync ─────────────────────────────────────────────────────────────

describe('popstate integration', () => {
  beforeEach(() => {
    // Reset hash to a known state
    history.replaceState(null, '', '#z=2')
  })

  afterEach(() => {
    history.replaceState(null, '', '#z=2')
    vi.restoreAllMocks()
  })

  it('window.location.hash reflects buildHash output', () => {
    const hash = buildHash(2, 'internal/analysis')
    history.pushState(null, '', hash)
    expect(parseHash(window.location.hash)).toEqual({
      zoomLevel: 2,
      selectedId: 'internal/analysis',
    })
  })
})

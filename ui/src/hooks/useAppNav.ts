import { useCallback, useEffect, useState } from 'react'
import type { ZoomLevel } from '../layout/types'

// ── URL serialisation ─────────────────────────────────────────────────────────

/** Parses zoom level and selected ID from a hash string (e.g. "#z=2&s=pkg%2Ffoo"). */
export function parseHash(hash: string): { zoomLevel: ZoomLevel; selectedId: string | null } {
  const params = new URLSearchParams(hash.replace(/^#/, ''))
  const z = Number(params.get('z'))
  const zoomLevel: ZoomLevel = z === 1 || z === 2 || z === 3 ? z : 2
  const s = params.get('s')
  return { zoomLevel, selectedId: s !== null && s !== '' ? s : null }
}

/** Serialises zoom level and optional selected ID to a hash string. */
export function buildHash(zoomLevel: ZoomLevel, selectedId: string | null): string {
  const params = new URLSearchParams()
  params.set('z', String(zoomLevel))
  if (selectedId !== null) params.set('s', selectedId)
  return `#${params.toString()}`
}

// ── Hook ──────────────────────────────────────────────────────────────────────

export interface AppNavState {
  zoomLevel: ZoomLevel
  selectedId: string | null
  setZoomLevel: (level: ZoomLevel) => void
  select: (id: string | null) => void
  /** Atomically sets both zoom level and selection in a single history push. */
  selectAndZoom: (id: string | null, level: ZoomLevel) => void
}

/**
 * Manages zoom level and selected node ID as URL-hash-synced state.
 *
 * - On mount: initialises from `window.location.hash`.
 * - On state change: pushes a new history entry so the back button works.
 * - On `popstate`: restores state from the URL (browser back/forward).
 *
 * Hash format: `#z=<level>&s=<encoded-id>`
 * Uses hash routing (not query strings) so it works correctly on `file://` URLs.
 */
export function useAppNav(): AppNavState {
  const [{ zoomLevel, selectedId }, setNav] = useState(() =>
    parseHash(window.location.hash),
  )

  // Sync state back when the browser navigates (back/forward buttons).
  useEffect(() => {
    function onPopState() {
      setNav(parseHash(window.location.hash))
    }
    window.addEventListener('popstate', onPopState)
    return () => window.removeEventListener('popstate', onPopState)
  }, [])

  const setZoomLevel = useCallback((level: ZoomLevel) => {
    setNav((prev) => {
      const next = { zoomLevel: level, selectedId: prev.selectedId }
      history.pushState(next, '', buildHash(level, prev.selectedId))
      return next
    })
  }, [])

  const select = useCallback((id: string | null) => {
    setNav((prev) => {
      const next = { zoomLevel: prev.zoomLevel, selectedId: id }
      history.pushState(next, '', buildHash(prev.zoomLevel, id))
      return next
    })
  }, [])

  const selectAndZoom = useCallback((id: string | null, level: ZoomLevel) => {
    setNav(() => {
      const next = { zoomLevel: level, selectedId: id }
      history.pushState(next, '', buildHash(level, id))
      return next
    })
  }, [])

  return { zoomLevel, selectedId, setZoomLevel, select, selectAndZoom }
}

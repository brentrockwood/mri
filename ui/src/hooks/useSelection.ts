import { useCallback, useState } from 'react'

export interface SelectionState {
  selectedId: string | null
}

/** Manages a single selected node ID. Deselect by calling select(null). */
export function useSelection() {
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const select = useCallback((id: string | null) => setSelectedId(id), [])

  return { selectedId, select }
}

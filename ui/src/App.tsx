import { useCallback, useEffect, useMemo, useState } from 'react'
import type React from 'react'
import { useAnalysis } from './hooks/useAnalysis'
import { useAppNav } from './hooks/useAppNav'
import { useZoom } from './hooks/useZoom'
import { computeLayout } from './layout/layered'
import { MapCanvas } from './components/MapCanvas'
import { StatusBar } from './components/StatusBar'
import { Tooltip } from './components/Tooltip'
import { Inspector } from './components/Inspector'
import { SearchBar } from './components/SearchBar'
import { matchingModuleIds } from './lib/search'
import type { SearchHit } from './lib/search'
import type { ZoomLevel } from './layout/types'

export function App() {
  const analysis = useAnalysis()
  const { zoomLevel, selectedId, select, selectAndZoom } = useAppNav()
  const [hoveredId, setHoveredId] = useState<string | null>(null)
  const [mousePos, setMousePos] = useState({ x: 0, y: 0 })
  const [searchQuery, setSearchQuery] = useState('')
  // ID of a node to centre on after the next layout update
  const [pendingCenterId, setPendingCenterId] = useState<string | null>(null)
  const [inspectorOpen, setInspectorOpen] = useState(false)
  const [layoutAvailableWidth, setLayoutAvailableWidth] = useState<number | undefined>(undefined)

  const layout = useMemo(
    () => computeLayout(analysis.modules, analysis.dependencies, analysis.files, zoomLevel, selectedId, layoutAvailableWidth),
    [analysis.modules, analysis.dependencies, analysis.files, zoomLevel, selectedId, layoutAvailableWidth],
  )

  const zoom = useZoom(layout.canvasWidth, layout.canvasHeight)
  const { handleMouseMove: zoomHandleMouseMove, centerOn } = zoom

  // Centre the map on pendingCenterId once the node appears in the layout.
  useEffect(() => {
    if (pendingCenterId === null) return
    const node = layout.nodes.find((n) => n.id === pendingCenterId)
    if (!node) return
    centerOn(node.x + node.width / 2, node.y + node.height / 2)
    setPendingCenterId(null)
  }, [layout, pendingCenterId, centerOn])

  /** Navigate to an id+level and queue centering. Used for all click/search/inspector nav. */
  const navigateTo = useCallback((id: string, level: ZoomLevel) => {
    selectAndZoom(id, level)
    setPendingCenterId(id)
  }, [selectAndZoom])

  // Single-click select only (no zoom).
  function handleNodeClick(id: string) {
    setSearchQuery('')
    if (zoomLevel >= 2) {
      select(id)
      setInspectorOpen(true)
    }
    // At z=1, single click is no-op for inspector/selection
  }

  const handleNodeDoubleClick = useCallback((id: string) => {
    setSearchQuery('')
    if (zoomLevel === 1) {
      const newWidth = inspectorOpen ? window.innerWidth - 360 : undefined
      setLayoutAvailableWidth(newWidth)
      selectAndZoom(null, 2)
    } else if (zoomLevel === 2) {
      const newWidth = inspectorOpen ? window.innerWidth - 360 : undefined
      setLayoutAvailableWidth(newWidth)
      navigateTo(id, 3)
      setInspectorOpen(true)
    }
    // z=3: double-click is same as single-click (already handled by onClick)
  }, [zoomLevel, inspectorOpen, selectAndZoom, navigateTo])

  function handleSearchSelect(hit: SearchHit) {
    setSearchQuery('')
    if (hit.kind === 'file') {
      navigateTo(hit.path, 3)
    } else if (hit.kind === 'module') {
      navigateTo(hit.id, 3)
    } else {
      // Finding: navigate to its module at z=3.
      navigateTo(hit.moduleId, 3)
    }
  }

  /**
   * Handles status-bar tab clicks. Adjusts selection intelligently:
   * - Going to z=1: clear selection (no inspector at arch level).
   * - Going to z=2: if a file is selected, switch to its parent module.
   * - Going to z=3: keep module selection, or keep file selection.
   */
  function handleLevelChange(newLevel: ZoomLevel) {
    if (newLevel === zoomLevel) return
    const newWidth = inspectorOpen ? window.innerWidth - 360 : undefined
    setLayoutAvailableWidth(newWidth)
    if (newLevel === 1) {
      setInspectorOpen(false)
      selectAndZoom(null, 1)
    } else if (newLevel === 2) {
      const selectedFile = analysis.files.find((f) => f.path === selectedId)
      const newId = selectedFile ? selectedFile.module : selectedId
      selectAndZoom(newId, 2)
    } else {
      selectAndZoom(selectedId, 3)
    }
  }

  const handleMouseMove = useCallback(
    (e: React.MouseEvent<SVGSVGElement>) => {
      zoomHandleMouseMove(e)
      setMousePos({ x: e.clientX, y: e.clientY })
    },
    [zoomHandleMouseMove],
  )

  const handleNodeHover = useCallback((id: string | null) => {
    setHoveredId(id)
  }, [])

  const handleBackgroundClick = useCallback(() => {
    select(null)
    setInspectorOpen(false)
    setSearchQuery('')
  }, [select])

  // Show tooltip only for Level-2 module nodes while inspector is not open
  const tooltipModuleId =
    hoveredId !== null &&
    zoomLevel === 2 &&
    !inspectorOpen &&
    analysis.modules.some((m) => m.id === hoveredId)
      ? hoveredId
      : null

  const matchingIds = useMemo(
    () => matchingModuleIds(searchQuery, analysis),
    [searchQuery, analysis],
  )

  return (
    <div className="flex flex-col w-screen h-screen bg-canvas overflow-hidden">
      <div className="flex-1 min-h-0 relative">
        <MapCanvas
          layout={layout}
          analysis={analysis}
          isArchLevel={zoomLevel === 1}
          isFilesLevel={zoomLevel === 3}
          viewBox={zoom.viewBox}
          selectedId={selectedId}
          matchingIds={matchingIds}
          svgRef={zoom.svgRef}
          onNodeClick={handleNodeClick}
          onNodeDoubleClick={handleNodeDoubleClick}
          onNodeHover={handleNodeHover}
          onBackgroundClick={handleBackgroundClick}
          onMouseDown={zoom.handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={zoom.stopPan}
        />

        <SearchBar
          query={searchQuery}
          onQueryChange={setSearchQuery}
          analysis={analysis}
          onSelect={handleSearchSelect}
        />

        {inspectorOpen && (
          <Inspector
            selectedId={selectedId}
            analysis={analysis}
            onClose={() => { select(null); setInspectorOpen(false) }}
            onNavigate={navigateTo}
          />
        )}

        {zoomLevel >= 2 && (
          <button
            onClick={() => setInspectorOpen(!inspectorOpen)}
            style={{ right: inspectorOpen ? 360 : 0 }}
            className="absolute top-16 z-[99] bg-panel border border-border-subtle text-text-muted cursor-pointer px-1 py-3 rounded-l-[4px] text-sm font-mono hover:[box-shadow:0_0_8px_rgba(147,197,253,0.3)] transition-all duration-150 shadow-[var(--shadow-panel)]"
            aria-label={inspectorOpen ? 'Close inspector' : 'Open inspector'}
          >
            {inspectorOpen ? '‹' : '›'}
          </button>
        )}

        {tooltipModuleId !== null && (
          <Tooltip
            moduleId={tooltipModuleId}
            analysis={analysis}
            mouseX={mousePos.x}
            mouseY={mousePos.y}
          />
        )}
      </div>
      <StatusBar
        level={zoomLevel}
        selectedId={selectedId}
        analysis={analysis}
        onLevelChange={handleLevelChange}
      />
    </div>
  )
}

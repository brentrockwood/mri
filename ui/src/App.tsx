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
  const { zoomLevel, setZoomLevel, selectedId, select } = useAppNav()
  const [hoveredId, setHoveredId] = useState<string | null>(null)
  const [mousePos, setMousePos] = useState({ x: 0, y: 0 })
  const [searchQuery, setSearchQuery] = useState('')
  // ID of a node to centre on after the next layout update
  const [pendingCenterId, setPendingCenterId] = useState<string | null>(null)

  const layout = useMemo(
    () =>
      computeLayout(analysis.modules, analysis.dependencies, analysis.files, zoomLevel, selectedId),
    [analysis.modules, analysis.dependencies, analysis.files, zoomLevel, selectedId],
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

  function handleNodeClick(id: string) {
    select(id)
    setSearchQuery('')
    // Switch to files view when a module is clicked from modules view
    if (zoomLevel === 2) setZoomLevel(3)
  }

  function handleSearchSelect(hit: SearchHit) {
    const targetZoom: ZoomLevel = hit.zoomLevel
    select(hit.moduleId)
    if (zoomLevel !== targetZoom) setZoomLevel(targetZoom)
    setPendingCenterId(hit.moduleId)
    // SearchBar is controlled — clearing via onQueryChange also clears the input.
    setSearchQuery('')
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
    setSearchQuery('')
  }, [select])

  // Show tooltip only for Level-2 module nodes while no inspector is open
  const tooltipModuleId =
    hoveredId !== null &&
    zoomLevel === 2 &&
    selectedId === null &&
    analysis.modules.some((m) => m.id === hoveredId)
      ? hoveredId
      : null

  const matchingIds = useMemo(
    () => matchingModuleIds(searchQuery, analysis),
    [searchQuery, analysis],
  )

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        width: '100vw',
        height: '100vh',
        background: '#0f172a',
        overflow: 'hidden',
      }}
    >
      <div style={{ flex: 1, minHeight: 0, position: 'relative' }}>
        <MapCanvas
          layout={layout}
          analysis={analysis}
          isArchLevel={zoomLevel === 1}
          viewBox={zoom.viewBox}
          selectedId={selectedId}
          matchingIds={matchingIds}
          svgRef={zoom.svgRef}
          onNodeClick={handleNodeClick}
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

        {selectedId !== null && (
          <Inspector
            selectedId={selectedId}
            analysis={analysis}
            onClose={() => select(null)}
          />
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
        onLevelChange={setZoomLevel}
      />
    </div>
  )
}

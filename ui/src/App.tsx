import { useCallback, useMemo, useState } from 'react'
import type React from 'react'
import { useAnalysis } from './hooks/useAnalysis'
import { useAppNav } from './hooks/useAppNav'
import { useZoom } from './hooks/useZoom'
import { computeLayout } from './layout/layered'
import { MapCanvas } from './components/MapCanvas'
import { StatusBar } from './components/StatusBar'
import { Tooltip } from './components/Tooltip'
import { Inspector } from './components/Inspector'

export function App() {
  const analysis = useAnalysis()
  const { zoomLevel, setZoomLevel, selectedId, select } = useAppNav()
  const [hoveredId, setHoveredId] = useState<string | null>(null)
  const [mousePos, setMousePos] = useState({ x: 0, y: 0 })

  const layout = useMemo(
    () =>
      computeLayout(analysis.modules, analysis.dependencies, analysis.files, zoomLevel, selectedId),
    [analysis.modules, analysis.dependencies, analysis.files, zoomLevel, selectedId],
  )

  const zoom = useZoom(layout.canvasWidth, layout.canvasHeight)
  const { handleMouseMove: zoomHandleMouseMove } = zoom

  function handleNodeClick(id: string) {
    select(id)
    // Switch to files view when a module is clicked from modules view
    if (zoomLevel === 2) setZoomLevel(3)
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
  }, [select])

  // Show tooltip only for Level-2 module nodes while no inspector is open
  const tooltipModuleId =
    hoveredId !== null &&
    zoomLevel === 2 &&
    selectedId === null &&
    analysis.modules.some((m) => m.id === hoveredId)
      ? hoveredId
      : null

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
          svgRef={zoom.svgRef}
          onNodeClick={handleNodeClick}
          onNodeHover={handleNodeHover}
          onBackgroundClick={handleBackgroundClick}
          onMouseDown={zoom.handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={zoom.stopPan}
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
      <StatusBar level={zoomLevel} analysis={analysis} onLevelChange={setZoomLevel} />
    </div>
  )
}

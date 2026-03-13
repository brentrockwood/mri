import { useMemo, useState } from 'react'
import { useAnalysis } from './hooks/useAnalysis'
import { useSelection } from './hooks/useSelection'
import { useZoom } from './hooks/useZoom'
import { computeLayout } from './layout/layered'
import type { ZoomLevel } from './layout/types'
import { MapCanvas } from './components/MapCanvas'
import { StatusBar } from './components/StatusBar'

export function App() {
  const analysis = useAnalysis()
  const { selectedId, select } = useSelection()
  const [zoomLevel, setZoomLevel] = useState<ZoomLevel>(2)

  const layout = useMemo(
    () =>
      computeLayout(analysis.modules, analysis.dependencies, analysis.files, zoomLevel, selectedId),
    [analysis.modules, analysis.dependencies, analysis.files, zoomLevel, selectedId],
  )

  const zoom = useZoom(layout.canvasWidth, layout.canvasHeight)

  function handleNodeClick(id: string) {
    select(id)
    // Switch to files view when a module is clicked from modules view
    if (zoomLevel === 2) setZoomLevel(3)
  }

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
      <div style={{ flex: 1, minHeight: 0 }}>
        <MapCanvas
          layout={layout}
          analysis={analysis}
          isArchLevel={zoomLevel === 1}
          viewBox={zoom.viewBox}
          selectedId={selectedId}
          onNodeClick={handleNodeClick}
          onNodeHover={() => {}}
          onWheel={zoom.handleWheel}
          onMouseDown={zoom.handleMouseDown}
          onMouseMove={zoom.handleMouseMove}
          onMouseUp={zoom.stopPan}
        />
      </div>
      <StatusBar level={zoomLevel} analysis={analysis} onLevelChange={setZoomLevel} />
    </div>
  )
}

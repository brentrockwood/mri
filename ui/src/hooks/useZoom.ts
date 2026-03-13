import { useCallback, useEffect, useRef, useState } from 'react'
import type React from 'react'

export interface ViewBox {
  x: number
  y: number
  width: number
  height: number
}

const ZOOM_FACTOR = 1.15
const MIN_SCALE = 0.1
const MAX_SCALE = 10

/**
 * Manages SVG viewBox state for pan and continuous scroll-wheel zoom.
 *
 * Returns a `svgRef` that must be attached to the SVG element so that a
 * non-passive native wheel listener can call `preventDefault()` — React's
 * synthetic onWheel is passive in Chrome and cannot prevent page scroll.
 */
export function useZoom(canvasWidth: number, canvasHeight: number) {
  const [viewBox, setViewBox] = useState<ViewBox>(() => ({
    x: 0,
    y: 0,
    width: canvasWidth,
    height: canvasHeight,
  }))

  // Keep a ref in sync so event handlers always read the latest viewBox
  const vbRef = useRef(viewBox)
  useEffect(() => {
    vbRef.current = viewBox
  }, [viewBox])

  // Reset viewBox when the canvas content changes (zoom-level switch)
  useEffect(() => {
    const vb = { x: 0, y: 0, width: canvasWidth, height: canvasHeight }
    vbRef.current = vb
    setViewBox(vb)
  }, [canvasWidth, canvasHeight])

  // Keep canvas dimensions accessible inside the native wheel handler without
  // re-attaching the listener on every dimension change.
  const dimsRef = useRef({ canvasWidth, canvasHeight })
  useEffect(() => {
    dimsRef.current = { canvasWidth, canvasHeight }
  }, [canvasWidth, canvasHeight])

  // The SVG element ref — consumers must attach this to the <svg> element.
  const svgRef = useRef<SVGSVGElement>(null)

  // Attach a non-passive native wheel listener so preventDefault() works.
  useEffect(() => {
    const svg = svgRef.current
    if (!svg) return

    function onWheel(e: WheelEvent) {
      e.preventDefault()
      const { canvasWidth: cw, canvasHeight: ch } = dimsRef.current
      const svgRect = svg!.getBoundingClientRect()
      const mx = (e.clientX - svgRect.left) / svgRect.width
      const my = (e.clientY - svgRect.top) / svgRect.height
      const factor = e.deltaY > 0 ? ZOOM_FACTOR : 1 / ZOOM_FACTOR
      setViewBox((prev) => {
        const curScale = cw / prev.width
        const newScale = Math.max(MIN_SCALE, Math.min(MAX_SCALE, curScale / factor))
        const newW = cw / newScale
        const newH = ch / newScale
        const vbMx = prev.x + mx * prev.width
        const vbMy = prev.y + my * prev.height
        return { x: vbMx - mx * newW, y: vbMy - my * newH, width: newW, height: newH }
      })
    }

    svg.addEventListener('wheel', onWheel, { passive: false })
    return () => svg.removeEventListener('wheel', onWheel)
  }, []) // Attached once; dimsRef and setViewBox are always current

  // ── Pan ───────────────────────────────────────────────────────────────────

  const isPanning = useRef(false)
  const panOrigin = useRef({ clientX: 0, clientY: 0, vbX: 0, vbY: 0 })

  const handleMouseDown = useCallback((e: React.MouseEvent<SVGSVGElement>) => {
    if (e.button !== 0) return
    isPanning.current = true
    panOrigin.current = {
      clientX: e.clientX,
      clientY: e.clientY,
      vbX: vbRef.current.x,
      vbY: vbRef.current.y,
    }
  }, [])

  const handleMouseMove = useCallback((e: React.MouseEvent<SVGSVGElement>) => {
    if (!isPanning.current) return
    const svgRect = e.currentTarget.getBoundingClientRect()
    const vb = vbRef.current
    const dx = ((panOrigin.current.clientX - e.clientX) / svgRect.width) * vb.width
    const dy = ((panOrigin.current.clientY - e.clientY) / svgRect.height) * vb.height
    setViewBox({ ...vb, x: panOrigin.current.vbX + dx, y: panOrigin.current.vbY + dy })
  }, [])

  const stopPan = useCallback(() => {
    isPanning.current = false
  }, [])

  return { viewBox, isPanning, svgRef, handleMouseDown, handleMouseMove, stopPan }
}

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
 * Reset when canvas dimensions change (e.g., after a zoom-level switch).
 */
export function useZoom(canvasWidth: number, canvasHeight: number) {
  const initialVb = (): ViewBox => ({ x: 0, y: 0, width: canvasWidth, height: canvasHeight })

  const [viewBox, setViewBox] = useState<ViewBox>(initialVb)
  // Keep a ref in sync so event handlers always read the latest value
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

  const handleWheel = useCallback(
    (e: React.WheelEvent<SVGSVGElement>) => {
      e.preventDefault()
      const svgRect = e.currentTarget.getBoundingClientRect()
      const mx = (e.clientX - svgRect.left) / svgRect.width
      const my = (e.clientY - svgRect.top) / svgRect.height
      const factor = e.deltaY > 0 ? ZOOM_FACTOR : 1 / ZOOM_FACTOR
      setViewBox((prev) => {
        const curScale = canvasWidth / prev.width
        const newScale = Math.max(MIN_SCALE, Math.min(MAX_SCALE, curScale / factor))
        const newW = canvasWidth / newScale
        const newH = canvasHeight / newScale
        const vbMx = prev.x + mx * prev.width
        const vbMy = prev.y + my * prev.height
        return { x: vbMx - mx * newW, y: vbMy - my * newH, width: newW, height: newH }
      })
    },
    [canvasWidth, canvasHeight],
  )

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

  return { viewBox, isPanning, handleWheel, handleMouseDown, handleMouseMove, stopPan }
}

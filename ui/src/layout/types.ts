/** A positioned node in the layout graph. */
export interface LayoutNode {
  id: string
  x: number
  y: number
  width: number
  height: number
}

/** A weighted directed edge in the layout graph. */
export interface LayoutEdge {
  fromId: string
  toId: string
  /** Total number of dependency declarations between this pair. */
  weight: number
}

/** Full output of the layout algorithm. */
export interface LayoutResult {
  nodes: LayoutNode[]
  edges: LayoutEdge[]
  canvasWidth: number
  canvasHeight: number
}

/** Information-zoom level. Determines which nodes are rendered. */
export type ZoomLevel = 1 | 2 | 3

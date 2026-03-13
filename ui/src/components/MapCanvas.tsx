import { useId } from 'react'
import type { Analysis, Module, Risk } from '../types/analysis'
import type { LayoutEdge, LayoutNode, LayoutResult } from '../layout/types'
import type { ViewBox } from '../hooks/useZoom'
import { complexityColor, hasHighSeverityRisk } from '../lib/risk'
import { topSegment } from '../layout/layered'

const LABEL_MAX = 22
const DIM_OPACITY = 0.2

function truncate(s: string, max: number): string {
  return s.length > max ? `${s.slice(0, max - 1)}\u2026` : s
}

/** Display label for a node: last path segment, truncated. */
function nodeLabel(id: string): string {
  const last = id.split('/').pop() ?? id
  return truncate(last, LABEL_MAX)
}

/** Complexity score for a node ID (returns 0 for Level 1 virtual nodes). */
function scoreFor(nodeId: string, modules: Module[], isArchLevel: boolean): number {
  if (!isArchLevel) {
    return modules.find((m) => m.id === nodeId)?.complexity_score ?? 0
  }
  // Architecture level: average complexity of constituent modules
  const group = modules.filter((m) => topSegment(m.id) === nodeId)
  if (group.length === 0) return 0
  return group.reduce((s, m) => s + m.complexity_score, 0) / group.length
}

/** Whether a node has a high-severity risk (handles both Module and Arch level). */
function glowFor(nodeId: string, risks: Risk[], isArchLevel: boolean): boolean {
  if (!isArchLevel) return hasHighSeverityRisk(nodeId, risks)
  return risks.some((r) => topSegment(r.module) === nodeId && r.severity === 'high')
}

/**
 * Whether a node should be dimmed based on the active search filter.
 * When matchingIds is null, no search is active and nothing is dimmed.
 * At architecture level, a segment is dimmed only if none of its constituent
 * modules appear in matchingIds.
 */
function dimFor(
  nodeId: string,
  matchingIds: Set<string> | null,
  isArchLevel: boolean,
): boolean {
  if (matchingIds === null) return false
  if (!isArchLevel) return !matchingIds.has(nodeId)
  // Arch level: dim unless at least one matching module belongs to this segment
  return ![...matchingIds].some((id) => topSegment(id) === nodeId)
}

// ── Edge endpoints ────────────────────────────────────────────────────────────

interface Endpoints {
  x1: number
  y1: number
  x2: number
  y2: number
}

function edgeEndpoints(from: LayoutNode, to: LayoutNode): Endpoints {
  const fcx = from.x + from.width / 2
  const tcx = to.x + to.width / 2
  if (Math.abs(from.y - to.y) > 10) {
    // Different rows: bottom→top
    if (from.y < to.y) {
      return { x1: fcx, y1: from.y + from.height, x2: tcx, y2: to.y }
    }
    return { x1: fcx, y1: from.y, x2: tcx, y2: to.y + to.height }
  }
  // Same row: centre→centre
  return {
    x1: fcx,
    y1: from.y + from.height / 2,
    x2: tcx,
    y2: to.y + to.height / 2,
  }
}

// ── Sub-components ────────────────────────────────────────────────────────────

interface EdgeProps {
  edge: LayoutEdge
  nodeMap: Map<string, LayoutNode>
  arrowId: string
}

function GraphEdge({ edge, nodeMap, arrowId }: EdgeProps) {
  const from = nodeMap.get(edge.fromId)
  const to = nodeMap.get(edge.toId)
  if (!from || !to) return null
  const { x1, y1, x2, y2 } = edgeEndpoints(from, to)
  const strokeWidth = Math.max(1, Math.min(6, edge.weight))
  return (
    <line
      x1={x1}
      y1={y1}
      x2={x2}
      y2={y2}
      stroke="#475569"
      strokeWidth={strokeWidth}
      markerEnd={`url(#${arrowId})`}
      opacity={0.8}
    />
  )
}

interface NodeProps {
  node: LayoutNode
  score: number
  glow: boolean
  glowId: string
  selected: boolean
  dimmed: boolean
  onClick: (id: string) => void
  onHover: (id: string | null) => void
}

function GraphNode({ node, score, glow, glowId, selected, dimmed, onClick, onHover }: NodeProps) {
  const { id, x, y, width, height } = node
  const fill = complexityColor(score)
  const strokeColor = selected ? '#f8fafc' : '#1e293b'
  const strokeWidth = selected ? 2.5 : 1.5
  return (
    <g
      onClick={(e) => { e.stopPropagation(); onClick(id) }}
      onMouseEnter={() => onHover(id)}
      onMouseLeave={() => onHover(null)}
      className="cursor-pointer"
      style={{ opacity: dimmed ? DIM_OPACITY : 1 }}
    >
      <rect
        x={x}
        y={y}
        width={width}
        height={height}
        rx={6}
        fill={fill}
        stroke={strokeColor}
        strokeWidth={strokeWidth}
        filter={glow ? `url(#${glowId})` : undefined}
      />
      <text
        x={x + width / 2}
        y={y + height / 2 + 4}
        textAnchor="middle"
        fontSize={12}
        fontFamily="monospace"
        fill="#1e293b"
        className="pointer-events-none select-none"
      >
        {nodeLabel(id)}
      </text>
    </g>
  )
}

// ── SVG defs ──────────────────────────────────────────────────────────────────

function SvgDefs({ arrowId, glowId }: { arrowId: string; glowId: string }) {
  return (
    <defs>
      <marker
        id={arrowId}
        markerWidth="8"
        markerHeight="6"
        refX="7"
        refY="3"
        orient="auto"
        markerUnits="strokeWidth"
      >
        <polygon points="0 0, 8 3, 0 6" fill="#475569" />
      </marker>
      <filter id={glowId} x="-50%" y="-50%" width="200%" height="200%">
        <feGaussianBlur stdDeviation="4" result="blur" />
        <feMerge>
          <feMergeNode in="blur" />
          <feMergeNode in="SourceGraphic" />
        </feMerge>
      </filter>
    </defs>
  )
}

// ── MapCanvas ────────────────────────────────────────────────────────────────

export interface MapCanvasProps {
  layout: LayoutResult
  analysis: Analysis
  isArchLevel: boolean
  viewBox: ViewBox
  selectedId: string | null
  /** Set of module IDs that match the active search; null means no active search. */
  matchingIds: Set<string> | null
  svgRef: React.RefObject<SVGSVGElement>
  onNodeClick: (id: string) => void
  onNodeHover: (id: string | null) => void
  onBackgroundClick: () => void
  onMouseDown: (e: React.MouseEvent<SVGSVGElement>) => void
  onMouseMove: (e: React.MouseEvent<SVGSVGElement>) => void
  onMouseUp: () => void
}

export function MapCanvas({
  layout,
  analysis,
  isArchLevel,
  viewBox,
  selectedId,
  matchingIds,
  svgRef,
  onNodeClick,
  onNodeHover,
  onBackgroundClick,
  onMouseDown,
  onMouseMove,
  onMouseUp,
}: MapCanvasProps) {
  const uid = useId()
  const arrowId = `arrow-${uid}`
  const glowId = `glow-${uid}`

  const { nodes, edges } = layout
  const { modules, risks } = analysis
  const nodeMap = new Map(nodes.map((n) => [n.id, n]))

  const vbStr = `${viewBox.x} ${viewBox.y} ${viewBox.width} ${viewBox.height}`

  return (
    <svg
      ref={svgRef}
      viewBox={vbStr}
      className="block w-full h-full bg-canvas"
      onClick={onBackgroundClick}
      onMouseDown={onMouseDown}
      onMouseMove={onMouseMove}
      onMouseUp={onMouseUp}
      onMouseLeave={onMouseUp}
    >
      <SvgDefs arrowId={arrowId} glowId={glowId} />

      {/* Edges drawn first so nodes render on top */}
      <g>
        {edges.map((edge) => (
          <GraphEdge
            key={`${edge.fromId}->${edge.toId}`}
            edge={edge}
            nodeMap={nodeMap}
            arrowId={arrowId}
          />
        ))}
      </g>

      <g>
        {nodes.map((node) => (
          <GraphNode
            key={node.id}
            node={node}
            score={scoreFor(node.id, modules, isArchLevel)}
            glow={glowFor(node.id, risks, isArchLevel)}
            glowId={glowId}
            selected={node.id === selectedId}
            dimmed={dimFor(node.id, matchingIds, isArchLevel)}
            onClick={onNodeClick}
            onHover={onNodeHover}
          />
        ))}
      </g>
    </svg>
  )
}

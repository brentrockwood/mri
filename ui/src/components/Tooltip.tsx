import type { Analysis } from '../types/analysis'

function complexityBand(score: number): string {
  if (score >= 0.7) return 'High'
  if (score >= 0.4) return 'Moderate'
  return 'Low'
}

export interface TooltipProps {
  moduleId: string
  analysis: Analysis
  mouseX: number
  mouseY: number
}

/**
 * Hover overlay showing module metadata.
 * Positioned via fixed coordinates derived from mouse client position.
 */
export function Tooltip({ moduleId, analysis, mouseX, mouseY }: TooltipProps) {
  const module = analysis.modules.find((m) => m.id === moduleId)
  if (!module) return null

  const loc = analysis.files
    .filter((f) => f.module === moduleId)
    .reduce((sum, f) => sum + f.lines, 0)

  const moduleRisks = analysis.risks.filter((r) => r.module === moduleId)
  const high = moduleRisks.filter((r) => r.severity === 'high').length
  const medium = moduleRisks.filter((r) => r.severity === 'medium').length
  const low = moduleRisks.filter((r) => r.severity === 'low').length
  const hasRisks = high + medium + low > 0

  return (
    <div
      style={{
        position: 'fixed',
        left: mouseX + 14,
        top: mouseY - 8,
        background: '#1e293b',
        border: '1px solid #334155',
        borderRadius: 6,
        padding: '8px 12px',
        color: '#e2e8f0',
        fontSize: 12,
        fontFamily: 'monospace',
        pointerEvents: 'none',
        zIndex: 200,
        maxWidth: 260,
        boxShadow: '0 4px 16px rgba(0,0,0,0.6)',
      }}
    >
      <div
        style={{
          fontWeight: 'bold',
          marginBottom: 4,
          color: '#f8fafc',
          wordBreak: 'break-all',
        }}
      >
        {moduleId}
      </div>
      <div style={{ color: '#94a3b8' }}>
        {module.file_count} file{module.file_count !== 1 ? 's' : ''}
        {loc > 0 ? ` · ${loc} LOC` : ''}
      </div>
      <div style={{ color: '#94a3b8' }}>
        Complexity: {complexityBand(module.complexity_score)}
      </div>
      {hasRisks && (
        <div style={{ marginTop: 6, display: 'flex', gap: 8 }}>
          {high > 0 && (
            <span style={{ color: '#f87171' }}>
              {high}H
            </span>
          )}
          {medium > 0 && (
            <span style={{ color: '#fbbf24' }}>
              {medium}M
            </span>
          )}
          {low > 0 && (
            <span style={{ color: '#94a3b8' }}>
              {low}L
            </span>
          )}
        </div>
      )}
    </div>
  )
}

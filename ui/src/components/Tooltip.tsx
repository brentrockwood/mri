import type { Analysis } from '../types/analysis'
import { complexityBand } from '../lib/risk'

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
      className="fixed bg-panel border border-border-subtle rounded-md py-2 px-3 text-text-secondary text-xs font-mono pointer-events-none z-[200] max-w-[260px] shadow-[0_4px_16px_rgba(0,0,0,0.6)]"
      style={{ left: mouseX + 14, top: mouseY - 8 }}
    >
      <div className="font-bold mb-1 text-text-primary break-all">
        {moduleId}
      </div>
      <div className="text-risk-low">
        {module.file_count} file{module.file_count !== 1 ? 's' : ''}
        {loc > 0 ? ` · ${loc} LOC` : ''}
      </div>
      <div className="text-risk-low">
        Complexity: {complexityBand(module.complexity_score)}
      </div>
      {hasRisks && (
        <div className="mt-1.5 flex gap-2">
          {high > 0 && <span className="text-risk-high">{high}H</span>}
          {medium > 0 && <span className="text-risk-med">{medium}M</span>}
          {low > 0 && <span className="text-risk-low">{low}L</span>}
        </div>
      )}
    </div>
  )
}

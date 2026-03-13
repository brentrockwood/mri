import type { Analysis } from '../types/analysis'
import type { ZoomLevel } from '../layout/types'
import { cn } from '../lib/cn'

const LEVEL_LABELS: Record<ZoomLevel, string> = {
  1: 'Architecture',
  2: 'Modules',
  3: 'Files',
}

const LEVELS: ZoomLevel[] = [1, 2, 3]

interface StatusBarProps {
  level: ZoomLevel
  selectedId: string | null
  analysis: Analysis
  onLevelChange: (level: ZoomLevel) => void
}

export function StatusBar({ level, selectedId, analysis, onLevelChange }: StatusBarProps) {
  const { repo, risks } = analysis
  const highCount = risks.filter((r) => r.severity === 'high').length
  const medCount = risks.filter((r) => r.severity === 'medium').length

  // Build breadcrumb: show the path through zoom levels up to the current view.
  const crumbs: string[] = []
  if (level >= 1) crumbs.push('Architecture')
  if (level >= 2) crumbs.push('Modules')
  if (level >= 3 && selectedId !== null) crumbs.push(selectedId)

  return (
    <div className="bg-panel border-t border-border-subtle shrink-0">
      {/* Tab row */}
      <div className="flex items-stretch">
        {LEVELS.map((l) => (
          <button
            key={l}
            onClick={() => onLevelChange(l)}
            className={cn(
              'flex-1 px-1 pt-[10px] pb-2 border-t-2 font-mono text-xs cursor-pointer transition-colors duration-150',
              l === level
                ? 'border-blue-500 bg-canvas text-text-primary'
                : 'border-transparent bg-transparent text-text-muted',
            )}
          >
            {LEVEL_LABELS[l]}
          </button>
        ))}
      </div>

      {/* Info row */}
      <div className="flex items-center gap-[10px] px-4 pb-[14px] pt-1 text-[11px] font-mono text-text-dim flex-wrap">
        {/* Breadcrumb */}
        <span>
          {crumbs.map((crumb, i) => (
            <span key={`${crumb}-${i}`}>
              {i > 0 && <span className="mx-1 text-border-subtle">›</span>}
              <span className={i === crumbs.length - 1 ? 'text-text-secondary' : 'text-text-dim'}>
                {crumb}
              </span>
            </span>
          ))}
        </span>

        <span className="text-border-subtle">|</span>

        {/* Summary counts */}
        <span>{repo.module_count} modules</span>
        <span>{repo.file_count} files</span>
        {highCount > 0 && <span className="text-risk-high">{highCount} high</span>}
        {medCount > 0 && <span className="text-risk-med">{medCount} med</span>}
      </div>
    </div>
  )
}

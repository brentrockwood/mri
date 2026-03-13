import type { Analysis } from '../types/analysis'
import type { ZoomLevel } from '../layout/types'

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
    <div
      style={{
        background: '#1e293b',
        borderTop: '1px solid #334155',
        flexShrink: 0,
      }}
    >
      {/* Tab row */}
      <div style={{ display: 'flex', alignItems: 'stretch' }}>
        {LEVELS.map((l) => (
          <button
            key={l}
            onClick={() => onLevelChange(l)}
            style={{
              flex: 1,
              padding: '10px 4px 8px',
              border: 'none',
              borderTop: l === level ? '2px solid #3b82f6' : '2px solid transparent',
              background: l === level ? '#0f172a' : 'transparent',
              color: l === level ? '#f8fafc' : '#64748b',
              fontFamily: 'monospace',
              fontSize: '12px',
              cursor: 'pointer',
              transition: 'color 0.15s, background 0.15s',
            }}
          >
            {LEVEL_LABELS[l]}
          </button>
        ))}
      </div>

      {/* Info row */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '10px',
          padding: '4px 16px 14px',
          fontSize: '11px',
          fontFamily: 'monospace',
          color: '#475569',
          flexWrap: 'wrap',
        }}
      >
        {/* Breadcrumb */}
        <span>
          {crumbs.map((crumb, i) => (
            <span key={`${crumb}-${i}`}>
              {i > 0 && <span style={{ margin: '0 4px', color: '#334155' }}>›</span>}
              <span style={{ color: i === crumbs.length - 1 ? '#cbd5e1' : '#475569' }}>
                {crumb}
              </span>
            </span>
          ))}
        </span>

        <span style={{ color: '#334155' }}>|</span>

        {/* Summary counts */}
        <span>{repo.module_count} modules</span>
        <span>{repo.file_count} files</span>
        {highCount > 0 && <span style={{ color: '#f87171' }}>{highCount} high</span>}
        {medCount > 0 && <span style={{ color: '#facc15' }}>{medCount} med</span>}
      </div>
    </div>
  )
}

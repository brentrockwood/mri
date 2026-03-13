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
  analysis: Analysis
  onLevelChange: (level: ZoomLevel) => void
}

export function StatusBar({ level, analysis, onLevelChange }: StatusBarProps) {
  const { repo, risks } = analysis
  const highCount = risks.filter((r) => r.severity === 'high').length
  const medCount = risks.filter((r) => r.severity === 'medium').length

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '12px',
        padding: '6px 16px',
        background: '#1e293b',
        borderTop: '1px solid #334155',
        fontSize: '12px',
        fontFamily: 'monospace',
        color: '#94a3b8',
        flexShrink: 0,
      }}
    >
      {/* Zoom level controls */}
      <div style={{ display: 'flex', gap: '4px' }}>
        {LEVELS.map((l) => (
          <button
            key={l}
            onClick={() => onLevelChange(l)}
            style={{
              padding: '2px 10px',
              borderRadius: '4px',
              border: '1px solid #475569',
              background: l === level ? '#3b82f6' : '#0f172a',
              color: l === level ? '#f8fafc' : '#94a3b8',
              fontFamily: 'monospace',
              fontSize: '11px',
              cursor: 'pointer',
            }}
          >
            {l} {LEVEL_LABELS[l]}
          </button>
        ))}
      </div>

      <span style={{ color: '#475569' }}>|</span>

      {/* Summary counts */}
      <span>{repo.module_count} modules</span>
      <span>{repo.file_count} files</span>
      {highCount > 0 && <span style={{ color: '#f87171' }}>{highCount} high</span>}
      {medCount > 0 && <span style={{ color: '#facc15' }}>{medCount} med</span>}
    </div>
  )
}

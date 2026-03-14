import Tabs from '@mui/material/Tabs'
import Tab from '@mui/material/Tab'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
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

  const crumbs: string[] = []
  if (level >= 1) crumbs.push('Architecture')
  if (level >= 2) crumbs.push('Modules')
  if (level >= 3 && selectedId !== null) crumbs.push(selectedId)

  return (
    <Box
      sx={{
        bgcolor: 'background.paper',
        flexShrink: 0,
        display: 'flex',
        alignItems: 'flex-end',
        gap: 2,
        px: 2,
        pb: 1.5,
      }}
    >
      {/* Zoom level tab strip */}
      <Tabs
        value={level}
        onChange={(_, v: ZoomLevel) => onLevelChange(v)}
        sx={{ alignItems: 'flex-end', minHeight: 'unset' }}
      >
        {LEVELS.map((l) => (
          <Tab key={l} label={LEVEL_LABELS[l]} value={l} disableRipple />
        ))}
      </Tabs>

      {/* Breadcrumb + summary counts */}
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          gap: 1.25,
          pb: 0.5,
          fontFamily: 'monospace',
          flexWrap: 'wrap',
        }}
      >
        <Typography component="span" sx={{ fontFamily: 'monospace', fontSize: '1rem', color: 'text.disabled' }}>
          {crumbs.map((crumb, i) => (
            <span key={`${crumb}-${i}`}>
              {i > 0 && (
                <Box component="span" sx={{ mx: 0.5, color: 'divider' }}>›</Box>
              )}
              <Box
                component="span"
                sx={{ color: i === crumbs.length - 1 ? 'text.secondary' : 'text.disabled' }}
              >
                {crumb}
              </Box>
            </span>
          ))}
        </Typography>

        <Typography component="span" sx={{ color: 'divider', fontFamily: 'monospace', fontSize: '1rem' }}>|</Typography>

        <Typography component="span" sx={{ fontFamily: 'monospace', fontSize: '1rem', color: 'text.disabled' }}>
          {repo.module_count} modules
        </Typography>
        <Typography component="span" sx={{ fontFamily: 'monospace', fontSize: '1rem', color: 'text.disabled' }}>
          {repo.file_count} files
        </Typography>
        {highCount > 0 && (
          <Typography component="span" sx={{ fontFamily: 'monospace', fontSize: '1rem', color: 'error.main' }}>
            {highCount} high
          </Typography>
        )}
        {medCount > 0 && (
          <Typography component="span" sx={{ fontFamily: 'monospace', fontSize: '1rem', color: 'warning.main' }}>
            {medCount} med
          </Typography>
        )}
      </Box>
    </Box>
  )
}

import Paper from '@mui/material/Paper'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
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
 * Uses MUI Paper for theming/elevation; fixed positioning for mouse-following.
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
    <Paper
      elevation={8}
      sx={{
        position: 'fixed',
        left: mouseX + 14,
        top: mouseY - 8,
        py: 1,
        px: 1.5,
        fontFamily: 'monospace',
        fontSize: '0.75rem',
        pointerEvents: 'none',
        zIndex: 200,
        maxWidth: 260,
        bgcolor: 'background.paper',
        border: '1px solid',
        borderColor: 'divider',
      }}
    >
      <Typography
        variant="body2"
        sx={{ fontWeight: 'bold', mb: 0.5, color: 'text.primary', wordBreak: 'break-all', fontFamily: 'monospace', fontSize: '0.75rem' }}
      >
        {moduleId}
      </Typography>
      <Typography sx={{ color: 'text.disabled', fontFamily: 'monospace', fontSize: '0.75rem' }}>
        {module.file_count} file{module.file_count !== 1 ? 's' : ''}
        {loc > 0 ? ` · ${loc} LOC` : ''}
      </Typography>
      <Typography sx={{ color: 'text.disabled', fontFamily: 'monospace', fontSize: '0.75rem' }}>
        Complexity: {complexityBand(module.complexity_score)}
      </Typography>
      {hasRisks && (
        <Box sx={{ mt: 0.75, display: 'flex', gap: 1 }}>
          {high > 0 && (
            <Typography component="span" sx={{ color: 'error.main', fontFamily: 'monospace', fontSize: '0.75rem' }}>
              {high}H
            </Typography>
          )}
          {medium > 0 && (
            <Typography component="span" sx={{ color: 'warning.main', fontFamily: 'monospace', fontSize: '0.75rem' }}>
              {medium}M
            </Typography>
          )}
          {low > 0 && (
            <Typography component="span" sx={{ color: 'text.disabled', fontFamily: 'monospace', fontSize: '0.75rem' }}>
              {low}L
            </Typography>
          )}
        </Box>
      )}
    </Paper>
  )
}

import { useCallback, useEffect, useMemo, useState } from 'react'
import Drawer from '@mui/material/Drawer'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import IconButton from '@mui/material/IconButton'
import Table from '@mui/material/Table'
import TableHead from '@mui/material/TableHead'
import TableBody from '@mui/material/TableBody'
import TableRow from '@mui/material/TableRow'
import TableCell from '@mui/material/TableCell'
import CloseIcon from '@mui/icons-material/Close'
import type { Analysis, File, Risk } from '../types/analysis'
import type { ZoomLevel } from '../layout/types'
import { complexityColor, riskSeverity, severityColorClass } from '../lib/risk'
import { copyToClipboard, detectWindowsPaths, githubUrl, vscodeUrl } from '../lib/deeplinks'
import { cn } from '../lib/cn'

// ── Constants ─────────────────────────────────────────────────────────────────

const SEVERITY_ORDER: Record<string, number> = { high: 0, medium: 1, low: 2 }

const DRAWER_WIDTH = 360

// ── Helpers ───────────────────────────────────────────────────────────────────

function confidencePct(confidence: number): string {
  return `${Math.round(confidence * 100)}%`
}

// ── CopyButton ────────────────────────────────────────────────────────────────

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)

  const handleClick = useCallback(async () => {
    try {
      await copyToClipboard(text)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard API unavailable (e.g. non-secure context)
    }
  }, [text])

  return (
    <button
      onClick={handleClick}
      title="Copy path"
      className={cn(
        'px-1.5 py-px text-[10px] font-mono border border-border-subtle rounded-[3px] cursor-pointer shrink-0',
        copied ? 'bg-green-800 text-green-300' : 'bg-panel text-text-muted',
      )}
    >
      {copied ? 'copied' : 'copy'}
    </button>
  )
}

// ── LinkButton ────────────────────────────────────────────────────────────────

function LinkButton({ href, label }: { href: string; label: string }) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="px-1.5 py-px text-[10px] font-mono bg-panel text-blue-400 border border-border-subtle rounded-[3px] no-underline shrink-0"
    >
      {label}
    </a>
  )
}

// ── FindingRow ────────────────────────────────────────────────────────────────

interface FindingRowProps {
  risk: Risk
  repoName: string | null
  rootPath: string | null
  isWindows: boolean
}

function FindingRow({ risk, repoName, rootPath, isWindows }: FindingRowProps) {
  const line = risk.evidence_lines?.[0]
  const ghUrl = repoName ? githubUrl(repoName, risk.file, line) : null
  const vsUrl = rootPath
    ? vscodeUrl(isWindows ? `${rootPath}\\${risk.file.replace(/\//g, '\\')}` : `${rootPath}/${risk.file}`, line)
    : null

  return (
    <Box sx={{ py: 1, borderBottom: '1px solid', borderColor: 'background.paper' }}>
      <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 0.75, mb: 0.5 }}>
        <Typography
          component="span"
          sx={{
            fontSize: '0.625rem',
            fontWeight: 'bold',
            textTransform: 'uppercase',
            flexShrink: 0,
            pt: '2px',
          }}
          className={severityColorClass(risk.severity)}
        >
          {risk.severity}
        </Typography>
        <Typography
          component="span"
          sx={{ color: 'text.secondary', fontSize: '0.75rem', flex: 1, minWidth: 0, wordBreak: 'break-word', fontFamily: 'monospace' }}
        >
          {risk.title}
        </Typography>
        <Typography
          component="span"
          sx={{ color: 'text.disabled', fontSize: '0.625rem', flexShrink: 0, fontFamily: 'monospace' }}
        >
          {confidencePct(risk.confidence)}
        </Typography>
      </Box>
      {risk.description && (
        <Typography
          sx={{ color: 'text.disabled', fontSize: '0.6875rem', mb: 0.75, lineHeight: 1.4, fontFamily: 'monospace' }}
        >
          {risk.description}
        </Typography>
      )}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, flexWrap: 'wrap' }}>
        <Typography
          component="span"
          sx={{
            color: 'text.disabled',
            fontSize: '0.625rem',
            fontFamily: 'monospace',
            flex: 1,
            minWidth: 0,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {risk.file}{line !== undefined ? `:${line}` : ''}
        </Typography>
        {ghUrl !== null && <LinkButton href={ghUrl} label="gh" />}
        {vsUrl !== null && <LinkButton href={vsUrl} label="vs" />}
        <CopyButton text={risk.file} />
      </Box>
    </Box>
  )
}

// ── FileRow ───────────────────────────────────────────────────────────────────

interface FileRowProps {
  file: File
  onNavigate: (id: string, level: ZoomLevel) => void
  repoName: string | null
  rootPath: string | null
  isWindows: boolean
}

function FileRow({ file, onNavigate, repoName, rootPath, isWindows }: FileRowProps) {
  const dotColor = complexityColor(file.risk_score)
  const ghUrl = repoName ? githubUrl(repoName, file.path) : null
  const vsUrl = rootPath
    ? vscodeUrl(isWindows ? `${rootPath}\\${file.path.replace(/\//g, '\\')}` : `${rootPath}/${file.path}`)
    : null

  return (
    <TableRow
      hover
      onClick={() => onNavigate(file.path, 3)}
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onNavigate(file.path, 3) }
      }}
      sx={{ cursor: 'pointer' }}
    >
      <TableCell>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Box
            component="span"
            sx={{ width: 8, height: 8, borderRadius: '50%', flexShrink: 0, background: dotColor }}
          />
          <Typography
            component="span"
            sx={{
              fontSize: '1.25rem',
              fontFamily: 'monospace',
              color: '#93c5fd',
              textDecoration: 'underline',
              textDecorationColor: '#334155',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {file.path.split('/').pop()}
          </Typography>
        </Box>
      </TableCell>
      <TableCell align="right">
        <Typography component="span" sx={{ color: 'text.disabled', fontSize: '1rem', fontFamily: 'monospace' }}>
          {file.lines}L
        </Typography>
      </TableCell>
      <TableCell align="right">
        <Typography
          component="span"
          sx={{ fontSize: '1rem', fontFamily: 'monospace' }}
          className={severityColorClass(riskSeverity(file.risk_score))}
        >
          {Math.round(file.risk_score * 100)}
        </Typography>
      </TableCell>
      <TableCell>
        <Box sx={{ display: 'flex', gap: 0.5 }}>
          {ghUrl !== null && <LinkButton href={ghUrl} label="gh" />}
          {vsUrl !== null && <LinkButton href={vsUrl} label="vs" />}
        </Box>
      </TableCell>
    </TableRow>
  )
}

// ── NavItem ───────────────────────────────────────────────────────────────────

function NavItem({ id, prefix, onNavigate }: { id: string; prefix: string; onNavigate: (id: string, level: ZoomLevel) => void }) {
  return (
    <Box
      onClick={() => onNavigate(id, 3)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onNavigate(id, 3) } }}
      sx={{
        fontSize: '0.6875rem',
        fontFamily: 'monospace',
        color: '#93c5fd',
        py: 0.5,
        borderBottom: '1px solid',
        borderColor: 'background.paper',
        cursor: 'pointer',
        textDecoration: 'underline',
        textDecorationColor: '#334155',
        '&:hover': { boxShadow: '0 0 8px rgba(147,197,253,0.3)' },
        transition: 'box-shadow 150ms',
      }}
    >
      {prefix} {id}
    </Box>
  )
}

// ── SectionHeader ─────────────────────────────────────────────────────────────

function SectionHeader({ children }: { children: React.ReactNode }) {
  return (
    <Typography
      sx={{
        fontSize: '1.25rem',
        fontWeight: 'bold',
        color: 'text.disabled',
        textTransform: 'uppercase',
        letterSpacing: '0.08em',
        pt: 1.5,
        pb: 0.75,
        borderBottom: '1px solid',
        borderColor: 'divider',
        mb: 0.25,
        fontFamily: 'monospace',
      }}
    >
      {children}
    </Typography>
  )
}

// ── Inspector ─────────────────────────────────────────────────────────────────

export interface InspectorProps {
  open: boolean
  selectedId: string | null
  analysis: Analysis
  onClose: () => void
  onNavigate: (id: string, level: ZoomLevel) => void
}

/**
 * Right-side detail panel implemented as a persistent MUI Drawer.
 * Displays findings, dependency list, and file table for the selected module/file.
 * Items in the dependency and file lists are clickable for navigation.
 */
export function Inspector({ open, selectedId, analysis, onClose, onNavigate }: InspectorProps) {
  const { modules, risks, files, dependencies, repo, meta } = analysis

  const isWindows = useMemo(() => detectWindowsPaths(files.map((f) => f.path)), [files])
  const githubSlug = repo.github_slug ?? null
  const rootPath = meta.root_path ?? null

  const { selectedFile, module } = useMemo(() => {
    if (selectedId === null) return { selectedFile: null, module: null }
    const sf = files.find((f) => f.path === selectedId) ?? null
    const mod = sf
      ? modules.find((m) => m.id === sf.module) ?? null
      : modules.find((m) => m.id === selectedId) ?? null
    return { selectedFile: sf, module: mod }
  }, [selectedId, files, modules])

  const displayedRisks = useMemo(
    () =>
      selectedId === null
        ? []
        : risks
            .filter((r) => selectedFile ? r.file === selectedId : r.module === selectedId)
            .sort((a, b) => (SEVERITY_ORDER[a.severity] ?? 3) - (SEVERITY_ORDER[b.severity] ?? 3)),
    [risks, selectedFile, selectedId],
  )

  const moduleFiles = useMemo(
    () => (selectedId === null || selectedFile) ? [] : files.filter((f) => f.module === selectedId).sort((a, b) => b.risk_score - a.risk_score),
    [selectedFile, files, selectedId],
  )

  const imports = useMemo(
    () => (selectedId === null || selectedFile) ? [] : dependencies.filter((d) => d.from === selectedId).map((d) => d.to),
    [selectedFile, dependencies, selectedId],
  )
  const importedBy = useMemo(
    () => (selectedId === null || selectedFile) ? [] : dependencies.filter((d) => d.to === selectedId).map((d) => d.from),
    [selectedFile, dependencies, selectedId],
  )

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  return (
    <Drawer
      variant="persistent"
      anchor="right"
      open={open}
      PaperProps={{
        sx: {
          width: DRAWER_WIDTH,
          position: 'absolute',
          top: 0,
          bottom: 0,
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        },
      }}
    >
      {/* Header */}
      <Box
        sx={{
          px: 2,
          py: 1.5,
          borderBottom: '1px solid',
          borderColor: 'divider',
          display: 'flex',
          alignItems: 'flex-start',
          gap: 1,
        }}
      >
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography
            sx={{
              fontSize: '0.8125rem',
              fontWeight: 'bold',
              color: 'text.primary',
              fontFamily: 'monospace',
              wordBreak: 'break-all',
            }}
          >
            {selectedId === null
              ? 'Inspector'
              : selectedFile
                ? selectedFile.path.split('/').pop()
                : selectedId}
          </Typography>
          {selectedFile && (
            <Typography sx={{ fontSize: '0.6875rem', color: 'text.disabled', mt: 0.25, fontFamily: 'monospace' }}>
              {selectedFile.path}
            </Typography>
          )}
          {selectedId !== null && selectedFile ? (
            <Box sx={{ mt: 0.5, display: 'flex', gap: 1.25, flexWrap: 'wrap' }}>
              <Typography component="span" sx={{ fontSize: '0.6875rem', color: 'text.disabled', fontFamily: 'monospace' }}>
                Risk{' '}
                <span className={severityColorClass(riskSeverity(selectedFile.risk_score))}>
                  {Math.round(selectedFile.risk_score * 100)}
                </span>
              </Typography>
              <Typography component="span" sx={{ fontSize: '0.6875rem', color: 'text.secondary', fontFamily: 'monospace' }}>
                {selectedFile.lines}L
              </Typography>
              {module && (
                <Typography
                  component="span"
                  onClick={() => onNavigate(module.id, 3)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onNavigate(module.id, 3) } }}
                  sx={{
                    fontSize: '0.6875rem',
                    color: '#93c5fd',
                    cursor: 'pointer',
                    textDecoration: 'underline',
                    textDecorationColor: '#334155',
                    fontFamily: 'monospace',
                    '&:hover': { boxShadow: '0 0 8px rgba(147,197,253,0.3)' },
                  }}
                >
                  in {module.id}
                </Typography>
              )}
            </Box>
          ) : selectedId !== null && module ? (
            <Box sx={{ mt: 0.5, display: 'flex', gap: 1.25 }}>
              <Typography component="span" sx={{ fontSize: '0.6875rem', color: 'text.disabled', fontFamily: 'monospace' }}>
                Risk{' '}
                <span className={severityColorClass(riskSeverity(module.risk_score))}>
                  {Math.round(module.risk_score * 100)}
                </span>
              </Typography>
              <Typography component="span" sx={{ fontSize: '0.6875rem', color: 'text.secondary', fontFamily: 'monospace' }}>
                Complexity{' '}
                <span style={{ color: '#cbd5e1' }}>{Math.round(module.complexity_score * 100)}</span>
              </Typography>
              <Typography component="span" sx={{ fontSize: '0.6875rem', color: 'text.disabled', fontFamily: 'monospace' }}>
                {module.file_count} files
              </Typography>
            </Box>
          ) : null}
        </Box>

        <IconButton
          onClick={onClose}
          aria-label="Close inspector"
          size="small"
          sx={{ flexShrink: 0, mt: -0.5 }}
        >
          <CloseIcon fontSize="small" />
        </IconButton>
      </Box>

      {/* Scrollable body */}
      <Box sx={{ flex: 1, overflowY: 'auto', px: 2, pb: 2 }}>
        {selectedId === null ? (
          <Typography sx={{ color: 'text.disabled', fontSize: '0.6875rem', py: 2, fontFamily: 'monospace' }}>
            Select a node to inspect
          </Typography>
        ) : (
          <>
            {/* Findings */}
            <SectionHeader>Findings ({displayedRisks.length})</SectionHeader>
            {displayedRisks.length === 0 ? (
              <Typography sx={{ color: 'divider', fontSize: '0.6875rem', py: 1, fontFamily: 'monospace' }}>
                No findings
              </Typography>
            ) : (
              displayedRisks.map((r) => (
                <FindingRow
                  key={r.id}
                  risk={r}
                  repoName={githubSlug}
                  rootPath={rootPath}
                  isWindows={isWindows}
                />
              ))
            )}

            {/* Imports */}
            {imports.length > 0 && (
              <>
                <SectionHeader>Imports ({imports.length})</SectionHeader>
                {imports.map((id) => (
                  <NavItem key={id} id={id} prefix="→" onNavigate={onNavigate} />
                ))}
              </>
            )}

            {/* Imported by */}
            {importedBy.length > 0 && (
              <>
                <SectionHeader>Imported by ({importedBy.length})</SectionHeader>
                {importedBy.map((id) => (
                  <NavItem key={id} id={id} prefix="←" onNavigate={onNavigate} />
                ))}
              </>
            )}

            {/* Files table */}
            {moduleFiles.length > 0 && (
              <>
                <SectionHeader>Files — sorted by risk</SectionHeader>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>name</TableCell>
                      <TableCell align="right">LOC</TableCell>
                      <TableCell align="right">risk</TableCell>
                      <TableCell />
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {moduleFiles.map((f) => (
                      <FileRow
                        key={f.path}
                        file={f}
                        onNavigate={onNavigate}
                        repoName={githubSlug}
                        rootPath={rootPath}
                        isWindows={isWindows}
                      />
                    ))}
                  </TableBody>
                </Table>
              </>
            )}
          </>
        )}
      </Box>
    </Drawer>
  )
}

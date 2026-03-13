import { useCallback, useEffect, useMemo, useState } from 'react'
import type { Analysis, File, Risk } from '../types/analysis'
import type { ZoomLevel } from '../layout/types'
import { complexityColor, riskSeverity } from '../lib/risk'
import { copyToClipboard, detectWindowsPaths, githubUrl, vscodeUrl } from '../lib/deeplinks'

// ── Helpers ───────────────────────────────────────────────────────────────────

function severityColor(severity: string): string {
  if (severity === 'high') return '#f87171'
  if (severity === 'medium') return '#fbbf24'
  return '#94a3b8'
}

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
      style={{
        padding: '1px 6px',
        fontSize: 10,
        fontFamily: 'monospace',
        background: copied ? '#166534' : '#1e293b',
        color: copied ? '#86efac' : '#64748b',
        border: '1px solid #334155',
        borderRadius: 3,
        cursor: 'pointer',
        flexShrink: 0,
      }}
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
      style={{
        padding: '1px 6px',
        fontSize: 10,
        fontFamily: 'monospace',
        background: '#1e293b',
        color: '#60a5fa',
        border: '1px solid #334155',
        borderRadius: 3,
        textDecoration: 'none',
        flexShrink: 0,
      }}
    >
      {label}
    </a>
  )
}

// ── FindingRow ────────────────────────────────────────────────────────────────

interface FindingRowProps {
  risk: Risk
  /** Full "owner/repo" GitHub slug; null for local repos (hides the GH link). */
  repoName: string | null
  /**
   * Absolute filesystem root of the repo (e.g. "/home/user/project").
   * When set, VS Code deep links are constructed as root_path + '/' + file.
   * When absent, the VS Code button is hidden (relative paths are unusable).
   */
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
    <div
      style={{
        padding: '8px 0',
        borderBottom: '1px solid #1e293b',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          gap: 6,
          marginBottom: 4,
        }}
      >
        <span
          style={{
            fontSize: 10,
            fontWeight: 'bold',
            color: severityColor(risk.severity),
            textTransform: 'uppercase',
            flexShrink: 0,
            paddingTop: 1,
          }}
        >
          {risk.severity}
        </span>
        <span style={{ color: '#e2e8f0', fontSize: 12, flex: 1, minWidth: 0, overflowWrap: 'break-word' }}>
          {risk.title}
        </span>
        <span style={{ color: '#64748b', fontSize: 10, flexShrink: 0 }}>
          {confidencePct(risk.confidence)}
        </span>
      </div>
      {risk.description && (
        <div
          style={{
            color: '#94a3b8',
            fontSize: 11,
            marginBottom: 6,
            lineHeight: 1.4,
          }}
        >
          {risk.description}
        </div>
      )}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 4,
          flexWrap: 'wrap',
        }}
      >
        <span
          style={{
            color: '#475569',
            fontSize: 10,
            fontFamily: 'monospace',
            flex: 1,
            minWidth: 0,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {risk.file}
          {line !== undefined ? `:${line}` : ''}
        </span>
        {ghUrl !== null && <LinkButton href={ghUrl} label="gh" />}
        {vsUrl !== null && <LinkButton href={vsUrl} label="vs" />}
        <CopyButton text={risk.file} />
      </div>
    </div>
  )
}

// ── FileRow ───────────────────────────────────────────────────────────────────

interface FileRowProps {
  file: File
  onNavigate: (id: string, level: ZoomLevel) => void
}

function FileRow({ file, onNavigate }: FileRowProps) {
  const dotColor = complexityColor(file.risk_score)
  return (
    <div
      onClick={() => onNavigate(file.path, 3)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === 'Enter' && onNavigate(file.path, 3)}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        padding: '5px 0',
        borderBottom: '1px solid #1e293b',
        fontSize: 11,
        fontFamily: 'monospace',
        cursor: 'pointer',
      }}
    >
      <span
        style={{
          width: 8,
          height: 8,
          borderRadius: '50%',
          background: dotColor,
          flexShrink: 0,
        }}
      />
      <span
        style={{
          flex: 1,
          color: '#93c5fd',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          textDecoration: 'underline',
          textDecorationColor: '#334155',
        }}
      >
        {file.path.split('/').pop()}
      </span>
      <span style={{ color: '#475569', flexShrink: 0 }}>{file.lines}L</span>
      <span
        style={{
          color: severityColor(riskSeverity(file.risk_score)),
          flexShrink: 0,
          width: 36,
          textAlign: 'right',
        }}
      >
        {Math.round(file.risk_score * 100)}
      </span>
    </div>
  )
}

// ── NavItem ───────────────────────────────────────────────────────────────────

/** Clickable row used for imports and imported-by lists. */
function NavItem({ id, prefix, onNavigate }: { id: string; prefix: string; onNavigate: (id: string, level: ZoomLevel) => void }) {
  return (
    <div
      onClick={() => onNavigate(id, 3)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === 'Enter' && onNavigate(id, 3)}
      style={{
        fontSize: 11,
        fontFamily: 'monospace',
        color: '#93c5fd',
        padding: '4px 0',
        borderBottom: '1px solid #1e293b',
        cursor: 'pointer',
        textDecoration: 'underline',
        textDecorationColor: '#334155',
      }}
    >
      {prefix} {id}
    </div>
  )
}

// ── SectionHeader ─────────────────────────────────────────────────────────────

function SectionHeader({ children }: { children: React.ReactNode }) {
  return (
    <div
      style={{
        fontSize: 10,
        fontWeight: 'bold',
        color: '#475569',
        textTransform: 'uppercase',
        letterSpacing: '0.08em',
        padding: '12px 0 6px',
        borderBottom: '1px solid #334155',
        marginBottom: 2,
      }}
    >
      {children}
    </div>
  )
}

// ── Inspector ─────────────────────────────────────────────────────────────────

export interface InspectorProps {
  selectedId: string
  analysis: Analysis
  onClose: () => void
  onNavigate: (id: string, level: ZoomLevel) => void
}

/**
 * Right-side detail panel shown when a module node or file is selected.
 * Displays findings, dependency list, and file table.
 * Items in the dependency and file lists are clickable for navigation.
 */
export function Inspector({ selectedId, analysis, onClose, onNavigate }: InspectorProps) {
  const { modules, risks, files, dependencies, repo, meta } = analysis

  const isWindows = detectWindowsPaths(files.map((f) => f.path))
  const githubSlug = repo.github_slug ?? null
  const rootPath = meta.root_path ?? null

  // Determine whether the selection is a module or a file.
  const { selectedFile, module } = useMemo(() => {
    const sf = files.find((f) => f.path === selectedId) ?? null
    const mod = sf
      ? modules.find((m) => m.id === sf.module) ?? null
      : modules.find((m) => m.id === selectedId) ?? null
    return { selectedFile: sf, module: mod }
  }, [selectedId, files, modules])

  const displayedRisks = useMemo(
    () =>
      risks
        .filter((r) => selectedFile ? r.file === selectedId : r.module === selectedId)
        .sort((a, b) => {
          const order = { high: 0, medium: 1, low: 2 }
          return (order[a.severity as keyof typeof order] ?? 3) -
            (order[b.severity as keyof typeof order] ?? 3)
        }),
    [risks, selectedFile, selectedId],
  )

  // Files table: shown only when viewing a module (not a file).
  const moduleFiles = useMemo(
    () => selectedFile ? [] : files.filter((f) => f.module === selectedId).sort((a, b) => b.risk_score - a.risk_score),
    [selectedFile, files, selectedId],
  )

  const imports = useMemo(
    () => selectedFile ? [] : dependencies.filter((d) => d.from === selectedId).map((d) => d.to),
    [selectedFile, dependencies, selectedId],
  )
  const importedBy = useMemo(
    () => selectedFile ? [] : dependencies.filter((d) => d.to === selectedId).map((d) => d.from),
    [selectedFile, dependencies, selectedId],
  )

  // Close on Escape key.
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  return (
    <div
      style={{
        position: 'absolute',
        top: 0,
        right: 0,
        bottom: 0,
        width: 360,
        background: '#0f172a',
        borderLeft: '1px solid #334155',
        display: 'flex',
        flexDirection: 'column',
        zIndex: 100,
        overflow: 'hidden',
      }}
    >
      {/* Header */}
      <div
        style={{
          padding: '12px 16px',
          borderBottom: '1px solid #334155',
          display: 'flex',
          alignItems: 'flex-start',
          gap: 8,
        }}
      >
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontSize: 13,
              fontWeight: 'bold',
              color: '#f8fafc',
              fontFamily: 'monospace',
              wordBreak: 'break-all',
            }}
          >
            {selectedFile ? selectedFile.path.split('/').pop() : selectedId}
          </div>
          {selectedFile && (
            <div style={{ fontSize: 11, color: '#475569', marginTop: 2, fontFamily: 'monospace' }}>
              {selectedFile.path}
            </div>
          )}
          {selectedFile ? (
            <div style={{ fontSize: 11, color: '#64748b', marginTop: 4, display: 'flex', gap: 10, flexWrap: 'wrap' }}>
              <span>
                Risk{' '}
                <span style={{ color: severityColor(riskSeverity(selectedFile.risk_score)) }}>
                  {Math.round(selectedFile.risk_score * 100)}
                </span>
              </span>
              <span style={{ color: '#cbd5e1' }}>{selectedFile.lines}L</span>
              {module && (
                <span
                  onClick={() => onNavigate(module.id, 3)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={(e) => e.key === 'Enter' && onNavigate(module.id, 3)}
                  style={{ color: '#93c5fd', cursor: 'pointer', textDecoration: 'underline', textDecorationColor: '#334155' }}
                >
                  in {module.id}
                </span>
              )}
            </div>
          ) : module && (
            <div
              style={{
                fontSize: 11,
                color: '#64748b',
                marginTop: 4,
                display: 'flex',
                gap: 10,
              }}
            >
              <span>
                Risk{' '}
                <span
                  style={{ color: severityColor(riskSeverity(module.risk_score)) }}
                >
                  {Math.round(module.risk_score * 100)}
                </span>
              </span>
              <span>
                Complexity{' '}
                <span style={{ color: '#cbd5e1' }}>
                  {Math.round(module.complexity_score * 100)}
                </span>
              </span>
              <span>{module.file_count} files</span>
            </div>
          )}
        </div>
        <button
          onClick={onClose}
          aria-label="Close inspector"
          style={{
            background: 'none',
            border: 'none',
            color: '#64748b',
            cursor: 'pointer',
            fontSize: 16,
            lineHeight: 1,
            padding: '2px 4px',
            flexShrink: 0,
          }}
        >
          ✕
        </button>
      </div>

      {/* Scrollable body */}
      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '0 16px 16px',
        }}
      >
        {/* Findings */}
        <SectionHeader>
          Findings ({displayedRisks.length})
        </SectionHeader>
        {displayedRisks.length === 0 ? (
          <div style={{ color: '#334155', fontSize: 11, padding: '8px 0' }}>
            No findings
          </div>
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

        {/* Dependencies */}
        {imports.length > 0 && (
          <>
            <SectionHeader>Imports ({imports.length})</SectionHeader>
            {imports.map((id) => (
              <NavItem key={id} id={id} prefix="→" onNavigate={onNavigate} />
            ))}
          </>
        )}

        {importedBy.length > 0 && (
          <>
            <SectionHeader>Imported by ({importedBy.length})</SectionHeader>
            {importedBy.map((id) => (
              <NavItem key={id} id={id} prefix="←" onNavigate={onNavigate} />
            ))}
          </>
        )}

        {/* Files */}
        {moduleFiles.length > 0 && (
          <>
            <SectionHeader>
              Files — sorted by risk
            </SectionHeader>
            <div
              style={{
                display: 'flex',
                fontSize: 10,
                color: '#334155',
                padding: '4px 0',
                gap: 8,
              }}
            >
              <span style={{ flex: 1 }}>name</span>
              <span style={{ width: 36 }}>LOC</span>
              <span style={{ width: 36, textAlign: 'right' }}>risk</span>
            </div>
            {moduleFiles.map((f) => (
              <FileRow key={f.path} file={f} onNavigate={onNavigate} />
            ))}
          </>
        )}
      </div>
    </div>
  )
}

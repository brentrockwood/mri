import { useCallback, useState } from 'react'
import type { Analysis, File, Risk } from '../types/analysis'
import { complexityColor } from '../lib/risk'
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
  repoName: string
  isWindows: boolean
}

function FindingRow({ risk, repoName, isWindows }: FindingRowProps) {
  const line = risk.evidence_lines?.[0]
  const ghUrl = githubUrl(repoName, risk.file, line)
  const vsUrl = vscodeUrl(
    isWindows ? risk.file : `/${risk.file}`,
    line,
  )

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
        <span style={{ color: '#e2e8f0', fontSize: 12, flex: 1 }}>
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
        <LinkButton href={ghUrl} label="gh" />
        <LinkButton href={vsUrl} label="vs" />
        <CopyButton text={risk.file} />
      </div>
    </div>
  )
}

// ── FileRow ───────────────────────────────────────────────────────────────────

function FileRow({ file }: { file: File }) {
  const dotColor = complexityColor(file.risk_score)
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        padding: '5px 0',
        borderBottom: '1px solid #1e293b',
        fontSize: 11,
        fontFamily: 'monospace',
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
          color: '#cbd5e1',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        }}
      >
        {file.path.split('/').pop()}
      </span>
      <span style={{ color: '#475569', flexShrink: 0 }}>{file.lines}L</span>
      <span
        style={{
          color: severityColor(file.risk_score >= 0.7 ? 'high' : file.risk_score >= 0.4 ? 'medium' : 'low'),
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
}

/**
 * Right-side detail panel shown when a module node is selected.
 * Displays findings, dependency list, and file table for the selected module.
 */
export function Inspector({ selectedId, analysis, onClose }: InspectorProps) {
  const { modules, risks, files, dependencies, repo } = analysis

  const module = modules.find((m) => m.id === selectedId)

  const isWindows = detectWindowsPaths(files.map((f) => f.path))

  const moduleRisks = risks
    .filter((r) => r.module === selectedId)
    .sort((a, b) => {
      const order = { high: 0, medium: 1, low: 2 }
      return (order[a.severity as keyof typeof order] ?? 3) -
        (order[b.severity as keyof typeof order] ?? 3)
    })

  const moduleFiles = files
    .filter((f) => f.module === selectedId)
    .sort((a, b) => b.risk_score - a.risk_score)

  const imports = dependencies.filter((d) => d.from === selectedId).map((d) => d.to)
  const importedBy = dependencies.filter((d) => d.to === selectedId).map((d) => d.from)

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
            {selectedId}
          </div>
          {module && (
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
                  style={{
                    color: severityColor(
                      module.risk_score >= 0.7
                        ? 'high'
                        : module.risk_score >= 0.4
                          ? 'medium'
                          : 'low',
                    ),
                  }}
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
          Findings ({moduleRisks.length})
        </SectionHeader>
        {moduleRisks.length === 0 ? (
          <div style={{ color: '#334155', fontSize: 11, padding: '8px 0' }}>
            No findings
          </div>
        ) : (
          moduleRisks.map((r) => (
            <FindingRow
              key={r.id}
              risk={r}
              repoName={repo.name}
              isWindows={isWindows}
            />
          ))
        )}

        {/* Dependencies */}
        {imports.length > 0 && (
          <>
            <SectionHeader>Imports ({imports.length})</SectionHeader>
            {imports.map((id) => (
              <div
                key={id}
                style={{
                  fontSize: 11,
                  fontFamily: 'monospace',
                  color: '#94a3b8',
                  padding: '4px 0',
                  borderBottom: '1px solid #1e293b',
                }}
              >
                → {id}
              </div>
            ))}
          </>
        )}

        {importedBy.length > 0 && (
          <>
            <SectionHeader>Imported by ({importedBy.length})</SectionHeader>
            {importedBy.map((id) => (
              <div
                key={id}
                style={{
                  fontSize: 11,
                  fontFamily: 'monospace',
                  color: '#94a3b8',
                  padding: '4px 0',
                  borderBottom: '1px solid #1e293b',
                }}
              >
                ← {id}
              </div>
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
              <FileRow key={f.path} file={f} />
            ))}
          </>
        )}
      </div>
    </div>
  )
}

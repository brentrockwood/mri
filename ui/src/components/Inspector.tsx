import { useCallback, useEffect, useMemo, useState } from 'react'
import type { Analysis, File, Risk } from '../types/analysis'
import type { ZoomLevel } from '../layout/types'
import { complexityColor, riskSeverity, severityColorClass } from '../lib/risk'
import { copyToClipboard, detectWindowsPaths, githubUrl, vscodeUrl } from '../lib/deeplinks'
import { cn } from '../lib/cn'

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
        'px-1.5 py-px text-[10px] font-mono border border-border-strong rounded-[3px] cursor-pointer shrink-0',
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
      className="px-1.5 py-px text-[10px] font-mono bg-panel text-blue-400 border border-border-strong rounded-[3px] no-underline shrink-0"
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
    <div className="py-2 border-b border-panel">
      <div className="flex items-start gap-1.5 mb-1">
        <span
          className={`text-[10px] font-bold uppercase shrink-0 pt-px ${severityColorClass(risk.severity)}`}
        >
          {risk.severity}
        </span>
        <span className="text-text-secondary text-xs flex-1 min-w-0 break-words">
          {risk.title}
        </span>
        <span className="text-text-muted text-[10px] shrink-0">
          {confidencePct(risk.confidence)}
        </span>
      </div>
      {risk.description && (
        <div className="text-risk-low text-[11px] mb-1.5 leading-[1.4]">
          {risk.description}
        </div>
      )}
      <div className="flex items-center gap-1 flex-wrap">
        <span className="text-text-dim text-[10px] font-mono flex-1 min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">
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
      className="flex items-center gap-2 py-[5px] border-b border-panel text-[11px] font-mono cursor-pointer"
    >
      <span
        className="w-2 h-2 rounded-full shrink-0"
        style={{ background: dotColor }}
      />
      <span className="flex-1 text-link overflow-hidden text-ellipsis whitespace-nowrap underline decoration-border-strong">
        {file.path.split('/').pop()}
      </span>
      <span className="text-text-dim shrink-0">{file.lines}L</span>
      <span
        className={`shrink-0 w-9 text-right ${severityColorClass(riskSeverity(file.risk_score))}`}
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
      className="text-[11px] font-mono text-link py-1 border-b border-panel cursor-pointer underline decoration-border-strong"
    >
      {prefix} {id}
    </div>
  )
}

// ── SectionHeader ─────────────────────────────────────────────────────────────

function SectionHeader({ children }: { children: React.ReactNode }) {
  return (
    <div className="text-[10px] font-bold text-text-dim uppercase tracking-[0.08em] pt-3 pb-1.5 border-b border-border-strong mb-0.5">
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
    <div className="absolute top-0 right-0 bottom-0 w-[360px] bg-canvas border-l border-border-strong flex flex-col z-[100] overflow-hidden">
      {/* Header */}
      <div className="px-4 py-3 border-b border-border-strong flex items-start gap-2">
        <div className="flex-1 min-w-0">
          <div className="text-[13px] font-bold text-text-primary font-mono break-all">
            {selectedFile ? selectedFile.path.split('/').pop() : selectedId}
          </div>
          {selectedFile && (
            <div className="text-[11px] text-text-dim mt-0.5 font-mono">
              {selectedFile.path}
            </div>
          )}
          {selectedFile ? (
            <div className="text-[11px] text-text-muted mt-1 flex gap-2.5 flex-wrap">
              <span>
                Risk{' '}
                <span className={severityColorClass(riskSeverity(selectedFile.risk_score))}>
                  {Math.round(selectedFile.risk_score * 100)}
                </span>
              </span>
              <span className="text-text-secondary">{selectedFile.lines}L</span>
              {module && (
                <span
                  onClick={() => onNavigate(module.id, 3)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={(e) => e.key === 'Enter' && onNavigate(module.id, 3)}
                  className="text-link cursor-pointer underline decoration-border-strong"
                >
                  in {module.id}
                </span>
              )}
            </div>
          ) : module && (
            <div className="text-[11px] text-text-muted mt-1 flex gap-2.5">
              <span>
                Risk{' '}
                <span className={severityColorClass(riskSeverity(module.risk_score))}>
                  {Math.round(module.risk_score * 100)}
                </span>
              </span>
              <span>
                Complexity{' '}
                <span className="text-text-secondary">
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
          className="bg-transparent border-none text-text-muted cursor-pointer text-base leading-none px-1 py-0.5 shrink-0"
        >
          ✕
        </button>
      </div>

      {/* Scrollable body */}
      <div className="flex-1 overflow-y-auto px-4 pb-4">
        {/* Findings */}
        <SectionHeader>
          Findings ({displayedRisks.length})
        </SectionHeader>
        {displayedRisks.length === 0 ? (
          <div className="text-border-strong text-[11px] py-2">
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
            <div className="flex text-[10px] text-border-strong py-1 gap-2">
              <span className="flex-1">name</span>
              <span className="w-9">LOC</span>
              <span className="w-9 text-right">risk</span>
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

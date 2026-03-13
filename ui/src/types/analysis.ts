/**
 * TypeScript mirror of schema/analysis.go (schema version 1.2).
 * Keep in sync when the CLI schema version changes.
 */

/** Top-level output artifact written to .repo-mri/analysis.json. */
export interface Analysis {
  meta: Meta
  repo: Repo
  modules: Module[]
  dependencies: Dependency[]
  risks: Risk[]
  files: File[]
  subsystems?: Subsystem[]
}

/** Metadata about the analysis run. */
export interface Meta {
  schema_version: string
  cli_version: string
  model_used?: string
  provider?: string
  analysis_duration_ms: number
  skipped_passes?: string[]
  /** Length of the longest module dependency chain. Set by Phase 2. */
  max_chain_depth?: number
  /**
   * Absolute filesystem path to the repository root.
   * Set for local analyses only; absent for remote (GitHub) analyses.
   * Required to construct working VS Code deep links.
   */
  root_path?: string
}

/** Repository-level summary data. */
export interface Repo {
  name: string
  /** Full "owner/repo" slug, populated only when analyzed from a GitHub URL. */
  github_slug?: string
  languages: string[]
  file_count: number
  module_count: number
  analysis_time: string
}

/** Logical grouping of files (typically a package/directory). */
export interface Module {
  id: string
  path: string
  language: string
  file_count: number
  risk_score: number
  complexity_score: number
  /** Number of other modules that import this module. Set by Phase 2. */
  import_count?: number
}

/** Directed import relationship between modules. */
export interface Dependency {
  from: string
  to: string
  type: string
}

/** Single finding from an analysis pass. */
export interface Risk {
  id: string
  severity: string
  type: string
  pass: string
  module: string
  file: string
  title: string
  description: string
  confidence: number
  evidence_lines?: number[]
  /** Classifies what the finding targets: "file", "module", or "repository". */
  target_type?: string
  /** File path, module ID, or repo name depending on target_type. */
  target_id?: string
}

/** Per-file metrics. */
export interface File {
  path: string
  module: string
  language: string
  lines: number
  /** File size in bytes. Set by Phase 2. */
  size?: number
  complexity: number
  risk_score: number
}

/** Groups related modules. */
export interface Subsystem {
  id: string
  modules: string[]
}

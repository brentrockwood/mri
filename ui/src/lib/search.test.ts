import { describe, expect, it } from 'vitest'
import { hitKey, matchingModuleIds, search } from './search'
import type { Analysis } from '../types/analysis'

const analysis: Analysis = {
  meta: { schema_version: '1.2', cli_version: '0.1.0', analysis_duration_ms: 0 },
  repo: { name: 'test-repo', languages: ['Go'], file_count: 4, module_count: 2, analysis_time: '' },
  modules: [
    {
      id: 'internal/analysis',
      path: 'internal/analysis',
      language: 'Go',
      file_count: 2,
      risk_score: 0.4,
      complexity_score: 0.5,
    },
    {
      id: 'cmd/repo-mri',
      path: 'cmd/repo-mri',
      language: 'Go',
      file_count: 1,
      risk_score: 0.1,
      complexity_score: 0.2,
    },
  ],
  dependencies: [],
  risks: [
    {
      id: 'risk-1',
      severity: 'high',
      type: 'complexity',
      pass: 'static',
      module: 'internal/analysis',
      file: 'internal/analysis/analyzer.go',
      title: 'High cyclomatic complexity',
      description: 'Function exceeds threshold.',
      confidence: 0.9,
    },
    {
      id: 'risk-2',
      severity: 'medium',
      type: 'security',
      pass: 'security',
      module: 'cmd/repo-mri',
      file: 'cmd/repo-mri/main.go',
      title: 'Context not checked',
      description: 'Missing ctx.Err() between phases.',
      confidence: 0.8,
    },
  ],
  files: [
    {
      path: 'internal/analysis/analyzer.go',
      module: 'internal/analysis',
      language: 'Go',
      lines: 200,
      complexity: 12,
      risk_score: 0.8,
    },
    {
      path: 'internal/analysis/passes.go',
      module: 'internal/analysis',
      language: 'Go',
      lines: 171,
      complexity: 7,
      risk_score: 0.4,
    },
    {
      path: 'cmd/repo-mri/main.go',
      module: 'cmd/repo-mri',
      language: 'Go',
      lines: 90,
      complexity: 3,
      risk_score: 0.1,
    },
  ],
}

// ── search ────────────────────────────────────────────────────────────────────

describe('search', () => {
  it('returns empty array for blank query', () => {
    expect(search('', analysis)).toHaveLength(0)
    expect(search('   ', analysis)).toHaveLength(0)
  })

  it('returns empty array when nothing matches', () => {
    expect(search('zzznomatch', analysis)).toHaveLength(0)
  })

  it('matches module by ID (case-insensitive)', () => {
    const hits = search('ANALYSIS', analysis)
    const moduleHits = hits.filter((h) => h.kind === 'module')
    expect(moduleHits).toHaveLength(1)
    expect(moduleHits[0].kind === 'module' && moduleHits[0].id).toBe('internal/analysis')
  })

  it('matches file by path (case-insensitive)', () => {
    const hits = search('analyzer', analysis)
    const fileHits = hits.filter((h) => h.kind === 'file')
    expect(fileHits).toHaveLength(1)
    expect(fileHits[0].kind === 'file' && fileHits[0].path).toBe(
      'internal/analysis/analyzer.go',
    )
  })

  it('matches finding by title', () => {
    const hits = search('cyclomatic', analysis)
    const findingHits = hits.filter((h) => h.kind === 'finding')
    expect(findingHits).toHaveLength(1)
    expect(findingHits[0].kind === 'finding' && findingHits[0].label).toBe(
      'High cyclomatic complexity',
    )
  })

  it('matches finding by description', () => {
    const hits = search('ctx.Err', analysis)
    const findingHits = hits.filter((h) => h.kind === 'finding')
    expect(findingHits).toHaveLength(1)
    expect(findingHits[0].kind === 'finding' && findingHits[0].id).toBe('risk-2')
  })

  it('returns correct moduleId and zoomLevel for file hit', () => {
    const hits = search('passes', analysis)
    const fileHit = hits.find((h) => h.kind === 'file')
    expect(fileHit?.moduleId).toBe('internal/analysis')
    expect(fileHit?.zoomLevel).toBe(3)
  })

  it('returns correct moduleId and zoomLevel for finding hit', () => {
    const hits = search('cyclomatic', analysis)
    const findingHit = hits.find((h) => h.kind === 'finding')
    expect(findingHit?.moduleId).toBe('internal/analysis')
    expect(findingHit?.zoomLevel).toBe(2)
  })

  it('returns results across multiple kinds for broad query', () => {
    const hits = search('internal', analysis)
    const kinds = new Set(hits.map((h) => h.kind))
    expect(kinds.has('module')).toBe(true)
    expect(kinds.has('file')).toBe(true)
  })
})

// ── hitKey ────────────────────────────────────────────────────────────────────

describe('hitKey', () => {
  it('produces distinct keys for different kinds', () => {
    const hits = search('analysis', analysis)
    const keys = hits.map(hitKey)
    const uniqueKeys = new Set(keys)
    expect(uniqueKeys.size).toBe(keys.length)
  })
})

// ── matchingModuleIds ─────────────────────────────────────────────────────────

describe('matchingModuleIds', () => {
  it('returns null for blank query', () => {
    expect(matchingModuleIds('', analysis)).toBeNull()
    expect(matchingModuleIds('  ', analysis)).toBeNull()
  })

  it('returns empty set when nothing matches', () => {
    const result = matchingModuleIds('zzznomatch', analysis)
    expect(result).not.toBeNull()
    expect(result?.size).toBe(0)
  })

  it('returns matching module IDs for a module hit', () => {
    const result = matchingModuleIds('analysis', analysis)
    expect(result?.has('internal/analysis')).toBe(true)
  })

  it('returns the owning module ID for a file hit', () => {
    const result = matchingModuleIds('main.go', analysis)
    expect(result?.has('cmd/repo-mri')).toBe(true)
  })

  it('returns the owning module ID for a finding hit', () => {
    const result = matchingModuleIds('cyclomatic', analysis)
    expect(result?.has('internal/analysis')).toBe(true)
  })
})

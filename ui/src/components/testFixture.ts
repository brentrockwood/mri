import type { Analysis } from '../types/analysis'

/** Minimal valid Analysis fixture for component tests. */
export const testAnalysis: Analysis = {
  meta: {
    schema_version: '1.3',
    cli_version: '0.1.0',
    analysis_duration_ms: 0,
    root_path: '/home/user/project',
  },
  repo: {
    name: 'test/repo',
    github_slug: 'owner/repo',
    languages: ['go'],
    file_count: 2,
    module_count: 1,
    analysis_time: '2026-03-14T00:00:00Z',
  },
  modules: [
    {
      id: 'internal/foo',
      path: 'internal/foo',
      language: 'go',
      risk_score: 0.7,
      complexity_score: 0.5,
      file_count: 2,
      import_count: 1,
    },
    {
      id: 'internal/auth',
      path: 'internal/auth',
      language: 'go',
      risk_score: 0.5,
      complexity_score: 0.4,
      file_count: 1,
      import_count: 0,
    },
  ],
  dependencies: [{ from: 'internal/foo', to: 'schema', type: 'import' }],
  risks: [
    {
      id: 'r1',
      severity: 'high',
      type: 'sast',
      pass: 'semgrep',
      module: 'internal/foo',
      file: 'internal/foo/foo.go',
      title: 'SQL Injection',
      description: 'Unsanitized input.',
      confidence: 0.9,
      target_type: 'module',
      target_id: 'internal/foo',
    },
    {
      id: 'r2',
      severity: 'medium',
      type: 'dep-vuln',
      pass: 'npm-audit',
      module: 'internal/foo',
      file: 'internal/foo/foo.go',
      title: 'Outdated dep',
      description: '',
      confidence: 1,
      target_type: 'file',
      target_id: 'internal/foo/foo.go',
    },
  ],
  files: [
    { path: 'internal/foo/foo.go', module: 'internal/foo', language: 'go', lines: 120, complexity: 6, risk_score: 0.7 },
    { path: 'internal/foo/bar.go', module: 'internal/foo', language: 'go', lines: 40, complexity: 2, risk_score: 0.2 },
    { path: 'internal/auth/auth.go', module: 'internal/auth', language: 'go', lines: 80, complexity: 3, risk_score: 0.5 },
  ],
  file_deps: [],
}

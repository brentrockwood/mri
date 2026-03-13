import { describe, expect, it } from 'vitest'
import { detectWindowsPaths, githubUrl, vscodeUrl } from './deeplinks'

describe('githubUrl', () => {
  it('builds URL without line number', () => {
    expect(githubUrl('org/repo', 'internal/analysis/analyzer.go')).toBe(
      'https://github.com/org/repo/blob/main/internal/analysis/analyzer.go',
    )
  })

  it('appends line anchor when line is provided', () => {
    expect(githubUrl('org/repo', 'internal/analysis/analyzer.go', 42)).toBe(
      'https://github.com/org/repo/blob/main/internal/analysis/analyzer.go#L42',
    )
  })

  it('handles single-segment repo names', () => {
    expect(githubUrl('myrepo', 'main.go', 1)).toBe(
      'https://github.com/myrepo/blob/main/main.go#L1',
    )
  })
})

describe('vscodeUrl — Unix paths', () => {
  it('produces double-slash for Unix absolute path without line', () => {
    expect(vscodeUrl('/home/user/project/internal/analysis/analyzer.go')).toBe(
      'vscode://file//home/user/project/internal/analysis/analyzer.go',
    )
  })

  it('appends line number for Unix absolute path', () => {
    expect(vscodeUrl('/home/user/project/internal/analysis/analyzer.go', 42)).toBe(
      'vscode://file//home/user/project/internal/analysis/analyzer.go:42',
    )
  })
})

describe('vscodeUrl — Windows paths', () => {
  it('produces drive-letter URL without line', () => {
    expect(vscodeUrl('C:/Users/user/project/internal/analysis/analyzer.go')).toBe(
      'vscode://file/C:/Users/user/project/internal/analysis/analyzer.go',
    )
  })

  it('appends line number for Windows path', () => {
    expect(vscodeUrl('C:/Users/user/project/internal/analysis/analyzer.go', 42)).toBe(
      'vscode://file/C:/Users/user/project/internal/analysis/analyzer.go:42',
    )
  })
})

describe('detectWindowsPaths', () => {
  it('returns false for Unix absolute paths', () => {
    expect(
      detectWindowsPaths(['/home/user/file.go', '/usr/local/src/file.go']),
    ).toBe(false)
  })

  it('returns false for relative forward-slash paths', () => {
    expect(detectWindowsPaths(['internal/analysis/file.go', 'cmd/main.go'])).toBe(
      false,
    )
  })

  it('returns true when a path has a drive letter', () => {
    expect(detectWindowsPaths(['C:/Users/user/project/file.go'])).toBe(true)
  })

  it('returns true when a path has a backslash', () => {
    expect(detectWindowsPaths(['internal\\analysis\\file.go'])).toBe(true)
  })

  it('returns true when only one path in a mixed set is Windows', () => {
    expect(
      detectWindowsPaths(['internal/analysis/file.go', 'C:/other/file.go']),
    ).toBe(true)
  })

  it('returns false for empty array', () => {
    expect(detectWindowsPaths([])).toBe(false)
  })
})

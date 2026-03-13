import { describe, it, expect, afterEach } from 'vitest'
import { loadAnalysis, MriDataMissingError, MriDataInvalidError } from './useAnalysis'
import fixture from '../../fixtures/analysis.json'

type MriWindow = Window & { __MRI_DATA__?: unknown }

const mriWin = window as MriWindow

describe('loadAnalysis', () => {
  afterEach(() => {
    delete mriWin.__MRI_DATA__
  })

  it('parses fixture data from window.__MRI_DATA__', () => {
    mriWin.__MRI_DATA__ = fixture
    const analysis = loadAnalysis()
    expect(analysis.repo.name).toBe('test-repo')
    expect(analysis.repo.module_count).toBe(2)
    expect(analysis.modules).toHaveLength(2)
    expect(analysis.modules.map((m) => m.id)).toContain('internal/analysis')
    expect(analysis.meta.schema_version).toBe('1.3')
    expect(analysis.meta.root_path).toBe('/home/user/project')
  })

  it('throws MriDataMissingError when window.__MRI_DATA__ is absent', () => {
    delete mriWin.__MRI_DATA__
    expect(() => loadAnalysis()).toThrow(MriDataMissingError)
  })

  it('throws MriDataInvalidError when window.__MRI_DATA__ is not an object', () => {
    mriWin.__MRI_DATA__ = 'not-an-object'
    expect(() => loadAnalysis()).toThrow(MriDataInvalidError)
  })

  it('throws MriDataInvalidError when window.__MRI_DATA__ is null', () => {
    mriWin.__MRI_DATA__ = null
    expect(() => loadAnalysis()).toThrow(MriDataInvalidError)
  })
})

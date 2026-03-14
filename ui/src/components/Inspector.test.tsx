import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { theme } from '../lib/theme'
import { Inspector } from './Inspector'
import { testAnalysis } from './testFixture'

function wrap(ui: React.ReactElement) {
  return render(<ThemeProvider theme={theme}>{ui}</ThemeProvider>)
}

describe('Inspector', () => {
  it('shows placeholder when no selection', () => {
    wrap(<Inspector open selectedId={null} analysis={testAnalysis} onClose={() => {}} onNavigate={() => {}} />)
    expect(screen.getByText('Select a node to inspect')).toBeDefined()
  })

  it('shows module id in header when a module is selected', () => {
    wrap(<Inspector open selectedId="internal/foo" analysis={testAnalysis} onClose={() => {}} onNavigate={() => {}} />)
    expect(screen.getByText('internal/foo')).toBeDefined()
  })

  it('renders findings for selected module', () => {
    wrap(<Inspector open selectedId="internal/foo" analysis={testAnalysis} onClose={() => {}} onNavigate={() => {}} />)
    expect(screen.getByText('SQL Injection')).toBeDefined()
  })

  it('renders files table rows for selected module', () => {
    wrap(<Inspector open selectedId="internal/foo" analysis={testAnalysis} onClose={() => {}} onNavigate={() => {}} />)
    expect(screen.getByText('foo.go')).toBeDefined()
    expect(screen.getByText('bar.go')).toBeDefined()
  })

  it('renders imports list', () => {
    wrap(<Inspector open selectedId="internal/foo" analysis={testAnalysis} onClose={() => {}} onNavigate={() => {}} />)
    expect(screen.getByText('→ schema')).toBeDefined()
  })

  it('calls onClose when close button is clicked', () => {
    const handler = vi.fn()
    wrap(<Inspector open selectedId="internal/foo" analysis={testAnalysis} onClose={handler} onNavigate={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: 'Close inspector' }))
    expect(handler).toHaveBeenCalled()
  })

  it('calls onClose on Escape key', () => {
    const handler = vi.fn()
    wrap(<Inspector open selectedId="internal/foo" analysis={testAnalysis} onClose={handler} onNavigate={() => {}} />)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(handler).toHaveBeenCalled()
  })

  it('calls onNavigate when a file row is clicked', () => {
    const handler = vi.fn()
    wrap(<Inspector open selectedId="internal/foo" analysis={testAnalysis} onClose={() => {}} onNavigate={handler} />)
    fireEvent.click(screen.getByText('foo.go'))
    expect(handler).toHaveBeenCalledWith('internal/foo/foo.go', 3)
  })
})

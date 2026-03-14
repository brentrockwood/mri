import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { theme } from '../lib/theme'
import { StatusBar } from './StatusBar'
import { testAnalysis } from './testFixture'

function wrap(ui: React.ReactElement) {
  return render(<ThemeProvider theme={theme}>{ui}</ThemeProvider>)
}

describe('StatusBar', () => {
  it('renders all three zoom level tabs', () => {
    wrap(<StatusBar level={1} selectedId={null} analysis={testAnalysis} onLevelChange={() => {}} />)
    expect(screen.getByRole('tab', { name: 'Architecture' })).toBeDefined()
    expect(screen.getByRole('tab', { name: 'Modules' })).toBeDefined()
    expect(screen.getByRole('tab', { name: 'Files' })).toBeDefined()
  })

  it('calls onLevelChange when a tab is clicked', () => {
    const handler = vi.fn()
    wrap(<StatusBar level={1} selectedId={null} analysis={testAnalysis} onLevelChange={handler} />)
    fireEvent.click(screen.getByRole('tab', { name: 'Modules' }))
    expect(handler).toHaveBeenCalledWith(2)
  })

  it('shows high risk count from fixture data', () => {
    wrap(<StatusBar level={2} selectedId={null} analysis={testAnalysis} onLevelChange={() => {}} />)
    expect(screen.getByText('1 high')).toBeDefined()
  })

  it('shows module and file counts from fixture', () => {
    wrap(<StatusBar level={1} selectedId={null} analysis={testAnalysis} onLevelChange={() => {}} />)
    expect(screen.getByText('1 modules')).toBeDefined()
    expect(screen.getByText('2 files')).toBeDefined()
  })
})

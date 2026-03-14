import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { theme } from '../lib/theme'
import { SearchBar } from './SearchBar'
import { testAnalysis } from './testFixture'

function wrap(ui: React.ReactElement) {
  return render(<ThemeProvider theme={theme}>{ui}</ThemeProvider>)
}

describe('SearchBar', () => {
  it('renders a search input with combobox role', () => {
    wrap(<SearchBar query="" onQueryChange={() => {}} analysis={testAnalysis} onSelect={() => {}} />)
    expect(screen.getByRole('combobox')).toBeDefined()
  })

  it('input has ARIA combobox attributes', () => {
    wrap(<SearchBar query="" onQueryChange={() => {}} analysis={testAnalysis} onSelect={() => {}} />)
    const input = screen.getByRole('combobox')
    expect(input.getAttribute('aria-expanded')).toBe('false')
    expect(input.getAttribute('aria-autocomplete')).toBe('list')
    expect(input.getAttribute('aria-controls')).toBe('mri-search-listbox')
  })

  it('calls onQueryChange when typing', () => {
    const handler = vi.fn()
    wrap(<SearchBar query="" onQueryChange={handler} analysis={testAnalysis} onSelect={() => {}} />)
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'auth' } })
    expect(handler).toHaveBeenCalledWith('auth')
  })

  it('calls onQueryChange with empty string on Escape', () => {
    const handler = vi.fn()
    wrap(<SearchBar query="foo" onQueryChange={handler} analysis={testAnalysis} onSelect={() => {}} />)
    fireEvent.keyDown(screen.getByRole('combobox'), { key: 'Escape' })
    expect(handler).toHaveBeenCalledWith('')
  })
})

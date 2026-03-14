import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { theme } from '../lib/theme'
import { Tooltip } from './Tooltip'
import { testAnalysis } from './testFixture'

function wrap(ui: React.ReactElement) {
  return render(<ThemeProvider theme={theme}>{ui}</ThemeProvider>)
}

describe('Tooltip', () => {
  it('renders module name and file count', () => {
    wrap(<Tooltip moduleId="internal/foo" analysis={testAnalysis} mouseX={100} mouseY={200} />)
    expect(screen.getByText('internal/foo')).toBeDefined()
    expect(screen.getByText(/2 files/)).toBeDefined()
  })

  it('renders risk count badge when module has findings', () => {
    wrap(<Tooltip moduleId="internal/foo" analysis={testAnalysis} mouseX={100} mouseY={200} />)
    expect(screen.getByText('1H')).toBeDefined()
  })

  it('returns null for unknown moduleId', () => {
    const { container } = wrap(<Tooltip moduleId="nonexistent" analysis={testAnalysis} mouseX={0} mouseY={0} />)
    expect(container.firstChild).toBeNull()
  })
})

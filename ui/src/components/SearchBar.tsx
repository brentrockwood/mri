import { useCallback, useMemo, useRef, useState } from 'react'
import TextField from '@mui/material/TextField'
import Paper from '@mui/material/Paper'
import List from '@mui/material/List'
import ListItemButton from '@mui/material/ListItemButton'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import type { Analysis } from '../types/analysis'
import { hitKey, search } from '../lib/search'
import type { SearchHit } from '../lib/search'

const KIND_LABEL: Record<SearchHit['kind'], string> = {
  module: 'Module',
  file: 'File',
  finding: 'Finding',
}

const KIND_COLOR: Record<SearchHit['kind'], string> = {
  module: '#60a5fa',
  file: '#94a3b8',
  finding: '#f87171',
}

export interface SearchBarProps {
  query: string
  onQueryChange: (q: string) => void
  analysis: Analysis
  onSelect: (hit: SearchHit) => void
}

/**
 * Controlled search input with a dropdown of matching modules, files, and findings.
 * Uses MUI TextField for the input and MUI Paper/List for the dropdown.
 * Keyboard: ArrowUp/Down to navigate results, Enter to select, Escape to clear.
 */
export function SearchBar({ query, onQueryChange, analysis, onSelect }: SearchBarProps) {
  const [activeIndex, setActiveIndex] = useState(0)
  const [open, setOpen] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const results = useMemo(() => search(query, analysis), [query, analysis])

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      onQueryChange(e.target.value)
      setActiveIndex(0)
      setOpen(true)
    },
    [onQueryChange],
  )

  const handleSelect = useCallback(
    (hit: SearchHit) => {
      setActiveIndex(0)
      setOpen(false)
      onSelect(hit)
    },
    [onSelect],
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setActiveIndex((i) => Math.min(i + 1, results.length - 1))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setActiveIndex((i) => Math.max(i - 1, 0))
      } else if (e.key === 'Enter') {
        const hit = results[activeIndex]
        if (hit) handleSelect(hit)
      } else if (e.key === 'Escape') {
        onQueryChange('')
        setOpen(false)
        inputRef.current?.blur()
      }
    },
    [results, activeIndex, handleSelect, onQueryChange],
  )

  const showDropdown = open && query.trim().length > 0 && results.length > 0
  const listboxId = 'mri-search-listbox'
  const activeOptionId = showDropdown ? `mri-search-option-${activeIndex}` : undefined

  return (
    <Box
      sx={{ position: 'absolute', top: 32, left: '50%', transform: 'translateX(-50%)', zIndex: 50, width: 340 }}
      onMouseDown={(e) => e.stopPropagation()} // prevent SVG pan starting
    >
      <TextField
        inputRef={inputRef}
        fullWidth
        size="small"
        value={query}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        onFocus={() => setOpen(true)}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
        placeholder="Search modules, files, findings…"
        inputProps={{
          role: 'combobox',
          'aria-expanded': showDropdown,
          'aria-controls': listboxId,
          'aria-activedescendant': activeOptionId,
          'aria-autocomplete': 'list',
          'aria-label': 'Search modules, files, findings',
        }}
        sx={{
          '& .MuiOutlinedInput-root': {
            borderBottomLeftRadius: showDropdown ? 0 : undefined,
            borderBottomRightRadius: showDropdown ? 0 : undefined,
            boxShadow: '0 4px 16px rgba(0,0,0,0.5)',
          },
        }}
      />

      {showDropdown && (
        <Paper
          elevation={8}
          sx={{
            borderTopLeftRadius: 0,
            borderTopRightRadius: 0,
            overflow: 'hidden',
            bgcolor: 'background.paper',
            border: '1px solid',
            borderColor: 'divider',
            borderTop: 'none',
          }}
        >
          <List id={listboxId} role="listbox" disablePadding aria-label="Search results">
            {results.map((hit, i) => (
              <ListItemButton
                key={hitKey(hit)}
                id={`mri-search-option-${i}`}
                role="option"
                aria-selected={i === activeIndex}
                selected={i === activeIndex}
                onMouseDown={() => handleSelect(hit)}
                dense
                sx={{
                  display: 'flex',
                  alignItems: 'baseline',
                  gap: 1,
                  py: 0.875,
                  px: 1.5,
                  borderTop: i > 0 ? '1px solid' : 'none',
                  borderColor: 'background.default',
                  fontFamily: 'monospace',
                  '&.Mui-selected': { bgcolor: 'background.default' },
                }}
              >
                <Typography
                  component="span"
                  sx={{
                    fontSize: '0.5625rem',
                    fontWeight: 'bold',
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                    flexShrink: 0,
                    width: 48,
                    color: KIND_COLOR[hit.kind],
                    fontFamily: 'monospace',
                  }}
                >
                  {KIND_LABEL[hit.kind]}
                </Typography>
                <Typography
                  component="span"
                  sx={{
                    flex: 1,
                    color: 'text.secondary',
                    fontSize: '0.75rem',
                    fontFamily: 'monospace',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {hit.label}
                </Typography>
                {'detail' in hit && (
                  <Typography
                    component="span"
                    sx={{
                      color: 'text.disabled',
                      fontSize: '0.625rem',
                      fontFamily: 'monospace',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                      maxWidth: 120,
                      flexShrink: 0,
                    }}
                  >
                    {hit.detail}
                  </Typography>
                )}
              </ListItemButton>
            ))}
          </List>
        </Paper>
      )}
    </Box>
  )
}

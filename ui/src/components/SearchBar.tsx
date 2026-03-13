import { useCallback, useMemo, useRef, useState } from 'react'
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
 * `query` and `onQueryChange` are owned by the parent so the input can be
 * cleared externally (e.g. when the user clicks a node or the background).
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

  return (
    <div
      style={{
        position: 'absolute',
        top: 12,
        left: '50%',
        transform: 'translateX(-50%)',
        zIndex: 50,
        width: 340,
      }}
      onMouseDown={(e) => e.stopPropagation()} // prevent SVG pan starting
    >
      {/* Input */}
      <input
        ref={inputRef}
        type="text"
        value={query}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        onFocus={() => setOpen(true)}
        onBlur={() => setTimeout(() => setOpen(false), 150)}
        placeholder="Search modules, files, findings…"
        style={{
          width: '100%',
          boxSizing: 'border-box',
          padding: '7px 12px',
          background: '#1e293b',
          border: '1px solid #334155',
          borderRadius: showDropdown ? '6px 6px 0 0' : '6px',
          color: '#e2e8f0',
          fontSize: 12,
          fontFamily: 'monospace',
          outline: 'none',
          boxShadow: '0 4px 16px rgba(0,0,0,0.5)',
        }}
      />

      {/* Dropdown */}
      {showDropdown && (
        <div
          style={{
            background: '#1e293b',
            border: '1px solid #334155',
            borderTop: 'none',
            borderRadius: '0 0 6px 6px',
            overflow: 'hidden',
            boxShadow: '0 8px 24px rgba(0,0,0,0.6)',
          }}
        >
          {results.map((hit, i) => (
            <div
              key={hitKey(hit)}
              onMouseDown={() => handleSelect(hit)}
              style={{
                display: 'flex',
                alignItems: 'baseline',
                gap: 8,
                padding: '7px 12px',
                cursor: 'pointer',
                background: i === activeIndex ? '#0f172a' : 'transparent',
                borderTop: i > 0 ? '1px solid #0f172a' : 'none',
              }}
            >
              <span
                style={{
                  fontSize: 9,
                  fontWeight: 'bold',
                  color: KIND_COLOR[hit.kind],
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  flexShrink: 0,
                  width: 48,
                }}
              >
                {KIND_LABEL[hit.kind]}
              </span>
              <span
                style={{
                  flex: 1,
                  color: '#e2e8f0',
                  fontSize: 12,
                  fontFamily: 'monospace',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {hit.label}
              </span>
              {'detail' in hit && (
                <span
                  style={{
                    color: '#475569',
                    fontSize: 10,
                    fontFamily: 'monospace',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                    maxWidth: 120,
                    flexShrink: 0,
                  }}
                >
                  {hit.detail}
                </span>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

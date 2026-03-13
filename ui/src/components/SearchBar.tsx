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
      className="absolute top-[2em] left-1/2 -translate-x-1/2 z-50 w-[340px]"
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
        className={[
          'w-full py-[7px] px-3 bg-panel border border-border-strong',
          showDropdown ? 'rounded-t-md' : 'rounded-md',
          'text-text-secondary text-xs font-mono outline-none',
          'shadow-[0_4px_16px_rgba(0,0,0,0.5)]',
        ].join(' ')}
      />

      {/* Dropdown */}
      {showDropdown && (
        <div className="bg-panel border border-border-strong border-t-0 rounded-b-md overflow-hidden shadow-[0_8px_24px_rgba(0,0,0,0.6)]">
          {results.map((hit, i) => (
            <div
              key={hitKey(hit)}
              onMouseDown={() => handleSelect(hit)}
              className={[
                'flex items-baseline gap-2 py-[7px] px-3 cursor-pointer',
                i === activeIndex ? 'bg-canvas' : 'bg-transparent',
                i > 0 ? 'border-t border-canvas' : '',
              ].join(' ')}
            >
              <span
                className="text-[9px] font-bold uppercase tracking-[0.06em] shrink-0 w-12"
                style={{ color: KIND_COLOR[hit.kind] }}
              >
                {KIND_LABEL[hit.kind]}
              </span>
              <span className="flex-1 text-text-secondary text-xs font-mono overflow-hidden text-ellipsis whitespace-nowrap">
                {hit.label}
              </span>
              {'detail' in hit && (
                <span className="text-text-dim text-[10px] font-mono overflow-hidden text-ellipsis whitespace-nowrap max-w-[120px] shrink-0">
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

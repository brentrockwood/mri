import { createTheme } from '@mui/material/styles'

/**
 * Dark MUI theme that mirrors the Tailwind design tokens defined in
 * src/index.css. All colour values must stay in sync with that file.
 */
export const theme = createTheme({
  palette: {
    mode: 'dark',
    background: {
      default: '#0f172a', // --color-canvas
      paper: '#1e293b',   // --color-panel
    },
    divider: '#334155',   // --color-border-subtle
    primary: { main: '#93c5fd' },  // --color-link
    error:   { main: '#f87171' },  // --color-risk-high
    warning: { main: '#fbbf24' },  // --color-risk-med
    text: {
      primary:   '#f8fafc', // --color-text-primary
      secondary: '#cbd5e1', // --color-text-secondary
      disabled:  '#64748b', // --color-text-muted
    },
  },

  typography: {
    fontFamily: 'monospace',
    fontSize: 12,
  },

  shape: { borderRadius: 4 },

  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: {
          margin: 0,
          padding: 0,
          background: '#0f172a',
          color: '#f8fafc',
          fontFamily: 'monospace',
        },
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: {
          backgroundColor: '#1e293b',
          borderLeft: '1px solid #334155',
          boxShadow: '-4px 0 24px rgba(0,0,0,0.6)',
        },
      },
    },
    MuiTabs: {
      styleOverrides: {
        root: { minHeight: 'unset' },
        indicator: { display: 'none' },
      },
    },
    MuiTab: {
      styleOverrides: {
        root: {
          fontFamily: 'monospace',
          fontSize: '1.25rem',
          minHeight: 'unset',
          padding: '8px 24px',
          textTransform: 'none',
          color: '#64748b',
          border: '1px solid #334155',
          borderRadius: '4px',
          marginRight: '4px',
          boxShadow: '0 2px 6px rgba(0,0,0,0.5)',
          '&.Mui-selected': {
            color: '#f8fafc',
            backgroundColor: '#0f172a',
            boxShadow: '2px 0 6px rgba(0,0,0,0.4), -2px 0 6px rgba(0,0,0,0.4)',
            borderTop: 'none',
          },
          '&:hover:not(.Mui-selected)': {
            boxShadow: '0 0 8px rgba(147,197,253,0.3)',
          },
        },
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          fontFamily: 'monospace',
          fontSize: '0.75rem',
          backgroundColor: '#1e293b',
          color: '#cbd5e1',
          '& .MuiOutlinedInput-notchedOutline': {
            borderColor: '#334155',
          },
          '&:hover .MuiOutlinedInput-notchedOutline': {
            borderColor: '#475569',
          },
          '&.Mui-focused .MuiOutlinedInput-notchedOutline': {
            borderColor: '#93c5fd',
          },
        },
        input: {
          padding: '7px 12px',
          '&::placeholder': { color: '#64748b', opacity: 1 },
        },
      },
    },
    MuiTable: {
      styleOverrides: {
        root: { fontFamily: 'monospace' },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        root: {
          fontFamily: 'monospace',
          fontSize: '1rem',
          borderBottom: '1px solid #1e293b',
          padding: '10px 6px',
          color: '#cbd5e1',
        },
        head: {
          fontSize: '0.75rem',
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          color: '#475569',
          padding: '4px 6px',
        },
      },
    },
    MuiTableRow: {
      styleOverrides: {
        root: {
          '&:hover': {
            boxShadow: '0 0 8px rgba(147,197,253,0.3)',
            // cursor is set per-row by the component; not every row is interactive
          },
        },
      },
    },
    MuiIconButton: {
      styleOverrides: {
        root: {
          color: '#64748b',
          '&:hover': { color: '#f8fafc' },
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none', // MUI adds a gradient in dark mode by default
        },
      },
    },
  },
})

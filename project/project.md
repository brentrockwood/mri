# Codebase MRI — UI Build Specification

## What This Is

A self-contained interactive visualization of the `analysis.json` artifact produced by the `repo-mri` CLI. The UI is a single HTML file with all analysis data inlined at generation time — open it directly in a browser with no server required.

The CLI remains the analysis engine. The UI is the navigation interface.

---

## Target Platform

**Minimum:** Raspberry Pi 5 (8 GB RAM, SSD) running current Chrome on ARM64 Linux.

**Also tested:** Windows (current Chrome, x86-64).

Chrome is the only supported browser. No polyfills for other browsers are required. SVG rendering is validated as sufficient for this hardware profile; the Canvas upgrade path exists for larger repositories, not for performance on the minimum target.

---

## Delivery Model

The CLI writes `.repo-mri/report.html` as the sole UI output — a single, fully self-contained file with all JS, CSS, and analysis data inlined. No server is required. The user opens `report.html` directly in a browser from disk. `analysis.json` continues to be written separately as the canonical machine-readable artifact but is not required by the HTML file at runtime.

The UI project lives in a `ui/` subdirectory of the `mri` repository and is built separately from the Go CLI. The CLI's `report` package is responsible for copying the built `report.html` into `.repo-mri/` as part of Phase 6 report generation.

---

## Stack Decisions

| Concern | Decision |
|---|---|
| Framework | React 18 + TypeScript |
| Bundler | Vite + `vite-plugin-singlefile` |
| Graph layout | D3 (layout math only — no DOM rendering via D3) |
| Graph rendering | SVG for MVP; Canvas upgrade path documented below |
| Styling | Tailwind CSS (utility classes only, no custom CSS framework) |
| Testing | Vitest + React Testing Library |
| Linting | ESLint + `@typescript-eslint` |
| Formatting | Prettier |

**Canvas upgrade trigger:** migrate SVG rendering to Canvas if node count exceeds 500 or if measurable jank appears during pan/zoom on target hardware. Document the trigger in context before beginning the migration.

**Vite singlefile:** `vite build` with `vite-plugin-singlefile` produces a single `report.html` with all JS and CSS inlined. No external CDN dependencies. The file must be fully functional with no network access.

---

## Project Structure

```
ui/
  src/
    components/
      MapCanvas.tsx       # SVG graph renderer
      Inspector.tsx       # Right-click / click detail panel
      SearchBar.tsx       # Module / file / finding search
      StatusBar.tsx       # Zoom level, summary counts
      Tooltip.tsx         # Hover overlay
    layout/
      layered.ts          # Path-hierarchy layout algorithm
      types.ts            # Layout graph types
    hooks/
      useAnalysis.ts      # Load and parse analysis.json
      useSelection.ts     # Selected node state
      useZoom.ts          # Zoom level and pan state
    lib/
      deeplinks.ts        # VS Code and GitHub URL builders
      risk.ts             # Risk scoring helpers
    types/
      analysis.ts         # TypeScript mirror of schema/analysis.go
    App.tsx
    main.tsx
  index.html
  vite.config.ts
  tsconfig.json
  package.json
  .eslintrc.json
  .prettierrc
```

---

## Data Model

The UI consumes `analysis.json` directly. The TypeScript types in `src/types/analysis.ts` mirror `schema/analysis.go` and must be kept in sync when the CLI schema version changes.

Key fields consumed by the UI:

**`modules[]`** — graph nodes. `id`, `path`, `risk_score`, `complexity_score`, `file_count`, `import_count`.

**`dependencies[]`** — directed edges. `from` (module id), `to` (module id), `type`.

**`risks[]`** — findings. `severity`, `type`, `target_type`, `target_id`, `module`, `file`, `title`, `description`, `confidence`.

**`files[]`** — per-file metrics. `path`, `module`, `lines`, `complexity`, `risk_score`.

**`meta`** — `schema_version`, `cli_version`, `model_used`, `provider`, `analysis_duration_ms`.

**`repo`** — `name`, `languages`, `file_count`, `module_count`.

---

## Layout Algorithm

The MVP uses a **path-hierarchy layout**. Module IDs are slash-separated paths (e.g. `internal/analysis`). The layout algorithm derives vertical position from path depth and groups siblings horizontally. Dependency edges are drawn on top of the positioned nodes.

This is simpler than a full Sugiyama layered layout and sufficient for Go repos where the directory structure maps closely to the dependency hierarchy.

**Upgrade path:** a full layered DAG layout (topological sort → layer assignment → barycentric crossing reduction → force relaxation within layers) is the correct long-term algorithm. Implement as a drop-in replacement for `layout/layered.ts` in a later phase when the path-hierarchy layout proves insufficient for non-Go repos or deeply tangled graphs.

---

## Visual Encoding

| Signal | Encoding |
|---|---|
| Node size | Proportional to `file_count` (min/max clamped) |
| Node color | `complexity_score` heatmap: green (low) → yellow (moderate) → red (high) |
| Node glow | Present when module has one or more `high` severity findings |
| Edge weight | Stroke width proportional to number of dependencies between module pair |
| Edge direction | Arrowhead on target end |

---

## Zoom Levels

Zoom changes the information model, not just the visual scale.

| Level | Label | Nodes shown |
|---|---|---|
| 1 | Architecture | Top-level path segments only (e.g. `internal`, `cmd`, `schema`) |
| 2 | Modules | All modules (e.g. `internal/analysis`, `internal/ingestion`) |
| 3 | Files | Individual files within the selected module |

Transitions between levels are triggered by scroll wheel depth or explicit zoom controls in the status bar. At Level 3, only the files of the currently selected module are shown to avoid clutter.

---

## Interaction Model

**Hover:** tooltip showing module name, file count, LOC, complexity band, and finding counts by severity.

**Click:** opens the Inspector panel with full module details — all findings for the module, dependency list, file table sorted by risk score.

**Deep links in Inspector:**
- GitHub: `https://github.com/{org}/{repo}/blob/main/{path}` (derived from `repo.name` + file path)
- VS Code: `vscode://file/{absolute_path}:{line}` (line number from `evidence_lines[0]` where available)
- File path copy button for all other editors

**Search:** filters by module name, file name, or finding title. Matching nodes are highlighted; non-matching nodes are dimmed. Selecting a result centers the map on that node and opens the Inspector.

---

## Large Repository Strategy

When `module_count` exceeds 50, default to Level 1 (Architecture) zoom on load. Surface the three areas of interest in the Inspector on load (no node selected):
- Highest `risk_score` module
- Highest `complexity_score` module
- Highest `import_count` module (most coupled)

This gives the user a starting point without rendering every node immediately.

---

## CLI Integration

The CLI's `internal/report` package gains a second output: `report.html`. The build process embeds the compiled `report.html` template into the Go binary using `//go:embed`. At report generation time, the embedded HTML is written to `.repo-mri/report.html` with `analysis.json` path injected.

**Embedding strategy:** the Vite build produces a single `report.html` with all JS and CSS inlined. At report generation time, the CLI templates `analysis.json` directly into the HTML as an inline `<script>` block — `window.__MRI_DATA__ = {...}` — so the file is fully self-contained with no runtime network requests. Chrome blocks `fetch()` on `file://` URLs; inline data is the only approach that works when the file is opened directly from disk.

The `useAnalysis` hook reads from `window.__MRI_DATA__` rather than calling `fetch`. The Vite dev server injects a fixture `analysis.json` via a Vite plugin or a simple `public/__mri_data__.js` shim so development works without the Go CLI.

This requires a `make ui-build` step in the CLI's Makefile that runs `vite build` in `ui/` and copies the output to `internal/report/static/report.html` before the Go binary is built. CI must run `make ui-build` before `make build`.

---

## Build Order

Complete each phase fully before moving to the next.

### Phase UI-1 — Project Skeleton

- Scaffold `ui/` with Vite + React + TypeScript + vite-plugin-singlefile
- Configure ESLint, Prettier, Vitest
- Add `make ui-build` and `make ui-dev` targets to root Makefile
- Implement `src/types/analysis.ts` mirroring current schema (v1.2)
- Implement `useAnalysis` hook: reads from `window.__MRI_DATA__`, parses, exposes typed result; throws a typed error if the global is absent
- Add a Vite dev plugin (or `public/mri-data-dev.js` shim) that populates `window.__MRI_DATA__` from a fixture `analysis.json` during `vite dev`
- Stub `App.tsx`: renders repo name and module count from loaded data
- Verify `vite build` produces a single self-contained `report.html`
- Write one Vitest test: `useAnalysis` parses a fixture object from `window.__MRI_DATA__` correctly and throws when the global is absent

### Phase UI-2 — Graph Layout and Rendering

- Implement `layout/layered.ts`: path-hierarchy layout producing `{id, x, y, width, height}` per node
- Implement `MapCanvas.tsx`: SVG renderer consuming layout output; nodes as rectangles with visual encoding (size, color, glow); edges as lines with arrowheads
- Implement `useZoom`: pan and zoom via SVG `viewBox` manipulation; scroll wheel zoom
- Implement zoom level switching (Architecture / Modules / Files) via status bar controls
- Tests: layout algorithm unit tests covering path grouping, depth assignment, and sibling ordering

### Phase UI-3 — Interaction

- Implement `Tooltip.tsx`: hover overlay with module metadata
- Implement `Inspector.tsx`: click detail panel with findings list, dependency list, file table
- Implement `useSelection`: selected node state, deselect on background click
- Implement `deeplinks.ts`: GitHub URL builder, VS Code URL builder, clipboard copy
  - VS Code deep links use Unix paths (`vscode://file//absolute/path:line`) on Linux and Windows paths (`vscode://file/C:/absolute/path:line`) on Windows; the builder must detect path format from the file paths present in `analysis.json`
- Wire deep links into Inspector findings rows
- Tests: deeplink URL generation for GitHub, VS Code (Unix), and VS Code (Windows) formats

### Phase UI-4 — Search

- Implement `SearchBar.tsx`: controlled input filtering modules, files, and findings by name/title
- Highlight matching nodes; dim non-matching nodes
- Center map on selected search result; open Inspector for that node
- Implement breadcrumb showing current zoom level and selected module path
- Tests: search filtering logic unit tests

### Phase UI-5 — CLI Integration

- Add `//go:embed static/report.html` to `internal/report/report.go`
- Update `report.Generate()` to template `analysis.json` content inline into the embedded HTML as `window.__MRI_DATA__ = <json>;` and write the result to `.repo-mri/report.html`
- `analysis.json` continues to be written separately as the canonical machine-readable artifact
- Update root `Makefile`: `ui-build` target runs `vite build` in `ui/` and copies output to `internal/report/static/report.html`; `build` target depends on `ui-build`
- Update `README.md`: document `report.html` output, how to open it, and the `make ui-dev` workflow
- Integration test: verify `report.html` is written, non-empty, and contains the repo name string after `repo-mri analyze`

---

## Error Handling

- `window.__MRI_DATA__` absent or not a valid object: render a full-page error state explaining that `report.html` must be generated by `repo-mri analyze`, not opened standalone from the `ui/` build directory
- `schema_version` mismatch: render a warning banner noting the CLI version that produced the file and suggesting a re-run with a current binary
- Empty `modules[]`: render an empty state with the repo name and a note that no modules were detected
- Missing `risks[]`: render the graph without findings overlay; no error

---

## Coding Standards

- All React components are functional components with named exports
- No class components
- No `any` types — use `unknown` and narrow explicitly
- All D3 layout code is pure functions in `layout/`; no D3 selection or DOM manipulation outside `MapCanvas.tsx`
- `useAnalysis`, `useSelection`, `useZoom` are the only hooks with side effects; all other state is derived
- ESLint must report zero errors and zero warnings before each commit
- Prettier must be clean before each commit

---

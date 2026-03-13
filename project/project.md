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

### Phase UI-3a — App Deep Links

Encode navigational state in the URL hash so the browser back/forward buttons work within the app and users can share a link that opens the exact view they are looking at.

- Implement `useAppNav` hook: manages `zoomLevel` and `selectedId` as URL-synced state
  - On mount: parse `window.location.hash` and initialise state from it
  - On every navigation action (zoom change, selection change): call `history.pushState` to update the hash without reloading
  - Listen for `popstate` events and sync state back to React on browser back/forward
  - Hash format: `#z=<level>&s=<encoded-id>` using `URLSearchParams`; absent `s` means no selection
  - Module IDs containing `/` must be percent-encoded in the URL
- Replace separate `useState` calls for `zoomLevel` and `selectedId` in `App.tsx` with `useAppNav`; exported interface is identical (`{ zoomLevel, setZoomLevel, selectedId, select }`)
- Use hash-based routing (not query-string `pushState`) — hash is safe on `file://` URLs and standard for server-free SPAs; query-string approach is deferred until a server-based delivery model is adopted
- Tests: URL serialisation, URL parsing (with and without selection), round-trip, `popstate` sync, and handling of malformed or missing hash

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

### Phase UI-6-css — Design System & Tailwind Migration

Establish a consistent design foundation before adding more UI surface area. All style changes in subsequent phases use Tailwind utility classes; inline `style` props are removed.

- Install Tailwind CSS v4 with the official Vite plugin (`@tailwindcss/vite`). No PostCSS config, no `tailwind.config.js` — v4 is configured via CSS `@import` and CSS custom properties.
- Define design tokens in `src/index.css` as Tailwind theme extensions: colour palette (`bg-canvas`, `bg-panel`, `bg-surface`, `border-subtle`, `border-strong`, severity colours `text-risk-high/med/low`, `glow-risk-high`), font sizes (`text-detail` = 1em, `text-label` = 1.25em), spacing, and shadow presets (`shadow-panel`, `shadow-tab-active`, `shadow-tab-inactive`).
- Migrate all inline `style` props across every component to Tailwind utility classes. No new visual changes — the output must look identical to the pre-migration state.
- Verify `vite build` + `vite-plugin-singlefile` produces a single fully-inlined `report.html` with no external CSS references.
- ESLint, Vitest, and Go tests all pass; visual output is unchanged.

### Phase UI-6 — UI Refinements

Builds on the Tailwind foundation from UI-6-css.

**Inspector**
- Add `gh` and `vs` link buttons to `FileRow` in the inspector (same `LinkButton` logic as `FindingRow`; VS Code link uses `rootPath`; GitHub link uses `repoName`).
- Increase `FileRow` touch target height and font size (`text-label`) — these are primary navigation items, not metadata.
- Inspector panel gets a left-side shadow (`shadow-panel`) so it visually floats above the canvas.
- Add a **vertical activation tab** fixed to the right edge of the viewport, near the top. Clicking it toggles inspector visibility independently of selection — if nothing is selected the inspector opens in an empty state (consistent with VS Code's Properties/Files panel behaviour). Tab shows a `‹` icon when inspector is open, `›` when closed. Tab also has a shadow.
- When the inspector is open and `inspectorOpen` is true, pass `availableWidth = containerWidth - 360` to the layout algorithm so nodes are positioned within the unobstructed area. Reflow only on explicit navigation (level change); toggling the inspector panel does not trigger reflow.

**Status bar tabs**
- Tabs are fixed-width (not `flex: 1`) with generous horizontal padding. They are positioned in the **bottom-left** corner of the canvas area, not spanning the full width.
- Remove the `borderTop` active indicator. Instead use box shadows to create depth: the active tab looks coplanar with the canvas (shadow on sides, no shadow on top edge); inactive tabs look recessed (uniform soft shadow underneath). No coloured highlight on any tab.
- Info row (breadcrumb + counts) moves to the remaining space in the bottom bar to the right of the tabs, or stacks below on narrow viewports.

**Typography**
- All inspector section headers, tab labels, and clickable items: `text-label` (1.25em).
- All metadata, descriptions, file paths, and counts: `text-detail` (1em).
- Apply consistently across Inspector, StatusBar, Tooltip, and SearchBar.

**Canvas**
- At zoom level 3 (Files), file nodes are coloured by `riskSeverity` using the same heatmap as module nodes (`complexityColor(risk_score)`). Previously they were a flat green.

**Interaction model**
- **Single click** on a canvas node: selects the node, updates the inspector, does **not** change zoom level, does **not** push to history.
- **Double-click** on a canvas node: navigates (changes zoom level — z=1→z=2, z=2→z=3) and pushes one history entry.
- **Inspector navigation links** (imports, imported-by, files, "in module"): single click navigates and pushes history (these are explicit navigation actions, not passive selection).
- **Background / whitespace click**: deselects at every zoom level (closes inspector, clears `selectedId`). Consistent behaviour across z=1, z=2, z=3.
- **History model**: only zoom level transitions push to `history`. Selection changes within a level are in-memory state only. The browser back button navigates level transitions; it does not undo individual node selections.

**Hover glows**
- All navigable items (canvas nodes, inspector file rows, import/imported-by nav items, "in module" link, activation tab) show a subtle glow on hover. Use a consistent `hover:` Tailwind variant; the glow colour should match the item's risk/severity colour where applicable, or default to a soft blue-white for neutral items.

**Search bar**
- Add `2em` top margin to the search bar so it doesn't sit flush against the top of the viewport.

**Future notes (not in this phase)**
- *Command palette*: a floating non-movable toolbar (two icon buttons — Home to z=1, Reflow to re-run layout for current viewport) in the style of CAD application viewport controls.
- *Selection vs. navigation*: the single/double-click model introduced here is the foundation; future phases may extend it (e.g. multi-select, range highlight).

### Phase 7 - Modules-JS — package.json-aware JS/TS module detection

The ingestion pipeline currently assigns TypeScript and JavaScript files to modules
using the full directory path (e.g. `ui/src/components`). This is correct for pure
TypeScript repos but produces a fragmented picture for mixed-language repos like this
one: `vite.config.ts` lands in a `ui` module of one file while the actual app code
splits into `ui/src`, `ui/src/components`, `ui/src/hooks`, etc.

A `package.json` file is the canonical marker of a self-contained JS/TS project. When
one appears in a non-root subdirectory, all TypeScript and JavaScript files within that
subtree belong to one logical unit.

**Rule:**
- Non-root `package.json` → all TS/JS files in that subtree share one module named after the project directory (e.g. all `ui/**/*.ts(x)` → module `ui`).
- No non-root `package.json` (pure TS/JS repo, or individual files outside any project) → fall back to directory-level granularity (same as Go).
- Do not descend into `.gitignore`d trees like `node-modules`.

**Implementation — `internal/ingestion/ingestion.go`:**
- Add `findJSProjectRoots(root string) []string`: walk the tree, collect repo-relative slash paths of non-root directories that contain `package.json`. Respect the same `skipDirs` and hidden-dir rules as the main walker.
- Update `moduleID(relPath, language string, jsProjectRoots []string) string`: for `typescript`/`javascript`, check whether the file is under a project root prefix first; if so, return that prefix. Otherwise, fall back to `strings.LastIndex` (directory-level).
- In `Ingest()`: call `findJSProjectRoots(root)` after resolving the repo root, thread the result through every `moduleID` call.

**Tests:**
- Update `TestModuleID` to pass `jsProjectRoots`; add cases for project-root grouping and the directory-level fallback.
- Add `TestIngest_TSProjectRootGrouping`: temp repo with `ui/package.json` + TS files at multiple depths; assert all collapse into module `ui`.
- Existing `TestIngest_TypeScriptPackageLevelModules` (no `package.json` present) continues to verify the directory-level fallback.

---

### Phase 8 - Dep-Audit — basic dependency vulnerability audit

Surface known dependency vulnerabilities in `analysis.json` (UI display deferred to a later phase).

**JS/TS:** Read `dependencies` and `devDependencies` from each `package.json` found by `findJSProjectRoots()`. Supplement with `npm audit --json` findings where npm is available. Map npm severities: `critical`/`high` → `high`, `moderate` → `medium`, `low`/`info` → `low`. Emit as `schema.Risk` entries with `type: "dep-vuln"`, `module` = project root module (e.g. `ui`), `file` = `package.json` path. Non-fatal if npm unavailable — add `"npm-audit"` to `skipped_passes`.

**Go:** Use `govulncheck` against `go.sum`. Map its findings to `schema.Risk` similarly. Non-fatal if govulncheck unavailable. Human note: Go implementation left to AI.

**Output:** `file_deps` and vulnerability risks appear in `analysis.json` only; no UI changes in this phase.

---

### Phase 9 - Static-Analysis Audit — broader tooling survey

Before committing to further AI analysis investment, audit what existing static analysis tools can provide faster and cheaper. Think broadly: linters, SAST tools, complexity analyzers, license scanners, dead-code detectors. Identify the highest-signal tools per language already present in the ecosystem and evaluate integrating their output as additional risk passes.

---

### Phase 10 - Material UI migration

*(Supersedes the Future note below.)*

Port the surrounding shell UI components to Material UI. See **Future** section below for detail.

---

### Phase 11 - Inspector richness

Deepen the Inspector panel. Specific scope to be defined after the architectural review and Phase 9 findings.

---

## Future

- **Material UI component library**: replace the hand-rolled Tailwind UI components (Inspector, SearchBar, StatusBar, Tooltip) with Material UI equivalents. MUI's `Drawer`, `TextField`, `Chip`, `Tooltip`, `Table`, and theming system would clean up a large amount of rough edge-case styling and improve accessibility out of the box. Requires defining a dark-mode MUI theme that matches the existing design token palette. The SVG graph canvas (`MapCanvas`) would remain unchanged — MUI is for the surrounding shell only.

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
- `useAnalysis`, `useAppNav`, `useZoom` are the only hooks with side effects; all other state is derived
- ESLint must report zero errors and zero warnings before each commit
- Prettier must be clean before each commit

---

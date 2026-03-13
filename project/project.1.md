# Codebase MRI — Build Specification

## What This Is

A CLI tool written in Go that analyzes a software repository and produces a structured diagnostic report. The CLI is the engine. A UI layer will consume its output later.

One command. One output directory. One canonical JSON artifact.

---

## Command Interface

```bash
repo-mri analyze https://github.com/org/repo
repo-mri analyze .
```

Output:

```
.repo-mri/
  analysis.json
  report.md
```

---

## Project Structure

```
repo-mri/
  cmd/
    repo-mri/
      main.go
  internal/
    ingestion/
    analysis/
    providers/
      anthropic.go
      openai.go
    aggregation/
    report/
  schema/
    analysis.go
```

---

## Build Order

Complete each phase fully before moving to the next.

### Phase 1 — Ingestion

- Accept a GitHub URL or local path as input
- Clone remote repos to a temp directory
- Walk the file tree, filter non-code files
- Detect languages per file
- Parse import statements to build a dependency graph
- Collect: file count, language breakdown, directory hierarchy, import relationships

### Phase 1a — Portable Binary Distribution

Goals: produce self-contained, portable binaries that can be used for manual testing and distribution without requiring a local Go toolchain.

**Checklist:**
- [x] Add `--version` flag to CLI via cobra's `rootCmd.Version`; embed version string, git commit, and build date through `ldflags`
- [x] Update `Makefile`:
  - `build`: native binary → `bin/repo-mri`, `CGO_ENABLED=0`, stripped (`-ldflags "-s -w -X ..."`)
  - `dist`: cross-compile → `dist/` for `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`
  - `install`: copy native binary to `/usr/local/bin/repo-mri`
  - `version`: print resolved version string
- [x] Version derived from `git describe --tags --always --dirty`; falls back to `dev` when no tags exist
- [x] Binary names in `dist/`: `repo-mri-<os>-<arch>` (Windows: `repo-mri-windows-amd64.exe`)
- [x] Update `README.md`: installation section covering pre-built binary usage, `make install`, and `make dist`

**Constraints:**
- `CGO_ENABLED=0` on all targets — pure Go, no libc dependency
- No code changes beyond version wiring in `main.go`

### Phase 2 — Static Analysis (no AI)

Compute cheap signals before invoking any model:

- Dependency graph between modules
- File sizes and line counts
- Cyclomatic complexity per file (best effort)
- Most imported files
- Deepest dependency chains

### Phase 3 — Provider Wiring

Both Anthropic and OpenAI implement this interface:

```go
type AnalysisProvider interface {
    RunPass(ctx context.Context, pass PassType, chunks []FileChunk) ([]Finding, error)
}
```

Provider is selected via environment variable:

```bash
REPO_MRI_ANTHROPIC_KEY
REPO_MRI_OPENAI_KEY
```

If both are set, default to Anthropic. If neither is set, exit with a clear error message.

Models:
- Anthropic: `claude-sonnet-4-20250514`
- OpenAI: `gpt-4o`

### Phase 4 — AI Reasoning Passes

Run three specialized passes. Each pass receives chunked file content and returns structured findings.

**Chunking rules:**
- Bug and security passes: file-level chunks, max 50 files per chunk
- Architecture pass: synthesized import graph summary only, not raw file content

**Passes:**

| Pass | Focus |
|---|---|
| `architecture` | Circular dependencies, layer violations, dependency anomalies |
| `bug` | Complexity hotspots, error handling gaps, fragile logic paths |
| `security` | Hardcoded secrets, injection risks, auth gaps |

Each pass returns an array of findings matching the risk schema below.

### Phase 5 — Aggregation

- Merge findings from all passes
- Deduplicate overlapping findings (same file + same issue type)
- Compute `risk_score` per file (0.0–1.0) from finding severity and confidence
- Compute `risk_score` per module as average of its files
- Rank modules by risk score descending

### Phase 6 — Report Generation

Write two files to `.repo-mri/`:

**`analysis.json`** — canonical structured artifact (schema below)

**`report.md`** — human-readable summary including:
- Repository overview (name, languages, file count)
- Overall health score (0–100, inverse of mean risk)
- Top 10 riskiest modules
- All high-severity findings grouped by module
- Security findings section

---

## Canonical Data Schema

### Top-Level

```json
{
  "meta": {},
  "repo": {},
  "modules": [],
  "dependencies": [],
  "risks": [],
  "files": [],
  "subsystems": []
}
```

### meta

```json
{
  "meta": {
    "schema_version": "1.0",
    "cli_version": "0.1.0",
    "model_used": "claude-sonnet-4-20250514",
    "provider": "anthropic",
    "analysis_duration_ms": 42300
  }
}
```

### repo

```json
{
  "repo": {
    "name": "example-repo",
    "languages": ["python", "typescript"],
    "file_count": 214,
    "module_count": 12,
    "analysis_time": "2026-03-11T18:00:00Z"
  }
}
```

### modules

```json
{
  "modules": [
    {
      "id": "payments",
      "path": "src/payments",
      "language": "python",
      "file_count": 17,
      "risk_score": 0.63,
      "complexity_score": 0.72
    }
  ]
}
```

### dependencies

```json
{
  "dependencies": [
    {
      "from": "payments",
      "to": "database",
      "type": "import"
    }
  ]
}
```

### risks

```json
{
  "risks": [
    {
      "id": "risk_001",
      "severity": "high",
      "type": "architecture",
      "pass": "architecture",
      "module": "payments",
      "file": "src/payments/processor.py",
      "title": "Circular dependency detected",
      "description": "processor imports ledger and ledger imports processor",
      "confidence": 0.82,
      "evidence_lines": [43, 44, 45]
    }
  ]
}
```

Severity values: `high`, `medium`, `low`
Type values: `architecture`, `bug`, `security`

### files

```json
{
  "files": [
    {
      "path": "src/payments/processor.py",
      "module": "payments",
      "language": "python",
      "lines": 842,
      "complexity": 0.78,
      "risk_score": 0.86
    }
  ]
}
```

### subsystems (optional)

```json
{
  "subsystems": [
    {
      "id": "payment_pipeline",
      "modules": ["payments", "ledger", "queue"]
    }
  ]
}
```

---

## Error Handling

- Missing API keys: exit with message listing which env vars are required
- Clone failure: exit with the git error
- Empty or non-code repository: exit with clear message
- AI pass failure: log the error, skip the pass, continue with remaining passes, note skipped passes in `meta`
- Partial results are acceptable — always write output if any analysis succeeded
---

## Post-Phase-6 Improvements

*Human-authorized addition. Phases 7, 7a, 8, and 9 must be completed before UI work begins. Phase 10 is deferred pending a spike.*

---

### Phase 7 — CLI Hardening

Low-risk, high-impact fixes. No schema changes. Complete all items before moving to Phase 7a.

**7a — Exclude test files from AI passes**

- In `internal/analysis/passes.go`, filter `buildFileChunks` to skip files matching test file patterns
- Go: `strings.HasSuffix(f.Path, "_test.go")`
- JS/TS: `*.test.ts`, `*.test.js`, `*.spec.ts`, `*.spec.js`
- Ruby: `*_spec.rb`
- Test files remain in static analysis (walker, complexity); exclusion applies only to `buildFileChunks`
- Add test case: verify `_test.go` files do not appear in any chunk returned by `buildFileChunks`

**7b — Token budget enforcement in chunking**

- Replace the fixed 50-file chunk limit with a dual constraint: max 50 files OR max 80,000 characters of content per chunk, whichever is hit first
- 80k chars ≈ ~20k tokens, well within model context with headroom for system prompt and response
- `chunkSize` constant remains as the file-count ceiling; add `chunkCharLimit = 80_000` constant alongside it
- Add test case covering a chunk that hits the character limit before the file limit

**7c — Context timeout on AI pipeline**

- Add `--timeout` flag to the `analyze` command (default: `5m`, type: `time.Duration`)
- Wrap `context.Background()` with `context.WithTimeout` before passing to `ingestion.Ingest` and downstream
- Deferred cancel must be called on all exit paths
- Add test: verify a pre-cancelled context surfaces as an error from `runAnalyze`

**7d — Fix `moduleForFile` longest-prefix matching**

- Current implementation returns the first module whose `Path` is a prefix of the file path; overlapping paths (e.g. `src/pay` and `src/payment`) can match the wrong module
- Fix: iterate all modules, track the match with the longest `m.Path`, return that
- Add table-driven test cases covering overlapping path prefixes
- This item is a prerequisite for Phase 8

**7e — Make `graph-summary` a shared constant**

- Define `const GraphSummaryPath = "graph-summary"` in `internal/providers/provider.go`
- Replace all string literals `"graph-summary"` and `"graph"` in `passes.go` and `main.go` with the constant
- No behaviour change; eliminates silent drift risk if the synthetic chunk name ever changes

---

### Phase 7a — Prompt Tuning

*Prerequisite: Phase 7 must be merged before beginning Phase 7a. This phase requires human evaluation of false-positive rate before and after — do not merge without that review.*

- Prepend a contextual preamble to `buildUserMessage` describing the repo type and intentional patterns
- Preamble is derived from `schema.Analysis` at call time: language list, a note that this is a CLI tool, and explicit acknowledgement that non-fatal error handling and `#nosec` annotations are intentional
- Define the static portion as a constant in `anthropic.go`; append the dynamic portion (languages) at call time
- Measurement: run `repo-mri analyze .` before and after; compare finding counts and false-positive rate; human reviews output before merge

---

### Phase 8 — Go Package-Level Module Granularity

*Prerequisite: Phase 7d (longest-prefix fix) must be merged before beginning Phase 8.*

The most impactful CLI improvement before UI work. Go import paths are already package-scoped in the AST output; only the module-assignment logic in ingestion needs to change.

- In `internal/ingestion/ingestion.go`, change module assignment for Go files: use the repo-relative directory path as the module ID rather than the top-level directory name
- `internal/ingestion` and `internal/analysis` become distinct modules instead of both collapsing into `internal`
- Module ID = repo-relative directory path (e.g. `internal/ingestion`); Module Path = same value
- Dependency edges are already at import-path granularity from the AST parser; no changes needed to `imports.go`
- Non-Go repos are unaffected; the existing top-level directory heuristic remains the fallback for all other languages
- Update `ingestion_test.go` with a Go repo fixture that verifies package-level module splitting
- Schema change: none required; `Module.ID` and `Module.Path` already support arbitrary strings
- Bump `SchemaVersion` from `"1.0"` to `"1.1"` in `schema/analysis.go` on merge
- Create `CHANGELOG.md` documenting the breaking change to `modules[]` and `dependencies[]` for Go repos

---

### Phase 9 — Architecture Finding Target Model

*Must not begin until Phase 7 is complete. No dependency on Phase 8.*

- Add `TargetType string` and `TargetID string` fields to `schema.Risk`
  - Valid `TargetType` values: `"file"`, `"module"`, `"repository"`
  - `TargetID` is the file path, module ID, or repo name respectively
- Architecture pass findings set `TargetType: "repository"`, `TargetID: a.Repo.Name`
- Bug and security pass findings set `TargetType: "file"`, `TargetID: <file path>`
- Update `report.go` to use `TargetType` for section routing rather than the `Type` field string comparison
- Add table-driven tests covering all three target types
- Bump `SchemaVersion` to `"1.2"` in `schema/analysis.go` on merge
- Update `CHANGELOG.md`

---

### Phase 10 — Structural Analysis (Deferred)

*Do not begin until UI work is underway and there is a visual consumer for the output.*

Before committing to structural embeddings and a vector store (a significant new infrastructure dependency), run a spike to determine whether k-means clustering on existing static metrics — complexity score, import count, file count, dependency edge count — provides sufficient clustering signal without new dependencies.

- **Spike deliverable:** a standalone script that reads `analysis.json` and outputs proposed clusters using only the data already present
- If the spike demonstrates useful clustering: implement in-process in Go, no new dependencies
- If the spike reveals embeddings are necessary: evaluate vector store options and reopen planning before committing to an approach
- Structural embeddings design notes are captured in `docs/phase10-embeddings-preview.md` for reference

---

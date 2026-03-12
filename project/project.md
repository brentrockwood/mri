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
- [ ] Add `--version` flag to CLI via cobra's `rootCmd.Version`; embed version string, git commit, and build date through `ldflags`
- [ ] Update `Makefile`:
  - `build`: native binary → `bin/repo-mri`, `CGO_ENABLED=0`, stripped (`-ldflags "-s -w -X ..."`)
  - `dist`: cross-compile → `dist/` for `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`
  - `install`: copy native binary to `/usr/local/bin/repo-mri`
  - `version`: print resolved version string
- [ ] Version derived from `git describe --tags --always --dirty`; falls back to `dev` when no tags exist
- [ ] Binary names in `dist/`: `repo-mri-<os>-<arch>` (Windows: `repo-mri-windows-amd64.exe`)
- [ ] Update `README.md`: installation section covering pre-built binary usage, `make install`, and `make dist`

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

# Changelog

## Unreleased

### Added

#### Architecture finding target model (`risks[]`)

**Schema version: 1.1 → 1.2**

`schema.Risk` has two new optional fields:

```json
{
  "target_type": "file" | "module" | "repository",
  "target_id":   "<file path | module ID | repo name>"
}
```

- Architecture pass findings: `target_type: "repository"`, `target_id: <repo name>`
- Bug and security pass findings: `target_type: "file"`, `target_id: <file path>`

`report.md` section routing now uses `target_type`:

- Architecture findings (`target_type: "repository"`) appear in a new **Architecture Findings** section.
- High-severity section excludes repository-level findings.
- Security section filter updated to `target_type: "file" && type: "security"`.

**Consumers of `analysis.json`** can use `target_type` to distinguish repository-level findings from file-level ones without inspecting the `type` field.

---

### Breaking changes

#### Go module granularity (`modules[]` and `dependencies[]`)

**Schema version: 1.0 → 1.1**

In `analysis.json`, Go repositories now produce package-level modules instead of
top-level-directory modules.

**Before (schema 1.0):**

```json
"modules": [
  { "id": "internal", "path": "internal", ... },
  { "id": "cmd",      "path": "cmd",      ... }
]
```

**After (schema 1.1):**

```json
"modules": [
  { "id": "internal/analysis",   "path": "internal/analysis",   ... },
  { "id": "internal/ingestion",  "path": "internal/ingestion",  ... },
  { "id": "internal/providers",  "path": "internal/providers",  ... },
  { "id": "cmd/repo-mri",        "path": "cmd/repo-mri",        ... }
]
```

Dependency edges in `dependencies[]` reflect the same change — `from` and `to`
values now use the full package path.

**Non-Go repositories are unaffected.** They continue to use the top-level
directory as the module ID.

**Consumers of `analysis.json`** that assume single-segment module IDs for Go
repos must be updated to handle slash-separated paths.

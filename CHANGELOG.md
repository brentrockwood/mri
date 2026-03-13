# Changelog

## Unreleased

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

# mri

[One paragraph description of what this project does, for whom, and why it exists.]

---

# Implementation Plan

> **This document is write-locked after the initial planning session.**
> Updates require explicit human authorization. See `doa.md`.

## Stack & Framework Decisions

### Language & Runtime
- **[e.g. Python 3.12, Node 20, Go 1.22]**

### Architecture
- **[e.g. CLI script, REST API, event-driven service]**
- Key characteristics: [unattended operation, stateless, etc.]

### Testing Framework
- **[e.g. pytest, Jest, go test]**

### Code Style & Quality
- **[Style guide, e.g. PEP 8, Google style]**
- **[Formatter, e.g. black, prettier]**
- **[Linter, e.g. ruff, eslint]**
- **[Type checker, e.g. mypy, tsc]**

### Security Scanning
- `scripts/security_scan.sh` (grep-based, no external deps)
- [Optional: trufflehog, gitleaks, etc.]

### Dependency Management
- **[e.g. venv + pip, npm, go modules]**
- Pin to major versions unless otherwise noted

---

## Technical Components

### 1. [Component Name]
- [What it does]
- [Key interfaces or dependencies]

### 2. [Component Name]
- [What it does]
- [Key interfaces or dependencies]

### 3. [Component Name]
- [What it does]
- [Key interfaces or dependencies]

---

## Implementation Phases

### Phase 1: Foundation
- [ ] Project structure
- [ ] Environment and dependency setup
- [ ] Configuration management (secrets.env pattern)
- [ ] Logging setup
- [ ] [Other foundation tasks]

### Phase 2: Core [Feature]
- [ ] [Task]
- [ ] [Task]
- [ ] Unit tests

### Phase 3: [Next Feature]
- [ ] [Task]
- [ ] [Task]
- [ ] Tests

### Phase 4: Production Readiness
- [ ] Error handling and retry logic
- [ ] Dry-run mode
- [ ] Security scan clean
- [ ] Full test suite passing
- [ ] Documentation complete
- [ ] Deployment guide

---

## Decisions & Rationale

### [Decision topic]
- **Decision:** [What was decided]
- **Rationale:** [Why]
- **Alternatives considered:** [What else was on the table]

### [Another decision]
- **Decision:** [What was decided]
- **Rationale:** [Why]

---

## Dependencies

### Core
- [package] — [purpose]

### Dev / Testing
- [package] — [purpose]

---

## Project Structure

```
[project]/
├── src/
│   ├── [module].py
│   └── [module].py
├── tests/
│   └── test_[module].py
├── scripts/
│   ├── run_checks.sh
│   └── security_scan.sh
├── docs/
│   └── DEPLOYMENT.md
├── project/
│   ├── doa.md
│   ├── project.md       ← this file
│   ├── context.md
│   └── scripts/
├── main.py              # or index.js, etc.
├── .env.example
├── secrets.env.example
└── requirements.txt     # or package.json, go.mod, etc.
```

---

## Security Considerations

- Secrets in `secrets.env` only (gitignored)
- No credentials, IPs, or tokens in source files
- `scripts/security_scan.sh` runs before every push
- [Any project-specific security notes]

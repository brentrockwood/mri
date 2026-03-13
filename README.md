# repo-mri

A CLI tool that analyzes a software repository and produces a structured diagnostic report. Point it at a GitHub URL or a local path and it outputs a JSON artifact and a human-readable Markdown summary.

## Requirements

- Go 1.23+ (to build from source)
- `git` (for cloning remote repositories)

No API keys are required for Phase 1 (ingestion only). Later phases will require `REPO_MRI_ANTHROPIC_KEY` or `REPO_MRI_OPENAI_KEY`.

## Installation

### Pre-built binary (recommended)

Download the binary for your platform from the `dist/` directory of a release, make it executable, and move it onto your `PATH`:

```bash
# macOS (Apple Silicon)
curl -Lo repo-mri https://github.com/brentrockwood/mri/releases/latest/download/repo-mri-darwin-arm64
chmod +x repo-mri
sudo mv repo-mri /usr/local/bin/

# macOS (Intel)
curl -Lo repo-mri https://github.com/brentrockwood/mri/releases/latest/download/repo-mri-darwin-amd64
chmod +x repo-mri
sudo mv repo-mri /usr/local/bin/

# Linux (amd64)
curl -Lo repo-mri https://github.com/brentrockwood/mri/releases/latest/download/repo-mri-linux-amd64
chmod +x repo-mri
sudo mv repo-mri /usr/local/bin/
```

### Build and install from source

```bash
git clone https://github.com/brentrockwood/mri.git
cd mri
make install        # builds and copies to /usr/local/bin/repo-mri
```

### Build only (no install)

```bash
make build          # produces bin/repo-mri
./bin/repo-mri --version
```

## Usage

```bash
repo-mri analyze https://github.com/org/repo
repo-mri analyze /path/to/local/repo
repo-mri analyze .
```

Output is written to `.repo-mri/` in the current working directory:

```
.repo-mri/
  analysis.json   # canonical structured artifact
  report.md       # human-readable Markdown summary
  report.html     # interactive visual report — open in any browser
```

Open the HTML report directly from disk (no web server required):

```bash
open .repo-mri/report.html          # macOS
xdg-open .repo-mri/report.html      # Linux
start .repo-mri/report.html         # Windows
```

## Development

### Prerequisites

- Go 1.23+
- Node.js 18+ and npm (for the UI)
- `golangci-lint` (`brew install golangci-lint`)
- `goimports` (`go install golang.org/x/tools/cmd/goimports@latest`)
- `gosec` (`go install github.com/securego/gosec/v2/cmd/gosec@latest`)

### Common tasks

```bash
make build          # build UI then compile native binary → bin/repo-mri
make ui-build       # build UI only → internal/report/static/report.html
make ui-dev         # start Vite dev server for UI development
make test           # go test -race -count=1 ./...
make lint           # golangci-lint run ./...
make vet            # go vet ./...
make fmt            # goimports -w .
make dist           # cross-compile all platforms → dist/
make install        # build + install to /usr/local/bin
make clean          # remove bin/ and dist/
make version        # print resolved version string
```

### UI development workflow

The interactive HTML report is a single-file React app built with Vite. During development, run the dev server against a real analysis output:

```bash
repo-mri analyze .                  # produces .repo-mri/analysis.json
make ui-dev                         # starts http://localhost:5173
```

The dev server reads `analysis.json` via `window.__MRI_DATA__` injected from the file. When you're done, `make build` bakes the compiled UI into the binary via `//go:embed`.

### Cross-compilation targets

`make dist` produces binaries for:

| Platform | Output |
|---|---|
| macOS (Intel) | `dist/repo-mri-darwin-amd64` |
| macOS (Apple Silicon) | `dist/repo-mri-darwin-arm64` |
| Linux (amd64) | `dist/repo-mri-linux-amd64` |
| Linux (arm64) | `dist/repo-mri-linux-arm64` |
| Windows (amd64) | `dist/repo-mri-windows-amd64.exe` |

All binaries are built with `CGO_ENABLED=0` (no libc dependency) and stripped of debug symbols.

## Project Structure

```
repo-mri/
├── cmd/repo-mri/       # CLI entry point
├── internal/
│   ├── ingestion/      # Phase 1: file walking, language detection, import parsing
│   ├── analysis/       # Phase 2: static analysis (planned)
│   ├── providers/      # Phase 3: AI provider wiring (planned)
│   ├── aggregation/    # Phase 5: finding aggregation (planned)
│   └── report/         # Phase 6: report generation (planned)
├── schema/             # Canonical data schema
├── scripts/            # Security scan and other utilities
├── project/            # Planning and work log (not user-facing)
│   ├── doa.md          # Development Operating Agreement
│   ├── project.md      # Implementation plan (write-locked)
│   ├── context.md      # Session work log
│   └── scripts/        # Context management utilities
└── Makefile
```

## Security

- Never commit secrets or API keys — load them from environment variables only
- Run `./scripts/security_scan.sh` before each push

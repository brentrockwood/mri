#!/usr/bin/env bash
# scripts/run_checks.sh
# Run all quality checks for a Go project.
# Exit codes: 0=all passed, 1=one or more checks failed, 2=script error.
#
# Steps (in order):
#   1. Security scan     (scripts/security_scan.sh)
#   2. go fmt / goimports
#   3. go vet
#   4. golangci-lint
#   5. go test -race
#   6. go build

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
FAILED=()

run_step() {
  local name="$1"
  shift
  echo ""
  echo "==> $name"
  if "$@"; then
    echo "    ✓ $name passed"
  else
    echo "    ✗ $name FAILED"
    FAILED+=("$name")
  fi
}

cd "$REPO_ROOT"

# 1. Security scan
if [[ -x scripts/security_scan.sh ]]; then
  run_step "Security scan" bash scripts/security_scan.sh
else
  echo "==> Security scan: scripts/security_scan.sh not found, skipping"
fi

# 2. Format check
run_step "goimports (format)" bash -c '
  unformatted=$(goimports -l . 2>/dev/null)
  if [[ -n "$unformatted" ]]; then
    echo "Unformatted files:"
    echo "$unformatted"
    exit 1
  fi
'

# 3. go vet
run_step "go vet" go vet ./...

# 4. golangci-lint
if command -v golangci-lint &>/dev/null; then
  run_step "golangci-lint" golangci-lint run ./...
else
  echo "==> golangci-lint: not installed, skipping (install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)"
fi

# 5. go test -race
run_step "go test -race" go test -race -count=1 ./...

# 6. go build
run_step "go build" go build ./...

# Summary
echo ""
if [[ ${#FAILED[@]} -eq 0 ]]; then
  echo "All checks passed."
  exit 0
else
  echo "The following checks failed:"
  for f in "${FAILED[@]}"; do
    echo "  - $f"
  done
  exit 1
fi

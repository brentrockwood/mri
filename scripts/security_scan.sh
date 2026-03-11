#!/usr/bin/env bash
#
# Security scan: checks for secrets, credentials, and sensitive patterns
# that should never be committed to version control.
#
# Exit codes:
#   0 - No issues found
#   1 - Issues detected
#   2 - Script error

set -eo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Security Scanner"
echo "================"
echo ""
echo "Scanning: $PROJECT_ROOT"
echo ""

ISSUES_FOUND=0

# ── Patterns ─────────────────────────────────────────────────────────────────
# Each entry: "Label|regex"
PATTERNS=(
    "API Keys|(api[_-]?key|apikey|api[_-]?secret)['\"]?\s*[:=]\s*['\"]?[A-Za-z0-9_-]{20,}"
    "AWS Keys|(aws[_-]?access[_-]?key[_-]?id|aws[_-]?secret)['\"]?\s*[:=]\s*['\"]?[A-Za-z0-9/+=]{20,}"
    "Private Keys|-----BEGIN (RSA |DSA |EC )?PRIVATE KEY-----"
    "Tokens|(token|auth[_-]?token|access[_-]?token)['\"]?\s*[:=]\s*['\"]?[A-Za-z0-9_.-]{20,}"
    "Passwords|(password|passwd|pwd)['\"]?\s*[:=]\s*['\"]?[^'\"\\s]{8,}"
    "OpenAI Keys|sk-[A-Za-z0-9]{48}"
    "Anthropic Keys|sk-ant-[A-Za-z0-9-]{95}"
    "Database URLs|(postgres|mysql|mongodb)://[^\\s'\"]*:[^\\s'\"]*@"
    "IP Addresses|([0-9]{1,3}\.){3}[0-9]{1,3}"
)

# ── Exclusions ────────────────────────────────────────────────────────────────
GREP_EXCLUDES="--exclude-dir=.git --exclude-dir=__pycache__ --exclude-dir=node_modules \
  --exclude-dir=.venv --exclude-dir=venv --exclude-dir=.mypy_cache \
  --exclude=*.example --exclude=*_test.py --exclude=test_*.py \
  --exclude=*.md --exclude=security_scan.sh --exclude=*.json"

scan_pattern() {
    local label=$1
    local pattern=$2
    local results

    results=$(cd "$PROJECT_ROOT" && grep -rn -E -i $GREP_EXCLUDES "$pattern" . 2>/dev/null || true)

    if [ -n "$results" ]; then
        echo -e "${RED}✗ Potential $label found:${NC}"
        echo "$results" | sed 's/^/  /'
        echo ""
        ISSUES_FOUND=$((ISSUES_FOUND + 1))
    fi
}

# ── Sensitive file check ──────────────────────────────────────────────────────
echo "Checking for sensitive files..."

SENSITIVE_FILES=(
    "secrets.env"
    "credentials.json"
    "token.json"
    "*.pem"
    "*.key"
    "id_rsa"
    "id_dsa"
)

for file_pattern in "${SENSITIVE_FILES[@]}"; do
    found=$(cd "$PROJECT_ROOT" && find . -name "$file_pattern" \
        -not -path "./.git/*" -not -path "./.venv/*" -not -path "./venv/*" 2>/dev/null || true)

    if [ -n "$found" ]; then
        echo -e "${YELLOW}⚠ Found: $file_pattern${NC}"
        echo "$found" | sed 's/^/  /'
        while IFS= read -r file; do
            if git -C "$PROJECT_ROOT" check-ignore "$file" >/dev/null 2>&1; then
                echo -e "  ${GREEN}✓ Properly gitignored${NC}"
            else
                echo -e "  ${RED}✗ NOT gitignored — fix this immediately${NC}"
                ISSUES_FOUND=$((ISSUES_FOUND + 1))
            fi
        done <<< "$found"
        echo ""
    fi
done

# ── Pattern scan ─────────────────────────────────────────────────────────────
echo "Scanning for sensitive patterns..."

for entry in "${PATTERNS[@]}"; do
    label="${entry%%|*}"
    pattern="${entry#*|}"
    scan_pattern "$label" "$pattern"
done

# ── Git history check ─────────────────────────────────────────────────────────
echo "Checking git history..."

LEAKED=$(cd "$PROJECT_ROOT" && git log --all --full-history --pretty=format: --name-only \
    | grep -E "(secrets\.env|credentials\.json|token\.json|\.pem|\.key)" \
    | grep -v "\.example" || true)

if [ -n "$LEAKED" ]; then
    echo -e "${RED}✗ Sensitive files found in git history:${NC}"
    echo "$LEAKED" | sort -u | sed 's/^/  /'
    echo ""
    echo -e "${YELLOW}Consider using BFG Repo-Cleaner or git filter-repo to remove them.${NC}"
    echo ""
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
fi

# ── Summary ──────────────────────────────────────────────────────────────────
echo "================================"
if [ $ISSUES_FOUND -eq 0 ]; then
    echo -e "${GREEN}✓ Security scan passed.${NC}"
    exit 0
else
    echo -e "${RED}✗ $ISSUES_FOUND issue(s) found.${NC}"
    echo ""
    echo "  1. Remove hardcoded secrets from source files"
    echo "  2. Move secrets to secrets.env (gitignored)"
    echo "  3. Ensure sensitive files are in .gitignore"
    echo "  4. If secrets were committed, rotate them immediately"
    echo ""
    exit 1
fi

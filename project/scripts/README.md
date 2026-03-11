# Context Management Scripts

Three bash scripts for managing structured context log files with automatic rotation and efficient reading. These scripts are the exclusive interface for `context.md` — never write to that file directly.

## Scripts

### `add-context`

Adds a properly formatted entry to a context file with auto-generated metadata.

**Usage:**
```bash
add-context --agent AGENT --model MODEL [OPTIONS] [BODY_TEXT]
```

**Required Parameters:**
- `--agent AGENT` — Agent name and version (e.g., `"Claude.ai"`, `"OpenCode/1.0"`)
- `--model MODEL` — Model name and version (e.g., `"claude-sonnet-4-5"`, `"gpt-4"`)

**Optional Parameters:**
- `--session SESSION` — Session identifier
- `--output FILE` — Output context file (default: `context.md`)
- `--file FILE` — Read body from file

**Body Input (priority order):**
1. Remaining arguments as body text (quoted)
2. `--file` to read from file
3. stdin if no body provided

**Auto-generated Fields:**
- `date` — ISO 8601 local date and time with timezone offset
- `hash` — Base64-encoded SHA-256 hash of body text
- `startCommit` — Git hash of most recent commit (if in a git repo)

**Examples:**
```bash
# Body as argument
add-context --agent "Claude.ai" --model "claude-sonnet-4-5" "Implemented retry logic for classifier"

# Body from file
add-context --agent "Claude.ai" --model "claude-sonnet-4-5" --file body.txt

# Body from stdin
cat body.txt | add-context --agent "Claude.ai" --model "claude-sonnet-4-5"

# With session and custom output file
add-context --agent "Claude.ai" --model "claude-sonnet-4-5" --session "abc123" \
  --output project/context.md "Context text"
```

**Important:** The `EOF` marker at the end of each entry is added automatically — do not include it in your body text.

---

### `read-context`

Reads the last N entries from a context file without loading the entire file into memory.

**Usage:**
```bash
read-context [OPTIONS]
```

**Options:**
- `-n, --num N` — Number of entries to read (default: `1`)
- `-f, --file FILE` — Context file to read (default: `context.md`)
- `-h, --headers-only` — Show only entry headers (date, hash, agent, model)
- `--help` — Show help message

**Behavior:**
- Reads entries in chronological order (oldest to newest of the N selected)
- Uses single-pass AWK parsing for efficiency
- Memory usage: only stores the requested N entries

**Examples:**
```bash
# Read last entry (default) — use at session start to resume context
read-context

# Read last 3 entries
read-context -n 3

# Show only headers of last entry
read-context --headers-only

# Show headers of last 5 entries
read-context -n 5 --headers-only

# Read from a specific file
read-context -f project/context-archive.md -n 2
```

**Use Cases:**
- **Session resumption:** See recent work without scrolling the full log
- **Debugging:** Check what happened in the last N interactions
- **Auditing:** Review recent context headers for integrity
- **Integration:** Parse headers programmatically for automation

---

### `rotate-context`

Rotates context files when they exceed a size limit, moving older entries to timestamped overflow files.

**Usage:**
```bash
rotate-context [OPTIONS]
```

**Options:**
- `--file FILE` — Context file to rotate (default: `context.md`)
- `--size BYTES` — Size limit in bytes (default: `1048576` = 1MB)
- `--keep N` — Number of recent entries to keep (default: `2`)

**Behavior:**
- Checks file size; if under limit, exits with code `1` (no rotation)
- If over limit and file has more than `--keep` entries, creates an overflow file
- Overflow filename: `<basename>-YYYY-MM-DDTHH_MM_SS±OFFSET.md`
- Original file is trimmed to retain only the last N entries
- Prints overflow filename to stdout on success (exit code `0`)
- Returns exit code `2` on errors

**Examples:**
```bash
# Use defaults (context.md, 1MB limit, keep 2 entries)
rotate-context

# Custom limits
rotate-context --file project/context.md --size 524288 --keep 3

# Capture the overflow filename
OVERFLOW=$(rotate-context)
if [ $? -eq 0 ]; then
    echo "Archived to: $OVERFLOW"
fi
```

---

## Entry Format

Each entry in the context file follows this structure:

```
---
date: 2026-01-25T13:44:33+0000
hash: KQjNYowLItHmgLl89cEJ4/5D+IMB3nkezcnIkaOTe2Q=
agent: Claude.ai
model: claude-sonnet-4-5
session: abc123
startCommit: a1b2c3d4
---

Body text goes here.
Can span multiple lines.
Any characters or whitespace.

EOF

```

**Notes:**
- A blank line is required between entries
- The `EOF` marker is on its own line, added automatically by `add-context`
- `session` and `startCommit` fields are optional
- Body text is hashed for integrity verification — do not edit entries after the fact

---

## Installation

```bash
chmod +x add-context read-context rotate-context
```

Scripts are designed to be called from `project/scripts/` relative to your project root. Optionally move to PATH for global access:

```bash
sudo mv add-context read-context rotate-context /usr/local/bin/
```

---

## Workflow Example

```bash
# Start of session: see where you left off
project/scripts/read-context -n 2

# Work happens...

# End of session: add a context entry
project/scripts/add-context --agent "Claude.ai" --model "claude-sonnet-4-5" \
  "Completed Phase 2 classifier. All tests passing. Next: wire up Gmail polling."

# Quick audit: review recent headers
project/scripts/read-context --headers-only -n 5

# Rotate if the log is getting large
project/scripts/rotate-context
```

---

## Integration Ideas

**Parse recent entry metadata programmatically:**
```bash
# Get dates from last 10 entries
read-context -n 10 --headers-only | grep "^date:" | cut -d' ' -f2-

# Get agent used in last entry
read-context --headers-only | grep "^agent:" | cut -d' ' -f2-
```

**Post-add rotation hook:**
```bash
# Wrapper that adds entry then rotates if needed
add-context "$@" && rotate-context || true
```

**Cron-based rotation:**
```bash
# Check and rotate daily
0 0 * * * /path/to/rotate-context --file /path/to/project/context.md
```

---

## Dependencies

- `bash`
- `openssl` (for SHA-256 hashing)
- `git` (optional, for `startCommit` field)
- Standard Unix tools: `awk`, `sed`, `grep`, `stat`

## Platform Notes

The scripts handle `stat` differences automatically:
- macOS: `stat -f%z`
- Linux: `stat -c%s`

---

## Exit Codes

**`add-context`:**
- `0` — Success
- `1` — Invalid arguments or missing required parameters
- `2` — File errors

**`read-context`:**
- `0` — Success
- `1` — Invalid arguments or file not found

**`rotate-context`:**
- `0` — Rotation performed (overflow filename printed to stdout)
- `1` — No rotation needed (file under size limit or insufficient entries)
- `2` — Error (file not found, etc.)

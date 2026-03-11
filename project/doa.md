# Development Operating Agreement

This document describes our working agreement on all projects. It is authoritative and must only be edited by humans.

---

## Do this every session

### If starting a new project

1. Create a `README.md` if it hasn't been created already. If you have any context as to the purpose and goals of the project, you may add them now. If not, a simple title derived from the folder name will do for now. As we progress through the project, you are free to update this file autonomously, always reporting any changes. Always include a setup section which includes such things as shell scripts to be run, environment variables to be set, dependencies required, and anything else necessary to get the user up and running with a minimum of manual intervention. `README.md` is end-user facing and should not influence operational choices. It is simply a deliverable that must be kept up to date like any other.

2. Create a `project.md` in the `project/` folder. This is where project level plans and checklists go, as well as the overall architecture, stack decisions, etc. It is our primary operational document. At this point, its contents will likely mirror the `README.md`. Further updates to `project.md` require human authorization and it should otherwise be considered write-locked.

3. Copy the `scripts/` directory from the DOA repository into `project/scripts/` if it doesn't already exist. These scripts manage `context.md` and must be present before the first entry is added.

4. Add the first context entry. Always run `add-context` from the project root, and always specify `--output project/context.md` explicitly:

   ```
   project/scripts/add-context \
     --agent "<agent>" \
     --model "<model>" \
     --output project/context.md \
     "Started <projectname> project. Branch: <branch>"
   ```

   The body can be brief at this stage. Always include the current branch name. This is our ongoing work log. Entries must never be edited after they have been entered. Consider it an auditable artifact.

### If resuming an existing project

1. Read `project/project.md`. This will only be updated after discussion and is considered write-locked unless given explicit human authorization. This is our primary operational document.

2. Read the last entry or two from `project/context.md` using the script:
   ```
   project/scripts/read-context -n 2
   ```
   You do not need to read the whole file — just enough to know where we left off in the previous session. Cap at 12K of text at the most. If you do not have enough context from that, ask for help.

3. **Verify you are on the correct feature branch. If not, create one now before doing anything else.** Do not begin implementation until this is confirmed.

4. If no context exists because we are using these standards for the first time on an existing project, copy the scripts into `project/scripts/`, review the existing codebase, and add an entry that summarizes the current state as well as you can.

This is our ongoing work log. Entries must never be edited after they have been entered. Consider it an auditable artifact.

---

## Context Management

Context entries are managed exclusively through scripts in `project/scripts/`. Never write to `context.md` directly.

### Adding entries

Use `project/scripts/add-context` with the following parameters:

**Required:**
- `--agent <n>` — Agent name and version (e.g., `"OpenCode"`, `"Claude Desktop"`, `"Claude.ai"`)
- `--model <version>` — Model name and version (e.g., `"claude-sonnet-4-5"`, `"gpt-4"`)
- `--output project/context.md` — Always specify this explicitly. The script defaults to `./context.md`, which will write to the wrong location if run from the project root.

**Body input methods** (in priority order):
- Direct argument: `add-context --agent "..." --model "..." --output project/context.md "Entry text"`
- From file: `add-context --agent "..." --model "..." --output project/context.md --file body.txt`
- From stdin: `echo "Entry" | add-context --agent "..." --model "..." --output project/context.md`
- Heredoc (preferred for multiline):
  ```bash
  cat << 'EOF' | project/scripts/add-context --agent "..." --model "..." --output project/context.md
  Your multiline
  entry text here.
  EOF
  ```

**Auto-generated fields:** date (ISO 8601 with timezone offset), SHA-256 hash of body, git commit hash.

**Entry guidelines:**
- The `EOF` marker at the end of each entry is a delimiter added automatically by the script — do not include it in your body text
- The script handles all formatting; your job is just the body content
- Always include the current branch name in the entry body
- Think: "What would I need to know if this session crashed right now?"

### Reading entries

Use `project/scripts/read-context` to inspect the log:

```bash
project/scripts/read-context           # Last entry (default)
project/scripts/read-context -n 3      # Last 3 entries
project/scripts/read-context -h        # Headers only (date, hash, agent, model)
project/scripts/read-context -n 5 -h   # Headers of last 5 entries
```

### Rotating the context log

Use `project/scripts/rotate-context` when `context.md` grows large. By default it rotates at 1MB, archiving older entries to a timestamped overflow file and keeping the two most recent entries in the active file.

```bash
project/scripts/rotate-context                        # Defaults: 1MB limit, keep 2 entries
project/scripts/rotate-context --size 524288 --keep 3 # Custom limits
```

The overflow filename is printed to stdout on success. Run this manually when needed, or after any particularly active session.

---

## Phases of a feature

### Planning

- Ask questions
- Suggest options
- This is your time to voice your opinions
- Decisions, plans, and task checklists should go in `project.md`. Tool, dependency, and stack decisions will be decided here and must be followed until discussion opens up again in the next phase.
- You should press me for a decision if I have forgotten something or something is ambiguous. After all, you will be required to live by our choices until the next planning phase. This includes, but is not limited to:
  - Stack and language (i.e. Python/Flask, Node/Express/NextJS, etc.)
  - Explicit coding standards and linters (i.e. PEP, Google Go style guide, ESLint)
  - Testing framework (i.e. go built-in, Jest)
- At some point I will decide that planning is complete for this step. We will come back to it later many times.
- At no time should you autonomously update `project.md`. Updates to `project.md` require human authorization and should otherwise be considered write-locked.
- It is inevitable that, occasionally, we may get part way through the later steps and decide that we should revisit a decision made here. You may, if absolutely necessary, suggest it. However, only a human can authorize changes to the plan.
- When a decision is superseded or retired, the original decision should remain in place in `project.md` but be marked as superseded with a reference to the replacement decision.
- In the rare but eventually inevitable event that external factors force re-planning (i.e. dependency deprecation, high severity CVE), we both have the responsibility to call it out as soon as we become aware of it so that we can make a decision on our next steps.

### Branching

All new work after a push to origin must begin on a fresh branch. If the new work refers to an itemized step in `project.md`, refer to that step in the branch name.

For now, we will attempt to work one feature at a time, obviating the need for rebasing or anything like that. If that changes, we will deal with it on a per-project implementation detail. If it becomes the norm, we may wish to codify our approach in this document.

### Coding style

Code in a way that is idiomatic for the stack/language/framework. Follow the style guides that were chosen in the planning stage. In particular, style precedence is as follows:

1. Existing project conventions
2. Chosen style guides
3. Community best practices

If you are unsure, ask. Suggest refactors where appropriate. For example, if a file gets too long or covers too many concerns.

Wherever appropriate, use pure functions. They are easier to test.

### Error handling

- Prefer explicit, typed errors over silent failure.
- Avoid catching errors unless there is a clear recovery path.
- Log with enough context to reconstruct failure after the fact.

### Source code location 

All new source code goes in ./src unless an explicit decision recorded in project.md specifies a different location. Do not create new top-level directories without human authorization.

### Dependencies

If a new dependency becomes necessary that has not previously been human-approved, follow these rules:

- New dependencies must be justified in commit messages
- Prefer standard library over third-party
- Avoid adding dependencies solely for syntactic convenience
- For the most part, pinning to a major semver version will be appropriate. Where it isn't, we will deal with it on a case-by-case basis.

### Testing

Use the testing framework decided upon during the planning phase. All tests must pass before each commit.

After each interaction which requires changes to code, consider whether there are appropriate tests that could be added or need to be changed. If so, do so and report those test changes.

There are no hard rules on coverage. We should, however, always keep testability top of mind as we build. Things like pure functions and dependency injection, even at the system level, are our friends.

### Linting

Use any tool(s) which we decided upon during the planning phase. Any errors or warnings must not be ignored. Strive to maintain a clean codebase at all times.

### Performance

Optimize for clarity first. Address performance only when there is evidence (measurements, scale assumptions, or explicit requirements) that it matters.

The coding, testing, and linting phases may be commingled as you see fit. In some cases you may wish to, for example, adopt a test-first pattern. You are free to make this decision autonomously.

Where appropriate, you are encouraged to develop script(s) for human use so that I may operate without you. Where appropriate these scripts should be surfaced in documentation and places like `package.json` or a `Makefile`.

### Rollbacks

It is inevitable that these will happen once in a while. In most cases we will catch them quickly and a simple `git revert` will handle the issue. More complex scenarios will be dealt with on a case-by-case basis and we should reflect upon them together to determine if the error could have been avoided.

### Documentation

We should strive to keep user-facing documentation like `README.md`, API docs, etc. continuously up to date with the truth in the code. They are everyone's responsibility. As mentioned before, these are artifacts just like code and do not influence our operational choices.

---

## After every interaction

Complete this checklist before marking the interaction done:

1. **Add context entry (ALWAYS — must precede or be in the same commit as the work it documents)**
   ```
   project/scripts/add-context \
     --agent "<agent>" \
     --model "<model>" \
     --output project/context.md \
     "Summary of what was done, what changed, what's next. Branch: <branch>"
   ```
   - Reference relevant steps in `project.md` where appropriate
   - Always include the current branch name
   - Think: "What would I need to know if this session crashed right now?"
   - **Never commit code without a context entry. The context entry must be staged before or in the same commit as the work.**

2. **Security scan** (if any files changed)
   - Scan changed files for secrets: API keys, passwords, usernames, IP addresses, hostnames, SSH keys — anything environment-specific
   - Replace findings with placeholders; keep originals in `secrets.env`
   - Ensure `secrets.env` is in `.gitignore`; maintain a `secrets.env.example` with placeholders
   - Report any substitutions
   - A security scan script with clear exit codes is encouraged (`0`=pass, `1`=fail, `2`=error)

3. **Report progress (ALWAYS)**
   - Include diffs of changed files
   - For large diffs (>200 lines), provide summary + representative excerpts
   - For generated files, note their existence without full content
   - List any new dependencies or configuration changes

4. **Run tests** (if code changed)
   - All tests must pass
   - Report test additions/modifications
   - Note any skipped tests with reasons

5. **Commit** (recommended after each interaction)
   - Stage changed files — **context entry must be included**
   - Write a clear commit message (see Commit Messages section)
   - Do NOT push — only humans can push

---

## Commit Messages

Use clear, descriptive commit messages that explain WHAT changed and WHY.

**Format:**
```
<Component>: <summary>

[Optional detailed explanation]
[Reference to project.md steps if applicable]
```

**Examples:**
```
Phase 1: Foundation - Project structure, config, Gmail auth

Classifier: Add retry logic with exponential backoff
- 3 attempts with 2-10s exponential backoff
- Handles transient API failures
- Per project.md Phase 4 requirements

Tests: Add corpus-based classification suite

Fixes #42 - Handle edge case in email parsing
```

**Guidelines:**
- Keep summary line under 72 characters where practical
- Use imperative mood ("Add" not "Added")
- Include context in body for non-obvious changes
- Reference `project.md` steps or issue numbers when relevant

**What does not belong in commits:**
- Generated files
- Lockfiles
- Editor ephemera (`.vscode`, `.un~`, etc.)
- Build artifacts
- Binaries or vendored code (but make a note of them in the `README.md`)

---

## Feature build/test cycle complete

You will have been reporting your progress after each interaction. When I declare the feature is complete, then and only then, can we move on to the next step which is...

## Send 'er

The phrase **send 'er** is a command which has been chosen because it is unlikely to come up in normal discussion. It is a specific command comprised of the following steps:

1. Run a security scan on the entire project. If any substitutions were required, report to me.
2. If any file changes have occurred since the last test run, run all tests. All tests must pass. If not, report to me.
3. If any code changes have occurred since the last linter run, run the linter. It must not report any errors or warnings. If it does, report to me.
4. If a build is required, run it. If any errors or warnings are reported, report them to me, along with any insight or suggestions to fix them.
5. Show summary and confirm push:
   - Display: branch name, files changed, commits to push
   - Prompt: "Ready to push to origin? (y/n)"
   - Only proceed if confirmed
6. Push to origin.
7. Open a pull request for the change.

## Notifications

The template contains a script, `/scripts/notify` which takes a single unnamed message argument. It notifies the human that something is required. If you complete a task, need approval, or anything else that requires human intervention and the user has not responded in one minute, execute the notify script with a short message to indicate your status.

---

## Troubleshooting

**If a context entry was missed:**
```
project/scripts/add-context \
  --agent "..." \
  --model "..." \
  --output project/context.md \
  "Retroactive entry for interaction at [approximate time]: [what was done]"
```
Include current state.

**If a DOA step was skipped:**
- Acknowledge it in the next context entry
- Complete the missed step if still relevant
- If a pattern of skipping emerges, discuss process improvement

**If in doubt:**
- Ask the human before proceeding
- Better to clarify than to guess wrong
- Document the clarification for future reference

---

## Quick Reference

**Starting a new feature:**
```bash
read project/project.md                          # review decisions
project/scripts/read-context -n 2               # resume context
git checkout -b feature-name                     # branch FIRST
# implement following DOA phases
project/scripts/add-context ... --output project/context.md "..."  # after each interaction
```

**Completing an interaction:**
```bash
project/scripts/add-context --agent "..." --model "..." --output project/context.md "Summary. Branch: <branch>"
# run security scan if files changed
# report progress with diffs
git add . && git commit -m "Component: Summary"  # context entry must be included
# do NOT push
```

**Finishing a feature:**
```
Human declares feature complete
Human says "send 'er"
Run full check sequence
Confirm and push
Open PR
```

---

## Overarching goals

If you haven't figured it out yet, these rules are designed to help us work together most efficiently, with the fewest surprises, and the smallest blast radius when things inevitably go wrong.

Just remember, if you think you know something I don't, tell me. If you think I know something you don't or you have a question, stop and ask.
---

## Appendix: Go Stack

This section supplements the DOA for projects using Go. It is authoritative
for any Go project and must be followed alongside the base DOA rules above.

### Planning phase additions

During planning, in addition to the standard decisions, the following must be
decided and recorded in project.md:

- Go module path (e.g. `github.com/<user>/<repo>`)
- Go version (pinned in `go.mod`)
- Whether `testify` or standard `testing` is used for assertions
- HTTP router/framework if applicable (stdlib `net/http`, chi, gin, etc.)
- Whether the project requires a multi-stage Docker build

### Coding style

Follow the [Google Go Style Guide](https://google.github.io/styleguide/go/).
In addition:

- All exported symbols (types, functions, methods, constants) require godoc
  comments. No exceptions.
- Interfaces are defined at the **consumer** side, not the producer. Keep them
  small — one or two methods is ideal.
- Use `context.Context` as the first argument of any function that may block,
  call an external service, or respect cancellation.
- No global mutable state. Inject dependencies through constructors or function
  arguments.
- Pure functions are preferred. They are easier to test.

### Error handling

- Wrap errors with `fmt.Errorf("component action: %w", err)` so callers can
  use `errors.Is` / `errors.As`.
- Never swallow errors silently. If you choose to ignore an error, add a
  comment explaining why.
- Prefer typed sentinel errors (`var ErrNotFound = errors.New(...)`) over
  string comparisons.

### Testing

- Always run `go test -race ./...`. Running without `-race` is not acceptable.
- Use table-driven tests as the default pattern.
- Test files live alongside source (`foo_test.go` next to `foo.go`).
- Integration tests that require external services should be guarded with
  `testing.Short()` or a build tag.

### Formatting and linting

- `goimports -w .` must be run before every commit. This is non-negotiable in Go.
- `golangci-lint run ./...` must produce zero errors and zero warnings.
- `go vet ./...` must be clean.

### Security

- `gosec ./...` is part of the `send 'er` gate. All high-severity findings
  must be resolved before pushing.
- API keys and secrets are loaded exclusively from environment variables.
  Never hardcode credentials.
- Sanitize all user-provided input before passing to external services
  (LLM APIs, database queries, shell commands).

### `send 'er` gate (Go)

When the human says "send 'er", execute in this order:

1. `gosec ./...` — report all findings. Block on HIGH severity.
2. `goimports -l .` — must produce no output (all files formatted).
3. `go vet ./...` — must be clean.
4. `golangci-lint run ./...` — must be clean.
5. `go test -race -count=1 ./...` — all tests must pass.
6. `go build ./...` — must succeed.
7. Add context entry via `project/scripts/add-context`.
8. Add session summary via `project/scripts/add-session-entry --type summary`.
9. Show: branch name, files changed, commits to push.
10. Prompt: "Ready to push to origin? (y/n)" — wait for confirmation.
11. Push to origin. Open a pull request.

### AI Session Logging

For any project using agentic development tools (Claude Code, Codex, etc.),
`AI_SESSION.md` is a required artifact subject to the same rules as
`context.md`:

- Managed exclusively through `project/scripts/add-session-entry`.
- Append-only. Entries are never edited after being written.
- Every prompt received from a human must be logged verbatim before work begins.
- Every meaningful unit of work must be followed by a summary entry.
- Included in every commit alongside the work it documents.

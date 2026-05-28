# Linter Setup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `golangci-lint` with a `.golangci.yml` config, a `Makefile` with standard targets, a Claude Code pre-commit hook that enforces lint + tests before every commit, and updated docs in `CLAUDE.md` and `README.md`.

**Architecture:** `golangci-lint` is installed once per developer (`~/go/bin/golangci-lint`). A `.golangci.yml` at the project root controls which linters run. A `Makefile` exposes `lint`, `build`, `test`, `vet`, and `all` targets. A Claude Code `PreToolUse` hook (`.claude/settings.json` + `.claude/hooks/pre-commit-check.sh`) intercepts every `git commit` Bash call and runs `make lint && go test ./...` first, blocking the commit if either fails.

**Tech Stack:** `golangci-lint` v1.64.8, GNU Make, Claude Code hooks (JSON + bash)

---

## File Map

| File | Action | Purpose |
|------|--------|---------|
| `.golangci.yml` | Create | Linter config — enabled linters, per-linter settings, test-file exclusions |
| `Makefile` | Create | `lint`, `build`, `test`, `vet`, `all` targets |
| `.claude/hooks/pre-commit-check.sh` | Create | Hook script: intercepts git commit, runs lint + tests |
| `.claude/settings.json` | Create | Registers the pre-commit hook with Claude Code |
| `CLAUDE.md` | Modify | Add `make` targets and golangci-lint install to Commands section |
| `README.md` | Modify | Replace raw go commands in Development section with `make` targets; add Prerequisites |

---

## Task 1: Create `.golangci.yml`

**Files:**
- Create: `.golangci.yml`

- [ ] **Step 1: Create the config file**

```yaml
# .golangci.yml
run:
  timeout: 5m

linters:
  enable:
    - misspell
    - unparam
    - nakedret
    - prealloc
    - exhaustive
    - nolintlint
    - gocritic

linters-settings:
  gocritic:
    enabled-tags:
      - diagnostic
      - style
  nakedret:
    max-func-lines: 30
  exhaustive:
    default-signifies-exhaustive: true

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
```

- [ ] **Step 2: Verify golangci-lint is on PATH**

Run:
```bash
golangci-lint --version
```

Expected: version output like `golangci-lint has version v1.64.8 ...`

If not found, add `~/go/bin` to PATH or use full path `~/go/bin/golangci-lint`. The binary is already installed at `/Users/hollandclarke/go/bin/golangci-lint`.

- [ ] **Step 3: Run the linter and observe baseline output**

Run:
```bash
golangci-lint run ./...
```

Expected: either clean output (no issues), or a list of findings. **Do not fix findings yet** — that is Task 6. Just confirm the config is valid (no `Error: ...` config parse errors).

- [ ] **Step 4: Commit**

```bash
git add .golangci.yml
git commit -m "build: add golangci-lint config"
```

---

## Task 2: Create `Makefile`

**Files:**
- Create: `Makefile`

- [ ] **Step 1: Create the Makefile**

```makefile
# Resolve golangci-lint: prefer PATH, fall back to default GOPATH install location
GOLANGCI_LINT := $(shell which golangci-lint 2>/dev/null || echo "$(HOME)/go/bin/golangci-lint")

.PHONY: all build test vet lint

## all: run vet, tests, and linter
all: vet test lint

## build: compile the binary
build:
	go build -o paceline .

## test: run all unit tests
test:
	go test ./...

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint
lint:
	$(GOLANGCI_LINT) run ./...
```

- [ ] **Step 2: Verify `make build` works**

Run:
```bash
make build
```

Expected: compiles without error, produces `./paceline` binary.

- [ ] **Step 3: Verify `make test` works**

Run:
```bash
make test
```

Expected: all tests pass (`ok` lines, no `FAIL`).

- [ ] **Step 4: Verify `make vet` works**

Run:
```bash
make vet
```

Expected: exits 0 with no output (no vet issues).

- [ ] **Step 5: Verify `make lint` resolves the binary**

Run:
```bash
make lint
```

Expected: linter runs (either clean or shows findings — no "command not found" error).

- [ ] **Step 6: Commit**

```bash
git add Makefile
git commit -m "build: add Makefile with lint, build, test, vet targets"
```

---

## Task 3: Create Claude Code pre-commit hook

**Files:**
- Create: `.claude/hooks/pre-commit-check.sh`
- Create: `.claude/settings.json`

This hook fires before every `Bash` tool call Claude makes. If the command contains `git commit`, it runs `make lint && go test ./...` and blocks the commit if either fails.

- [ ] **Step 1: Create the hook script**

```bash
mkdir -p .claude/hooks
```

Create `.claude/hooks/pre-commit-check.sh`:

```bash
#!/usr/bin/env bash
# Claude Code PreToolUse hook: enforce lint + tests before git commit.
# Reads tool input JSON from stdin. Exits 2 to block the tool call.

set -euo pipefail

# Parse the bash command from the tool input JSON
input=$(cat)
command=$(echo "$input" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(data.get('command', ''))
" 2>/dev/null || echo "")

# Only intercept git commit calls
if ! echo "$command" | grep -qE '(^|;|&&|\|\|)\s*git commit'; then
  exit 0
fi

# Navigate to the project root
project_root=$(git -C "$(dirname "$0")" rev-parse --show-toplevel 2>/dev/null || pwd)
cd "$project_root"

# Resolve golangci-lint
GOLANGCI_LINT=$(which golangci-lint 2>/dev/null || echo "$HOME/go/bin/golangci-lint")

echo "🔍 Pre-commit check: running linter..."
if ! "$GOLANGCI_LINT" run ./... ; then
  echo ""
  echo "❌ Linter failed. Fix the issues above before committing."
  exit 2
fi

echo "🧪 Pre-commit check: running tests..."
if ! go test ./... ; then
  echo ""
  echo "❌ Tests failed. Fix the failures above before committing."
  exit 2
fi

echo "✅ Lint and tests passed — proceeding with commit."
exit 0
```

- [ ] **Step 2: Make the hook executable**

Run:
```bash
chmod +x .claude/hooks/pre-commit-check.sh
```

- [ ] **Step 3: Create `.claude/settings.json`**

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/pre-commit-check.sh"
          }
        ]
      }
    ]
  }
}
```

- [ ] **Step 4: Verify the hook script parses JSON correctly**

Run:
```bash
echo '{"command": "git commit -m \"test\""}' | bash .claude/hooks/pre-commit-check.sh
```

Expected: the hook runs lint and tests, then prints `✅ Lint and tests passed` and exits 0 (assuming the codebase is clean after Task 6). If there are lint issues, it exits 2 — that is expected until Task 6 cleans them up. The key validation here is that the script runs without a syntax error.

- [ ] **Step 5: Verify the hook skips non-commit commands**

Run:
```bash
echo '{"command": "git status"}' | bash .claude/hooks/pre-commit-check.sh
echo "Exit code: $?"
```

Expected: no output, exit code 0 (hook silently passes non-commit commands through).

- [ ] **Step 6: Commit**

```bash
git add .claude/hooks/pre-commit-check.sh .claude/settings.json
git commit -m "build: add Claude Code pre-commit hook to enforce lint + tests"
```

---

## Task 4: Update `CLAUDE.md`

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Replace the Commands section**

In `CLAUDE.md`, find the `## Commands` section. Replace the existing commands block with:

````markdown
## Commands

```bash
# Install linter (one-time, adds to ~/go/bin/)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Build
make build          # go build -o paceline .
go build -o paceline .   # also works directly

# Run without building
go run . <command>

# Run all checks (vet + tests + lint)
make all

# Run tests
make test           # go test ./...
go test ./...       # also works directly

# Run tests for a single package
go test ./internal/parser/...
go test ./internal/store/...
go test ./internal/display/...

# Run a single test by name
go test ./internal/parser/... -run TestParseFIT

# Vet
make vet            # go vet ./...
go vet ./...        # also works directly

# Lint
make lint           # golangci-lint run ./...
```
````

- [ ] **Step 2: Verify the file looks correct**

Run:
```bash
head -40 CLAUDE.md
```

Expected: the Commands section shows the updated content with `make` targets and the golangci-lint install line.

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md with Makefile targets and golangci-lint install"
```

---

## Task 5: Update `README.md`

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Replace the Development section**

Find the `## Development` section in `README.md` (currently lines 278–293). Replace it with:

````markdown
## Development

### Prerequisites

- **Go 1.21+**
- **golangci-lint** (one-time install):
  ```bash
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  ```
  This installs the binary to `~/go/bin/`. Ensure that directory is on your `$PATH`, or the `Makefile` will fall back to the full path automatically.

### Common commands

```bash
# Run all checks (vet + tests + lint) — the recommended pre-commit command
make all

# Build the binary
make build

# Run tests
make test

# Lint only
make lint

# Run tests for a single package
go test ./internal/parser/...

# Run a single test by name
go test ./internal/parser/... -run TestParseFIT
```

Test data files live in `testdata/` (`sample.fit`, `sample.gpx`, `sample.tcx`).
````

- [ ] **Step 2: Verify the file looks correct**

Run:
```bash
grep -n "Development\|make\|golangci" README.md
```

Expected: shows the new `make` commands and golangci-lint install under the Development section.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: update README Development section with make targets and golangci-lint"
```

---

## Task 6: Fix any lint findings

**Files:**
- Modify: whichever source files golangci-lint reports issues in

- [ ] **Step 1: Run the linter and capture output**

Run:
```bash
make lint 2>&1 | tee /tmp/lint-output.txt
cat /tmp/lint-output.txt
```

Expected: either `0 issues.` (done — skip to Task 7), or a list of findings per file.

- [ ] **Step 2: Triage findings**

For each finding, decide:
- **Fix it** — if it catches a real issue (unused param, unchecked error, misspelling)
- **Suppress with reason** — if it's a false positive or intentional pattern, add `//nolint:lintername // reason` inline

Common patterns in this codebase:
- `errcheck` on `fmt.Fprintf` calls to stderr: safe to suppress — `//nolint:errcheck // stderr write, non-critical`
- `unparam` on cobra `RunE` signatures: the `cmd *cobra.Command` param is required by the interface — `//nolint:unparam // cobra RunE signature`
- `exhaustive` on switches that use a default case: add `default-signifies-exhaustive: true` is already set in `.golangci.yml`, so these should not fire

- [ ] **Step 3: Apply fixes or suppressions**

Edit each reported file. For a suppression, add the comment on the same line as the offending code:

```go
// Example suppression:
fmt.Fprintf(os.Stderr, "error: %v\n", err) //nolint:errcheck // stderr write, non-critical
```

For a real fix, fix the code directly.

- [ ] **Step 4: Re-run lint to confirm zero findings**

Run:
```bash
make lint
```

Expected: `0 issues.` or no output (clean exit 0).

- [ ] **Step 5: Run tests to confirm nothing is broken**

Run:
```bash
make test
```

Expected: all tests pass.

- [ ] **Step 6: Commit any fixes**

```bash
git add -A
git commit -m "fix: resolve golangci-lint findings"
```

If there were no findings, skip this commit.

---

## Task 7: Final end-to-end verification

- [ ] **Step 1: Run `make all`**

Run:
```bash
make all
```

Expected: `go vet ./...` passes, `go test ./...` passes, `golangci-lint run ./...` passes. Exit 0.

- [ ] **Step 2: Verify the pre-commit hook fires**

Run:
```bash
echo '{"command": "git commit -m \"verification\""}' | bash .claude/hooks/pre-commit-check.sh
```

Expected:
```
🔍 Pre-commit check: running linter...
🧪 Pre-commit check: running tests...
✅ Lint and tests passed — proceeding with commit.
```

Exit code 0.

- [ ] **Step 3: Verify the hook blocks a commit when lint fails**

Temporarily introduce a lint error — add an unused variable to any `.go` file:

```go
// At the top of any function, add:
_ = "lint test" // remove after verification
unusedVar := 42
```

Then run:
```bash
echo '{"command": "git commit -m \"should be blocked\""}' | bash .claude/hooks/pre-commit-check.sh
echo "Exit code: $?"
```

Expected: linter output showing the `unused variable` finding, then `❌ Linter failed.` message, exit code 2.

Remove the temporary change:
```bash
git checkout -- .
```

- [ ] **Step 4: Done**

All components are in place:
- `.golangci.yml` — linter config
- `Makefile` — `make all / build / test / vet / lint`
- `.claude/hooks/pre-commit-check.sh` + `.claude/settings.json` — pre-commit enforcement
- `CLAUDE.md` and `README.md` — updated docs

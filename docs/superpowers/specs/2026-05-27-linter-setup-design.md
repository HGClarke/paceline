# Linter Setup Design

**Date:** 2026-05-27
**Status:** Approved

---

## Overview

Add `golangci-lint` as the project linter with a `.golangci.yml` config file, a `Makefile` with standard targets, and updates to `CLAUDE.md` and `README.md` to document the new workflow.

---

## Components

| File | Action | Purpose |
|------|--------|---------|
| `.golangci.yml` | Create | Linter config — enabled linters, per-linter settings, exclusions |
| `Makefile` | Create | `lint`, `build`, `test`, `vet`, `all` targets |
| `CLAUDE.md` | Update | Add `make lint` (and other Makefile targets) to the Commands section; document golangci-lint install |
| `README.md` | Update | Update Development section with `make` targets and golangci-lint dependency |

---

## Linter Configuration

### Tool

`golangci-lint` — installed once per developer via:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

Not committed to the repo. Version pinned in `.golangci.yml` via the `run.go` minimum version field.

### Enabled Linters

**Default linters** (golangci-lint enables these automatically):

| Linter | Catches |
|--------|---------|
| `errcheck` | Unchecked error return values |
| `govet` | Suspicious constructs (same as `go vet`) |
| `gosimple` | Code simplifications |
| `staticcheck` | Broad static analysis |
| `ineffassign` | Assignments whose value is never used |
| `unused` | Unused code (functions, variables, constants) |

**Additional linters for moderate strictness:**

| Linter | Catches |
|--------|---------|
| `misspell` | Spelling mistakes in comments and string literals |
| `unparam` | Function parameters that always receive the same value |
| `nakedret` | Naked returns in functions longer than a threshold |
| `prealloc` | Slices that could be pre-allocated for performance |
| `exhaustive` | Missing cases in `switch` statements over enum types |
| `nolintlint` | Malformed or unnecessary `//nolint` directives |
| `gocritic` | Style and correctness checks (moderate subset via `enable-tags: diagnostic,style`) |

### Exclusions

- `errcheck` is suppressed in `*_test.go` files — test helper calls rarely need checked errors, and the noise is high.
- `gocritic` runs only the `diagnostic` and `style` tag subsets, not `performance` or `experimental`.

---

## Makefile Targets

```makefile
.PHONY: all build test vet lint

all: vet test lint

build:
	go build -o paceline .

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...
```

---

## Markdown Updates

### `CLAUDE.md` — Commands section

Add under the existing commands block:

```bash
# Lint (requires golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
make lint

# Run all checks (vet + test + lint)
make all
```

### `README.md` — Development section

Replace the raw `go build`/`go test`/`go vet` commands with `make` equivalents, and add a **Prerequisites** subsection documenting golangci-lint installation.

---

## Error Handling

- If `golangci-lint` is not installed, `make lint` fails with a clear `golangci-lint: command not found` message — no silent failure.
- The `Makefile` does not attempt to auto-install `golangci-lint`; installation is the developer's responsibility (documented in both markdown files).

---

## Testing

No automated tests for the linter config itself. Validation is manual:

1. `make lint` runs cleanly on the existing codebase (zero findings, or findings are reviewed and either fixed or suppressed with `//nolint` with a reason comment).
2. `make all` succeeds end-to-end.

---

## Out of Scope

- CI integration (GitHub Actions) — local only for now.
- Pre-commit hooks — not added; developer runs `make lint` manually.
- Auto-fixing — `golangci-lint` has a `--fix` flag for some linters; not wired up in the Makefile to keep the target predictable.

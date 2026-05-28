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
project_root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
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

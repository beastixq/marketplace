#!/usr/bin/env bash
set -euo pipefail

OUT="project_snapshot.md"

{
  echo "# Project Snapshot"
  echo
  echo "Generated at: $(date -Is)"
  echo

  echo "## 1. Git"
  echo
  echo '```text'
  git rev-parse --show-toplevel 2>/dev/null || true
  git branch --show-current 2>/dev/null || true
  git status --short 2>/dev/null || true
  echo '```'
  echo

  echo "## 2. Important files presence"
  echo
  echo '```text'
  for f in \
    AGENTS.md \
    CLAUDE.md \
    README.md \
    Makefile \
    justfile \
    go.mod \
    go.sum \
    .golangci.yml \
    .env.example \
    docker-compose.yml \
    deployments/docker-compose.yml \
    api/openapi.yaml \
    .github/workflows/ci.yml \
    .github/pull_request_template.md
  do
    if [ -e "$f" ]; then
      echo "[x] $f"
    else
      echo "[ ] $f"
    fi
  done
  echo '```'
  echo

  echo "## 3. Top-level tree"
  echo
  echo '```text'
  if command -v tree >/dev/null 2>&1; then
    tree -a -L 4 \
      -I '.git|vendor|node_modules|tmp|dist|build|coverage|.cache|bin|*.log' .
  else
    find . \
      -path './.git' -prune -o \
      -path './vendor' -prune -o \
      -path './node_modules' -prune -o \
      -path './tmp' -prune -o \
      -path './coverage' -prune -o \
      -type f -maxdepth 4 -print | sort
  fi
  echo '```'
  echo

  echo "## 4. Go version and modules"
  echo
  echo '```text'
  go version 2>/dev/null || true
  echo
  if [ -f go.mod ]; then
    echo "--- go.mod ---"
    sed -n '1,220p' go.mod
  fi
  echo '```'
  echo

  echo "## 5. Go packages"
  echo
  echo '```text'
  go list ./... 2>/dev/null || true
  echo '```'
  echo

  echo "## 6. Makefile targets"
  echo
  echo '```text'
  if [ -f Makefile ]; then
    grep -E '^[a-zA-Z0-9_.-]+:' Makefile | sed 's/:.*//' | sort -u
  fi
  echo '```'
  echo

  echo "## 7. Internal package files"
  echo
  echo '```text'
  find internal -type f 2>/dev/null \
    | sort \
    | sed 's#^\./##' \
    | grep -vE 'vendor|tmp|coverage' || true
  echo '```'
  echo

  echo "## 8. Tests"
  echo
  echo '```text'
  find . -name '*_test.go' \
    -not -path './vendor/*' \
    -not -path './.git/*' \
    | sort || true
  echo '```'
  echo

  echo "## 9. Migrations"
  echo
  echo '```text'
  find . -maxdepth 4 \( -path './migrations/*' -o -path './db/migrations/*' \) -type f 2>/dev/null \
    | sort || true
  echo '```'
  echo

  echo "## 10. API/contracts/docs"
  echo
  echo '```text'
  find api docs .specify -maxdepth 4 -type f 2>/dev/null \
    | sort || true
  echo '```'
  echo

  echo "## 11. Handlers/services/repositories quick map"
  echo
  echo '```text'
  find internal -type f -name '*.go' 2>/dev/null \
    | grep -E '/(handler|service|repository|cache|adapter|config|model|domain)/' \
    | sort || true
  echo '```'
  echo

    echo "## 12. Domain-critical references"
    echo
    echo '```text'

    SEARCH_DIRS=()
    for d in internal cmd migrations docs config; do
    [ -d "$d" ] && SEARCH_DIRS+=("$d")
    done

    rg -n \
    'reserved_quantity|stock_quantity|order.status|OrderStatus|payment|Payment|PayOrder|CancelOrder|ShipOrder|DeliverOrder|Checkout|idempotent|callback|expiration|FOR UPDATE|transaction|Tx|role|RBAC|jwt|auth' \
    "${SEARCH_DIRS[@]}" \
    --glob '!internal/techui/**' \
    --glob '!docs/RPZ.md' \
    --glob '!docs/tz.md' \
    | head -250 || true

    echo '```'

    echo "## 13. Agent setup references"
    echo
    echo '```text'

    find . -maxdepth 5 \( \
    -name 'AGENTS.md' -o \
    -name 'CLAUDE.md' -o \
    -name 'SKILL.md' -o \
    -name 'settings.json' -o \
    -name 'settings.local.json' -o \
    -name 'config.toml' \
    \) \
    -not -path './.git/*' \
    -not -path './caveman/*' \
    | sort

    echo '```'

    echo "## 14. Check commands"
    echo
    echo '```text'

    if [ -f Makefile ]; then
    grep -E '^[a-zA-Z0-9_.-]+:' Makefile | sed 's/:.*//' | sort -u
    fi

    echo
    echo "Existing test commands likely to work:"
    echo "go test ./..."
    echo "go test ./internal/service/..."
    echo "go test ./internal/repository/..."
    echo "go test ./internal/cache/..."

    echo '```'

} > "$OUT"

echo "Wrote $OUT"

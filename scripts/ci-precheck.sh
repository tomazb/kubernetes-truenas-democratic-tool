#!/usr/bin/env bash
# Validate paths referenced by CI, Makefile, and release workflows exist.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

failures=0

require_path() {
  local label="$1"
  local path="$2"
  if [[ ! -e "$path" ]]; then
    echo "MISSING [$label]: $path" >&2
    failures=$((failures + 1))
  fi
}

echo "== CI precheck: Go commands =="
require_path "go-cmd" "go/cmd/monitor"
require_path "go-cmd" "go/cmd/api-server"

echo "== CI precheck: Go module =="
require_path "go-mod" "go/go.mod"
require_path "go-sum" "go/go.sum"

echo "== CI precheck: Dockerfiles (CI + release) =="
for df in monitor api cli; do
  require_path "dockerfile" "deploy/docker/Dockerfile.${df}"
done

echo "== CI precheck: Python package and tests =="
require_path "python-package" "python/pyproject.toml"
require_path "python-tests" "python/tests"

echo "== CI precheck: CI workflow references =="
require_path "ci-workflow" ".github/workflows/ci.yml"
require_path "release-workflow" ".github/workflows/release.yml"

echo "== CI precheck: Makefile build outputs =="
# Mirrors make go-build (monitor + api-server only).
while IFS= read -r pkg; do
  require_path "go-build-target" "go/cmd/${pkg}"
done <<'EOF'
monitor
api-server
EOF

echo "== CI precheck: release container matrix mapping =="
# release.yml uses component names that must map to existing Dockerfiles.
require_path "release-dockerfile:monitor" "deploy/docker/Dockerfile.monitor"
require_path "release-dockerfile:api-server" "deploy/docker/Dockerfile.api"
require_path "release-dockerfile:cli" "deploy/docker/Dockerfile.cli"

if [[ "$failures" -gt 0 ]]; then
  echo "ci-precheck failed: $failures missing path(s)" >&2
  exit 1
fi

echo "ci-precheck passed."

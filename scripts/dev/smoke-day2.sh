#!/usr/bin/env zsh
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${0}")" && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/../.." && pwd)

cd "${REPO_ROOT}"

# I run the focused smoke test first so I can iterate faster on Day 2 behavior.
mkdir -p .cache/go-build .cache/go-mod
GOCACHE="${REPO_ROOT}/.cache/go-build" \
GOMODCACHE="${REPO_ROOT}/.cache/go-mod" \
go test ./tests/smoke -run TestDay2MDSAndOSTFlow -v

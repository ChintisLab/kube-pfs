#!/usr/bin/env zsh
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${0}")" && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/../.." && pwd)

# Tooling sanity is intentionally strict because every later step depends on it.
cd "${REPO_ROOT}"
make check-prereqs

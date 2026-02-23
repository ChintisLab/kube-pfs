#!/usr/bin/env zsh
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${0}")" && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/../.." && pwd)

# I keep prerequisite logic in Makefile so this wrapper stays tiny.
cd "${REPO_ROOT}"
make check-prereqs

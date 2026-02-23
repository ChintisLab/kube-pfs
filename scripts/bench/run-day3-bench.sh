#!/usr/bin/env zsh
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${0}")" && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/../.." && pwd)
OUT_DIR="${REPO_ROOT}/artifacts/bench"
STAMP=$(date +"%Y%m%d-%H%M%S")
RUN_DIR="${OUT_DIR}/${STAMP}"

mkdir -p "${RUN_DIR}"

if ! command -v fio >/dev/null 2>&1; then
  echo "fio is missing. Install with: brew install fio"
  exit 1
fi

# I keep benchmark files in a timestamped folder so I can compare runs over time.
fio --output-format=json "${REPO_ROOT}/benchmarks/fio/seq-readwrite.fio" > "${RUN_DIR}/fio-seq.json"
fio --output-format=json "${REPO_ROOT}/benchmarks/fio/rand-readwrite.fio" > "${RUN_DIR}/fio-rand.json"

if command -v mdtest >/dev/null 2>&1; then
  mdtest -n 500 -d "${RUN_DIR}/mdtest-dir" > "${RUN_DIR}/mdtest.txt" 2>&1 || true
else
  echo "mdtest not found; skipping metadata benchmark" > "${RUN_DIR}/mdtest.txt"
fi

cat > "${RUN_DIR}/summary.json" <<JSON
{
  "timestamp": "${STAMP}",
  "fio_seq": "fio-seq.json",
  "fio_rand": "fio-rand.json",
  "mdtest": "mdtest.txt"
}
JSON

echo "Day 3 benchmark artifacts written to ${RUN_DIR}"

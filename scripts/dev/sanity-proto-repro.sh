#!/usr/bin/env zsh
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${0}")" && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/../.." && pwd)

cd "${REPO_ROOT}"

hash_generated() {
  if [[ ! -d pkg/proto/gen ]]; then
    echo ""
    return
  fi

  local files
  files=$(find pkg/proto/gen -type f -name '*.go' | sort)
  if [[ -z "${files}" ]]; then
    echo ""
    return
  fi

  while IFS= read -r file; do
    shasum -a 256 "${file}"
  done <<< "${files}" | shasum -a 256 | awk '{print $1}'
}

# I expect two back-to-back generations to produce identical files.
make proto-gen
first_hash=$(hash_generated)

if [[ -z "${first_hash}" ]]; then
  echo "No generated files found under pkg/proto/gen after proto generation."
  exit 1
fi

make proto-gen
second_hash=$(hash_generated)

if [[ "${first_hash}" != "${second_hash}" ]]; then
  echo "Proto generation is not reproducible."
  echo "First hash:  ${first_hash}"
  echo "Second hash: ${second_hash}"
  exit 1
fi

echo "Proto generation reproducibility check passed (${first_hash})."

#!/usr/bin/env zsh
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${0}")" && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/../.." && pwd)

cd "${REPO_ROOT}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is missing."
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "Docker daemon is not reachable. Start Docker Desktop and re-run."
  exit 1
fi

components=(mds ost csi-controller csi-node)
for component in "${components[@]}"; do
  tag="kube-pfs/${component}-smoke:local"
  echo "Building ${tag}"
  docker build \
    --build-arg COMPONENT="${component}" \
    -f hack/docker/smoke/Dockerfile \
    -t "${tag}" \
    .
done

echo "Container smoke builds completed for all components."

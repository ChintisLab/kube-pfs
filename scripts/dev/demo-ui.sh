#!/usr/bin/env zsh
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${0}")" && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/../.." && pwd)

cd "${REPO_ROOT}"

cleanup() {
  if [[ -n "${PROM_PID:-}" ]]; then
    kill "${PROM_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${GRAFANA_PID:-}" ]]; then
    kill "${GRAFANA_PID}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

# I keep both forwards local so my demo UI can query Prometheus and open Grafana directly.
kubectl -n kube-pfs-observability port-forward svc/kube-pfs-prometheus 9090:9090 >/tmp/kube-pfs-prom-port-forward.log 2>&1 &
PROM_PID=$!

kubectl -n kube-pfs-observability port-forward svc/kube-pfs-grafana 3000:3000 >/tmp/kube-pfs-grafana-port-forward.log 2>&1 &
GRAFANA_PID=$!

sleep 2

echo "Demo UI: http://127.0.0.1:8088"
echo "Prometheus: http://127.0.0.1:9090"
echo "Grafana: http://127.0.0.1:3000"
echo "If metric cards are flat, I run: make seed-metrics N=20"

go run ./cmd/demo-ui --listen :8088 --repo-root . --prom-url http://127.0.0.1:9090

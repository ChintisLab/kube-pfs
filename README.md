# kube-pfs: Kubernetes CSI Driver for a Simulated Parallel File System

## Description
`kube-pfs` is a local-first learning project that simulates a parallel file system architecture behind a Kubernetes CSI-style workflow.
It separates metadata operations (MDS) from block data operations (OST), then exposes observability, benchmarking, and fault-injection tooling around that path.
The goal is to practice and demonstrate CSI-oriented systems thinking, not just build a basic storage demo.
It helps explain how storage control/data planes, runtime metrics, and failure behavior come together in one workflow.

## Features
- Metadata service (`Create`, `Lookup`, `Stat`, `ListDir`, `Unlink`) with BoltDB persistence.
- Object storage service (`WriteBlock`, `ReadBlock`, `DeleteBlock`, `GetHealth`) using flat-file blocks.
- CSI controller/node service implementation for local MVP behavior.
- Prometheus metrics across MDS, OST, CSI, and fault-injection events.
- Grafana-compatible dashboard JSON for Day 3 observability panels.
- Benchmark runner with `fio` profiles and optional `mdtest`.
- Fault injector CLI (`delete-pod`, `netem-delay`, `corrupt-block`) with timeline logging.
- Demo UI to present cluster status, metrics snapshots, benchmark summaries, and fault history.

## Tech Stack
- Go
- gRPC + Protocol Buffers
- BoltDB (`go.etcd.io/bbolt`)
- Docker + kind + kubectl
- Prometheus + Grafana
- `fio` (and optional `mdtest`)
- GitHub Actions

## Installation
1. Clone the repository:
```bash
git clone <your-repo-url>
cd kube-pfs
```

2. Install local development tools (macOS helper):
```bash
./scripts/dev/bootstrap-macos.sh
```

3. Verify prerequisites:
```bash
make doctor
```

4. Run baseline sanity checks:
```bash
make sanity
```

## Usage
1. Build project binaries:
```bash
make build-day3
```

2. Start local Kubernetes and observability:
```bash
make cluster-up
make ns-init
make observability-up
```

3. Start project services (run each in a separate terminal):
```bash
GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go run ./cmd/mds --listen :50051 --metrics-listen :9101
```
```bash
GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go run ./cmd/ost --ost-id ost-0 --listen :50061 --metrics-listen :9102 --data-dir ./data/ost0
```
```bash
GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go run ./cmd/csi-controller --endpoint unix:///tmp/kube-pfs-csi-controller.sock --metrics-listen :9103
```
```bash
GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go run ./cmd/csi-node --endpoint unix:///tmp/kube-pfs-csi-node.sock --metrics-listen :9104
```

4. Start the demo UI:
```bash
make demo-ui
```

5. Generate live data for dashboards:
```bash
make seed-metrics N=50
make benchmark
```

6. Open:
- `http://127.0.0.1:8088` (demo UI)
- `http://127.0.0.1:9090` (Prometheus)
- `http://127.0.0.1:3000` (Grafana)

7. Useful validation commands:
```bash
make prom-targets
make prom-metrics
make smoke
```

## Live Demo Validation Checklist
I use this checklist before every local demo so I can prove the system is live end-to-end.

1. Verify prerequisite and build health:
```bash
make doctor
make build-day3
make smoke
```
Expected: no failures, and `TestDay2MDSAndOSTFlow` passes.

2. Verify observability stack is running:
```bash
make observability-up
kubectl -n kube-pfs-observability get pods
```
Expected: Prometheus and Grafana pods are `Running`.

3. Verify Prometheus can scrape local services:
```bash
make prom-targets
```
Expected: `host.docker.internal:9101`, `9102`, `9103`, and `9104` show `health: "up"`.

4. Generate live metrics and benchmark data:
```bash
make seed-metrics N=50
make benchmark
```
Expected: seed command succeeds, and benchmark artifacts are written to `artifacts/bench/<timestamp>/`.

5. Generate fault events:
```bash
POD=$(kubectl -n kube-pfs-observability get pod -l app=kube-pfs-grafana -o jsonpath='{.items[0].metadata.name}')
make fault-delete POD=$POD NAMESPACE=kube-pfs-observability
mkdir -p artifacts/faults
dd if=/dev/zero of=artifacts/faults/demo.blk bs=1024 count=1
make fault-corrupt PATH_TO_BLOCK=artifacts/faults/demo.blk CORRUPT_BYTES=128
```
Expected: both fault commands return `succeeded`.

6. Verify API endpoints consumed by the UI:
```bash
curl -sS http://127.0.0.1:8088/api/status | jq .
curl -sS http://127.0.0.1:8088/api/benchmarks/latest | jq .
curl -sS http://127.0.0.1:8088/api/faults | jq .
curl -sS 'http://127.0.0.1:8088/api/prometheus?expr=sum(pfs_iops_total)' | jq .
```
Expected: non-empty JSON responses with live values/events.

7. Final visual check in browser:
- Open `http://127.0.0.1:8088`
- Hard refresh once (`Cmd+Shift+R`)
- Confirm cluster status, benchmark numbers, fault timeline, and metrics cards all populate.

## Project Structure
```text
kube-pfs/
├── cmd/
│   ├── csi-controller/
│   ├── csi-node/
│   ├── demo-ui/
│   ├── fault-injector/
│   ├── mds/
│   ├── ost/
│   └── seed-metrics/
├── pkg/
│   ├── csi/
│   ├── mds/
│   ├── metrics/
│   ├── ost/
│   └── proto/gen/
├── proto/
├── deploy/
│   ├── k8s/
│   └── observability/
├── benchmarks/
├── scripts/
├── tests/
├── docs/
├── Makefile
└── README.md
```

## Example Output / Screenshots
Example smoke test output:
```text
=== RUN   TestDay2MDSAndOSTFlow
--- PASS: TestDay2MDSAndOSTFlow (0.03s)
PASS
ok      github.com/.../tests/smoke
```

Example benchmark output:
```text
Day 3 benchmark artifacts written to .../artifacts/bench/<timestamp>
```

Example fault timeline event (`artifacts/faults/timeline.jsonl`):
```json
{"timestamp":"2026-02-24T04:40:41Z","action":"corrupt-block","status":"ok","detail":"corrupt-block completed"}
```

For UI screenshots, add image files under `docs/screenshots/` and reference them here.

## Future Improvements
- Replace current CSI MVP behavior with full end-to-end Kubernetes CSI deployment objects for MDS/OST/CSI services.
- Add real FUSE mount integration in the node path.
- Expand failure-path tests and automate them in CI.
- Add deeper performance reports (latency percentiles and trend comparison across runs).
- Add packaged one-command demo launcher for local presentations.

## Contributing
Pull requests are welcome. For major changes, open an issue first to discuss design and scope.

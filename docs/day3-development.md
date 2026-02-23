# Day 3 Development Notes

I completed Day 3 in five tracks.

## 1) Observability

I added `/metrics` endpoints and Prometheus instrumentation to:

- MDS
- OST
- CSI controller
- CSI node
- fault injector events

Metrics include:

- `pfs_write_latency_seconds`
- `pfs_read_throughput_bytes`
- `pfs_iops_total`
- `pfs_mds_lock_contention_seconds`
- `pfs_csi_operations_total`
- `pfs_fault_injection_events_total`

## 2) Fault injection

I added `cmd/fault-injector` with three actions:

- `delete-pod`
- `netem-delay`
- `corrupt-block`

I also log each event into `artifacts/faults/timeline.jsonl`.

## 3) Benchmarking

I added benchmark profiles and a runner script:

- `benchmarks/fio/seq-readwrite.fio`
- `benchmarks/fio/rand-readwrite.fio`
- `scripts/bench/run-day3-bench.sh`

Outputs are stored under `artifacts/bench/<timestamp>/`.

## 4) CI/CD

I added two workflows:

- `.github/workflows/ci.yml`
- `.github/workflows/benchmark.yml`

## 5) Ops docs and manifests

I added:

- `deploy/observability/prometheus.yaml`
- `deploy/observability/grafana-dashboard-kube-pfs.json`
- `docs/runbooks/day3-ops-runbook.md`

## Commands I run for Day 3

```bash
make build-day3
make compile-check
make smoke
make observability-up
make benchmark
```

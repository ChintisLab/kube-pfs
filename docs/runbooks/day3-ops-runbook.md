# Day 3 Operations Runbook

## I use this runbook for local Day 3 operations

### 1) Build and quick test

```bash
make build-day3
make compile-check
make smoke
```

### 2) Start local cluster baseline

```bash
make cluster-up
make ns-init
```

### 3) Deploy observability stack

```bash
make observability-up
```

I import `deploy/observability/grafana-dashboard-kube-pfs.json` in Grafana after first login.

### 4) Run benchmark suite

```bash
make benchmark
```

### 5) Run fault injection actions

Delete a pod:

```bash
make fault-delete POD=ost-0 NAMESPACE=kube-pfs-system
```

Inject delay:

```bash
make fault-netem POD=ost-0 NAMESPACE=kube-pfs-system DELAY=250ms
```

Corrupt a block:

```bash
make fault-corrupt PATH_TO_BLOCK=./data/ost/sample.blk CORRUPT_BYTES=512
```

### 6) Inspect artifacts

- Benchmark artifacts: `artifacts/bench/*`
- Fault timeline: `artifacts/faults/timeline.jsonl`

I inspect timeline safely with:

```bash
make fault-timeline
```

I follow new events live with:

```bash
make fault-timeline-follow
```

I clear old events before a fresh test with:

```bash
make fault-timeline-clear
```

### 7) Prime and verify Prometheus metrics

I generate synthetic traffic for demo metrics with:

```bash
make seed-metrics N=20
```

I inspect scrape health with:

```bash
make prom-targets
```

I inspect key `pfs_*` metric presence with:

```bash
make prom-metrics
```

### 8) Tear down

```bash
make cluster-down
```

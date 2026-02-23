# kube-pfs

I am building `kube-pfs`, a Kubernetes CSI driver project for a simulated parallel file system.  
I am developing it in small, reviewable steps so I can test and explain each part before moving forward.

## Why I am building this

I want this project to demonstrate hands-on understanding of:

- CSI controller and node plugin behavior in Kubernetes
- metadata/data plane separation (MDS + OST model)
- FUSE and mount workflow integration
- observability, benchmarking, and fault behavior

## Current status

I have completed Day 1 groundwork, Day 2 MVP development, and Day 3 implementation.

### Day 1 groundwork

- local prerequisites and environment checks
- setup automation scripts
- kind cluster bootstrap manifests
- repository scaffolding
- protobuf API contracts for MDS and OST services
- sanity checks for tooling, proto reproducibility, and smoke image builds

### Day 2 MVP

- Metadata service (`Create`, `Lookup`, `Stat`, `ListDir`, `Unlink`) with BoltDB-backed persistence
- Object storage service (`WriteBlock`, `ReadBlock`, `DeleteBlock`, `GetHealth`) with flat-file block storage
- CSI controller and node service binaries for local iteration
- Day 2 smoke test for create/write/read/unlink behavior

### Day 3 implementation

- Prometheus metrics endpoints on MDS, OST, CSI controller, and CSI node
- Metrics emitted for write latency, read throughput, IOPS, CSI ops, and MDS lock contention
- Fault injector CLI for pod kill, netem delay, and block corruption with timeline logging
- Benchmark runner (`fio` + optional `mdtest`) with timestamped artifacts
- GitHub Actions for CI checks and scheduled/manual benchmark artifacts
- Operations runbook and observability manifests

## Target architecture

```text
Pods
  -> CSI Node Plugin (mount path)
  -> MetadataService (MDS)
  -> ObjectStorageService (OST shards)
  -> Prometheus/Grafana + benchmark/fault tooling
```

## Repository layout

- `cmd/`: binary entrypoints (`mds`, `ost`, `csi-controller`, `csi-node`)
- `proto/`: protobuf contracts
- `pkg/proto/gen/`: generated Go stubs
- `deploy/k8s/`: local cluster and namespace manifests
- `scripts/dev/`: setup and sanity scripts
- `hack/docker/smoke/`: smoke container assets
- `docs/`: prerequisites, contracts, conventions, sanity notes

## Local quickstart (my workflow)

1. Install dependencies:

```bash
./scripts/dev/bootstrap-macos.sh
```

2. Validate prerequisites:

```bash
make doctor
```

3. Bring up local cluster:

```bash
make cluster-up
make ns-init
```

4. Validate contracts and baseline checks:

```bash
make proto-gen
make compile-check
make sanity
```

5. Tear down when done:

```bash
make cluster-down
```

## Commands I use most

- `make print-required-versions`
- `make check-prereqs`
- `make doctor`
- `make install-tools-macos`
- `make cluster-up`
- `make cluster-down`
- `make ns-init`
- `make proto-gen`
- `make compile-check`
- `make sanity`
- `make build-day2`
- `make build-day3`
- `make smoke`
- `make observability-up`
- `make benchmark`
- `make fault-delete POD=kube-pfs-grafana-abcde NAMESPACE=kube-pfs-observability`
- `make fault-netem POD=ost-0 NAMESPACE=kube-pfs-system DELAY=250ms`
- `make fault-corrupt PATH_TO_BLOCK=artifacts/faults/demo.blk CORRUPT_BYTES=512`
- `make fault-timeline`
- `make fault-timeline-follow`
- `make fault-timeline-clear`

## Troubleshooting notes

### Go missing

```bash
brew install go
go version
make check-prereqs
```

### Protobuf plugins missing

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
export PATH="$(go env GOPATH)/bin:$PATH"
```

### Docker daemon not reachable

```bash
docker info
```

If that fails, I start Docker Desktop and rerun sanity checks.

### kind missing

```bash
brew install kind
kind version
```

## How I am working

- I move step-by-step with approval gates.
- I run local checks before starting new implementation work.
- I keep comments practical and human, focused on intent and tradeoffs.
- I push incremental commits with sensible scope.

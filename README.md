# kube-pfs

A Kubernetes CSI driver project for a simulated parallel file system, built in an approval-gated learning workflow.

## Project Goal

`kube-pfs` models the core control/data-plane ideas used by parallel file systems:

- CSI controller and node plugins for Kubernetes volume lifecycle
- Metadata service (MDS) for namespace and stripe decisions
- Object storage targets (OSTs) for striped block reads/writes
- Observability and benchmarking from day one
- Fault injection to study degraded behavior under stress

This repository is structured so you can learn each layer before approving the next implementation step.

## Current Status

Day 1 groundwork is in place:

- prerequisites and local setup automation
- kind cluster bootstrap targets
- repository skeleton
- protobuf service contracts for MDS and OST
- sanity checks for tooling, proto reproducibility, and image build pipeline

Day 2 implementation (runtime services + CSI flow) starts only after your explicit approval.

## Architecture (target)

```text
Kubernetes Pods
  -> CSI Node Plugin (FUSE mount path)
  -> MetadataService (MDS)
  -> ObjectStorageService (OST shards)
  -> Prometheus/Grafana + benchmark/fault tooling
```

## Repository Layout

- `cmd/`: service entrypoints (`mds`, `ost`, `csi-controller`, `csi-node`)
- `proto/`: protobuf contracts
- `pkg/proto/gen/`: generated Go stubs
- `deploy/k8s/`: local cluster and namespace manifests
- `scripts/dev/`: setup and sanity scripts
- `hack/docker/smoke/`: baseline container smoke build assets
- `docs/`: prerequisites, contracts, conventions, sanity notes

## Local Development Quick Start

1. Install dependencies:

```bash
./scripts/dev/bootstrap-macos.sh
```

2. Validate your machine:

```bash
make doctor
```

3. Start local Kubernetes:

```bash
make cluster-up
make ns-init
```

4. Validate generated contracts and smoke pipeline:

```bash
make proto-gen
make compile-check
make sanity
```

5. Tear down cluster when done:

```bash
make cluster-down
```

## Core Commands

- `make print-required-versions`: print expected tool versions
- `make check-prereqs`: strict local environment verification
- `make doctor`: wrapper for prerequisite checks
- `make install-tools-macos`: install local tools using bootstrap script
- `make cluster-up`: create `kind` cluster `kube-pfs`
- `make cluster-down`: delete `kind` cluster `kube-pfs`
- `make ns-init`: apply `kube-pfs-system` and `kube-pfs-test` namespaces
- `make proto-gen`: generate Go protobuf and gRPC stubs
- `make compile-check`: run `go test ./...`
- `make sanity`: run all Day 1 sanity checks

## Troubleshooting

### `go is not installed or not in PATH`

- Install Go: `brew install go`
- Verify: `go version`
- Re-run: `make check-prereqs`

### `protoc-gen-go is missing`

- Install plugins:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
```

- Ensure Go tools are in PATH:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

### `Cannot connect to the Docker daemon`

- Start Docker Desktop
- Check daemon status: `docker info`
- Re-run sanity checks after daemon is healthy

### `kind: command not found`

- Install: `brew install kind`
- Verify: `kind version`

## Learning Workflow (Approval Gated)

Every step follows this sequence:

1. concept brief (what and why)
2. local runnable checkpoint
3. explicit pause for your approval

No next step starts until you approve the current one.

## Commenting and Commit Style

- Comments must be human-written, short, and useful.
- Comments should explain intent or tradeoffs, not obvious syntax.
- Commit messages should read like natural engineering notes, not generated text.

See `docs/coding-conventions.md` for full standards.

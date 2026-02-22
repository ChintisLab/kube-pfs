# kube-pfs Prerequisites (Day 1)

This project is intentionally strict about local setup so later storage behavior is easier to reason about.

## System Requirements

- OS: macOS arm64 for local development
- CPU/RAM: 8 vCPU / 16 GB RAM recommended
- Disk: 20 GB free (images, kind cluster state, benchmark artifacts)

## Required Tool Versions

These are the versions currently supported by the repo checks.

- Go: `>= 1.23.x`
- kind: `>= 0.26.x`
- kubectl: `1.32.x`
- Helm: `>= 3.16.x`
- fio: `3.x`
- protoc: `25.x`
- Docker CLI + Docker daemon: must both be available

## Installation (macOS)

Use the project bootstrap script:

```bash
./scripts/dev/bootstrap-macos.sh
```

Manual path (if needed):

```bash
brew install go kind helm fio protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
```

If `protoc-gen-go` is still not found, add Go tool binaries to your shell profile:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

## Validation commands

```bash
make print-required-versions
make check-prereqs
```

## What to observe

- Missing tools fail with explicit install guidance.
- Version mismatches fail with required version boundaries.
- Docker daemon issues fail early (before cluster or image commands).

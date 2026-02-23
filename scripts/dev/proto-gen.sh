#!/usr/bin/env zsh
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${0}")" && pwd)
REPO_ROOT=$(cd -- "${SCRIPT_DIR}/../.." && pwd)

cd "${REPO_ROOT}"

if ! command -v protoc >/dev/null 2>&1; then
  echo "protoc is not installed."
  exit 1
fi

if ! command -v protoc-gen-go >/dev/null 2>&1; then
  echo "protoc-gen-go is not installed. Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.0"
  exit 1
fi

if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
  echo "protoc-gen-go-grpc is not installed. Run: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1"
  exit 1
fi

mkdir -p pkg/proto/gen

# I generate message types and service stubs together so outputs stay in sync.
protoc \
  --proto_path=proto \
  --go_out=pkg/proto/gen --go_opt=paths=source_relative \
  --go-grpc_out=pkg/proto/gen --go-grpc_opt=paths=source_relative \
  proto/metadata.proto proto/object_storage.proto

echo "protobuf generation completed: pkg/proto/gen"

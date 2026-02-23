#!/usr/bin/env zsh
set -euo pipefail

# I only support macOS in this helper because my local workflow is built around Homebrew.
if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "This bootstrap script is intended for macOS."
  exit 1
fi

if ! command -v brew >/dev/null 2>&1; then
  echo "Homebrew is required. Install from https://brew.sh and re-run."
  exit 1
fi

echo "Installing kube-pfs local dependencies via Homebrew..."
brew install go kind helm fio protobuf

# I install these plugin binaries because I need them for protobuf generation.
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

# I add GOPATH/bin when this shell cannot find the Go tool binaries.
GOBIN_PATH="$(go env GOPATH)/bin"
if [[ ":${PATH}:" != *":${GOBIN_PATH}:"* ]]; then
  echo "Add this to your shell profile (~/.zshrc):"
  echo "  export PATH=\"${GOBIN_PATH}:\$PATH\""
fi

echo "Verifying versions after install..."
go version
kind version
helm version --short
fio --version
protoc --version

echo "Bootstrap complete. Run: make doctor"

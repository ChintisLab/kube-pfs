SHELL := /bin/zsh

GO_REQUIRED_MAJOR := 1
GO_REQUIRED_MINOR_MIN := 23
KIND_REQUIRED_MAJOR := 0
KIND_REQUIRED_MINOR_MIN := 26
KUBECTL_REQUIRED_MAJOR := 1
KUBECTL_REQUIRED_MINOR := 32
HELM_REQUIRED_MAJOR := 3
HELM_REQUIRED_MINOR_MIN := 16
FIO_REQUIRED_MAJOR := 3
PROTOC_REQUIRED_MAJOR := 25
GO_CACHE_DIR := $(PWD)/.cache/go-build
GO_MOD_CACHE_DIR := $(PWD)/.cache/go-mod
GO_ENV := GOCACHE=$(GO_CACHE_DIR) GOMODCACHE=$(GO_MOD_CACHE_DIR)

.PHONY: print-required-versions check-prereqs doctor install-tools-macos
.PHONY: cluster-up cluster-down ns-init
.PHONY: proto-check-tools proto-gen compile-check
.PHONY: sanity-tooling sanity-proto-repro sanity-container-smoke sanity
.PHONY: build-day2 smoke

print-required-versions:
	@echo "Required versions (minimum unless noted):"
	@echo "  go:       >= $(GO_REQUIRED_MAJOR).$(GO_REQUIRED_MINOR_MIN).x"
	@echo "  kind:     >= $(KIND_REQUIRED_MAJOR).$(KIND_REQUIRED_MINOR_MIN).x"
	@echo "  kubectl:  $(KUBECTL_REQUIRED_MAJOR).$(KUBECTL_REQUIRED_MINOR).x"
	@echo "  helm:     >= $(HELM_REQUIRED_MAJOR).$(HELM_REQUIRED_MINOR_MIN).x"
	@echo "  fio:      $(FIO_REQUIRED_MAJOR).x"
	@echo "  protoc:   $(PROTOC_REQUIRED_MAJOR).x"

check-prereqs:
	@set -euo pipefail; \
	fail() { echo "[FAIL] $$1"; exit 1; }; \
	pass() { echo "[PASS] $$1"; }; \
	require_cmd() { \
		local cmd="$$1"; \
		local hint="$$2"; \
		if ! command -v "$$cmd" >/dev/null 2>&1; then \
			fail "$$cmd is not installed or not in PATH. $$hint"; \
		fi; \
	}; \
	require_min_mm() { \
		local name="$$1"; \
		local actual_major="$$2"; \
		local actual_minor="$$3"; \
		local req_major="$$4"; \
		local req_minor="$$5"; \
		if (( actual_major < req_major )) || (( actual_major == req_major && actual_minor < req_minor )); then \
			fail "$$name $$actual_major.$$actual_minor.x found, required >= $$req_major.$$req_minor.x"; \
		fi; \
		pass "$$name $$actual_major.$$actual_minor.x"; \
	}; \
	require_exact_mm() { \
		local name="$$1"; \
		local actual_major="$$2"; \
		local actual_minor="$$3"; \
		local req_major="$$4"; \
		local req_minor="$$5"; \
		if (( actual_major != req_major || actual_minor != req_minor )); then \
			fail "$$name $$actual_major.$$actual_minor.x found, required $$req_major.$$req_minor.x"; \
		fi; \
		pass "$$name $$actual_major.$$actual_minor.x"; \
	}; \
	echo "Checking kube-pfs prerequisites..."; \
	require_cmd docker "Install Docker Desktop and ensure it is in PATH."; \
	docker_version="$$(docker --version | sed -nE 's/^Docker version ([0-9.]+).*/\1/p')"; \
	[[ -n "$$docker_version" ]] || fail "could not parse Docker version"; \
	pass "docker $$docker_version"; \
	docker info >/dev/null 2>&1 || fail "Docker daemon is not reachable. Start Docker Desktop and re-run."; \
	pass "docker daemon reachable"; \
	require_cmd kubectl "Install with: brew install kubectl"; \
	kubectl_mm="$$(kubectl version --client 2>/dev/null | sed -nE 's/^Client Version: v([0-9]+)\.([0-9]+).*/\1 \2/p')"; \
	[[ -n "$$kubectl_mm" ]] || fail "could not parse kubectl version"; \
	kubectl_major="$${kubectl_mm%% *}"; \
	kubectl_minor="$${kubectl_mm##* }"; \
	require_exact_mm kubectl "$$kubectl_major" "$$kubectl_minor" "$(KUBECTL_REQUIRED_MAJOR)" "$(KUBECTL_REQUIRED_MINOR)"; \
	require_cmd protoc "Install with: brew install protobuf"; \
	protoc_major="$$(protoc --version | sed -nE 's/^libprotoc ([0-9]+).*/\1/p')"; \
	[[ -n "$$protoc_major" ]] || fail "could not parse protoc version"; \
	(( protoc_major == $(PROTOC_REQUIRED_MAJOR) )) || fail "protoc major $$protoc_major found, required $(PROTOC_REQUIRED_MAJOR).x"; \
	pass "protoc $$(protoc --version | sed -E 's/^libprotoc //')"; \
	require_cmd go "Install with: brew install go"; \
	go_mm="$$(go version | sed -nE 's/^go version go([0-9]+)\.([0-9]+).*/\1 \2/p')"; \
	[[ -n "$$go_mm" ]] || fail "could not parse go version"; \
	go_major="$${go_mm%% *}"; \
	go_minor="$${go_mm##* }"; \
	require_min_mm go "$$go_major" "$$go_minor" "$(GO_REQUIRED_MAJOR)" "$(GO_REQUIRED_MINOR_MIN)"; \
	require_cmd kind "Install with: brew install kind"; \
	kind_mm="$$(kind version | sed -nE 's/.*v([0-9]+)\.([0-9]+).*/\1 \2/p')"; \
	[[ -n "$$kind_mm" ]] || fail "could not parse kind version"; \
	kind_major="$${kind_mm%% *}"; \
	kind_minor="$${kind_mm##* }"; \
	require_min_mm kind "$$kind_major" "$$kind_minor" "$(KIND_REQUIRED_MAJOR)" "$(KIND_REQUIRED_MINOR_MIN)"; \
	require_cmd helm "Install with: brew install helm"; \
	helm_mm="$$(helm version --short 2>/dev/null | sed -nE 's/^v([0-9]+)\.([0-9]+).*/\1 \2/p')"; \
	[[ -n "$$helm_mm" ]] || fail "could not parse helm version"; \
	helm_major="$${helm_mm%% *}"; \
	helm_minor="$${helm_mm##* }"; \
	(( helm_major == $(HELM_REQUIRED_MAJOR) )) || fail "helm major $$helm_major found, required $(HELM_REQUIRED_MAJOR).x"; \
	(( helm_minor >= $(HELM_REQUIRED_MINOR_MIN) )) || fail "helm minor $$helm_minor found, required >= $(HELM_REQUIRED_MINOR_MIN)"; \
	pass "helm $$helm_major.$$helm_minor.x"; \
	require_cmd fio "Install with: brew install fio"; \
	fio_major="$$(fio --version | sed -nE 's/^fio-([0-9]+).*/\1/p')"; \
	[[ -n "$$fio_major" ]] || fail "could not parse fio version"; \
	(( fio_major == $(FIO_REQUIRED_MAJOR) )) || fail "fio major $$fio_major found, required $(FIO_REQUIRED_MAJOR).x"; \
	pass "fio $$(fio --version | sed -E 's/^fio-//')"; \
	echo "All prerequisite checks passed."

install-tools-macos:
	@./scripts/dev/bootstrap-macos.sh

doctor:
	@./scripts/dev/check-prereqs.sh

cluster-up:
	@set -euo pipefail; \
	command -v kind >/dev/null 2>&1 || { echo "kind is missing. Install with: brew install kind"; exit 1; }; \
	command -v kubectl >/dev/null 2>&1 || { echo "kubectl is missing. Install with: brew install kubectl"; exit 1; }; \
	docker info >/dev/null 2>&1 || { echo "Docker daemon is not reachable. Start Docker Desktop."; exit 1; }; \
	if kind get clusters | grep -qx kube-pfs; then \
		echo "kind cluster 'kube-pfs' already exists."; \
	else \
		kind create cluster --config deploy/k8s/kind-config.yaml; \
	fi; \
	kubectl cluster-info >/dev/null; \
	echo "kind cluster 'kube-pfs' is ready."

cluster-down:
	@set -euo pipefail; \
	if ! command -v kind >/dev/null 2>&1; then \
		echo "kind is not installed. Nothing to delete."; \
		exit 0; \
	fi; \
	if kind get clusters | grep -qx kube-pfs; then \
		kind delete cluster --name kube-pfs; \
		echo "kind cluster 'kube-pfs' deleted."; \
	else \
		echo "kind cluster 'kube-pfs' does not exist."; \
	fi

ns-init:
	@set -euo pipefail; \
	command -v kubectl >/dev/null 2>&1 || { echo "kubectl is missing. Install with: brew install kubectl"; exit 1; }; \
	kubectl apply -f deploy/k8s/namespaces.yaml; \
	echo "kube-pfs namespaces are applied."

proto-check-tools:
	@set -euo pipefail; \
	mkdir -p "$(GO_CACHE_DIR)" "$(GO_MOD_CACHE_DIR)"; \
	command -v protoc >/dev/null 2>&1 || { echo "protoc is missing. Install with: brew install protobuf"; exit 1; }; \
	command -v go >/dev/null 2>&1 || { echo "go is missing. Install with: brew install go"; exit 1; }; \
	command -v protoc-gen-go >/dev/null 2>&1 || { echo "protoc-gen-go is missing. Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.0"; exit 1; }; \
	command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "protoc-gen-go-grpc is missing. Run: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1"; exit 1; }; \
	echo "protobuf tooling is available."

proto-gen: proto-check-tools
	@./scripts/dev/proto-gen.sh

compile-check:
	@set -euo pipefail; \
	mkdir -p "$(GO_CACHE_DIR)" "$(GO_MOD_CACHE_DIR)"; \
	command -v go >/dev/null 2>&1 || { echo "go is missing. Install with: brew install go"; exit 1; }; \
	$(GO_ENV) go test ./...

build-day2:
	@set -euo pipefail; \
	mkdir -p "$(GO_CACHE_DIR)" "$(GO_MOD_CACHE_DIR)"; \
	$(GO_ENV) go build ./cmd/mds ./cmd/ost ./cmd/csi-controller ./cmd/csi-node

smoke:
	@set -euo pipefail; \
	./scripts/dev/smoke-day2.sh

sanity-tooling:
	@./scripts/dev/sanity-tooling.sh

sanity-proto-repro:
	@./scripts/dev/sanity-proto-repro.sh

sanity-container-smoke:
	@./scripts/dev/sanity-container-smoke.sh

sanity:
	@set -euo pipefail; \
	echo "Running Day 1 sanity checks..."; \
	$(MAKE) sanity-tooling; \
	$(MAKE) sanity-proto-repro; \
	$(MAKE) sanity-container-smoke; \
	echo "All Day 1 sanity checks passed."

# Space Sim Makefile
#
# Binaries:
#   space-sim-direct  — Raylib client + in-process server (no network transport)
#   space-sim-grpc    — Raylib client talking to embedded gRPC server via ConnectRPC
#                       Long-term target: split into separate client and server binaries.
#
# Proto generation requires buf (https://buf.build/docs/installation).
# Run `make proto` after installing buf and adding buf.yaml / buf.gen.yaml.

GO      := go
BIN_DIR := bin

DIRECT_BIN := $(BIN_DIR)/space-sim-direct
DIRECT_CMD := ./cmd/space-sim-direct

GRPC_BIN := $(BIN_DIR)/space-sim-grpc
GRPC_CMD := ./cmd/space-sim-grpc

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-20s %s\n", $$1, $$2}'

# ── Build ─────────────────────────────────────────────────────────────────────

.PHONY: build
build: build-direct build-grpc ## Build both binaries

.PHONY: build-direct
build-direct: ## Build space-sim-direct (in-process, no gRPC)
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(DIRECT_BIN) $(DIRECT_CMD)

.PHONY: build-grpc
build-grpc: ## Build space-sim-grpc (Raylib + ConnectRPC)
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(GRPC_BIN) $(GRPC_CMD)

# ── Run ───────────────────────────────────────────────────────────────────────

.PHONY: run
run: run-direct ## Alias for run-direct

.PHONY: run-direct
run-direct: build-direct ## Run the direct (in-process) binary
	./$(DIRECT_BIN)

.PHONY: run-grpc
run-grpc: build-grpc ## Run the gRPC-coupled binary
	./$(GRPC_BIN)

# ── Proto ─────────────────────────────────────────────────────────────────────

.PHONY: proto
proto: ## Regenerate Go (and future TS) code from api/proto via buf
	buf generate

# ── Test ──────────────────────────────────────────────────────────────────────

.PHONY: test
test: ## Run all unit tests with race detector
	$(GO) test -race ./...

.PHONY: test-direct
test-direct: ## Run tests for direct/server packages only
	$(GO) test -race ./internal/sim/... ./internal/server/... ./internal/persist/... ./internal/protocol/...

# ── Maintenance ───────────────────────────────────────────────────────────────

.PHONY: tidy
tidy: ## Tidy module dependencies
	$(GO) mod tidy

.PHONY: vet
vet: ## Run go vet across all packages
	$(GO) vet ./...

.PHONY: clean
clean: ## Remove build outputs and ephemeral logs
	rm -rf $(BIN_DIR)
	rm -f performance_debug.log

.PHONY: json-check
json-check: ## Validate the default system JSON and build the app
	bash scripts/test_json_system.sh
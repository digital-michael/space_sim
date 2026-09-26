# Space Sim Makefile
#
# Binaries:
#   space-sim        — Raylib renderer. Standalone (no flags) or --server host:port remote-renderer.
#   space-sim-server — Headless physics server; streams snapshots via gRPC on :9090.
#   space-sim-admin  — Interactive control REPL for a running space-sim-server.
#
# Proto generation requires buf (https://buf.build/docs/installation).
# Run `make proto` after installing buf and adding buf.yaml / buf.gen.yaml.

GO      := go
BIN_DIR := bin

SIM_BIN    := $(BIN_DIR)/space-sim
SIM_CMD    := ./cmd/space-sim

SERVER_BIN := $(BIN_DIR)/space-sim-server
SERVER_CMD := ./cmd/space-sim-server

ADMIN_BIN  := $(BIN_DIR)/space-sim-admin
ADMIN_CMD  := ./cmd/space-sim-admin

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-20s %s\n", $$1, $$2}'

# ── Build ─────────────────────────────────────────────────────────────────────

.PHONY: build
build: build-sim build-server build-admin ## Build all binaries

.PHONY: build-sim
build-sim: ## Build space-sim (standalone or --server remote-renderer)
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(SIM_BIN) $(SIM_CMD)

.PHONY: build-server
build-server: ## Build space-sim-server (headless physics server, no Raylib)
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(SERVER_BIN) $(SERVER_CMD)

.PHONY: build-admin
build-admin: ## Build space-sim-admin (control REPL)
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(ADMIN_BIN) $(ADMIN_CMD)

# ── Run ───────────────────────────────────────────────────────────────────────

.PHONY: run
run: build-sim ## Run space-sim in standalone mode
	./$(SIM_BIN)

.PHONY: run-server
run-server: build-server ## Run the headless physics server (set ADDR= and SYSTEM= to override)
	./$(SERVER_BIN) --addr $${ADDR:-:9090} --system-config $${SYSTEM:-data/systems/solar_system}

.PHONY: run-admin
run-admin: build-admin ## Run the admin REPL (set ADDR= to override server address)
	./$(ADMIN_BIN) --addr $${ADDR:-http://localhost:9090}

.PHONY: run-remote
run-remote: build-sim ## Run space-sim connected to a server (set SERVER= to override)
	./$(SIM_BIN) --server $${SERVER:-localhost:9090}

DEMO_SCRIPT := scripts/solar-tour.txt

.PHONY: demo
demo: build-admin ## Run the solar system tour demo (requires space-sim-server running; set ADDR= to override)
	@echo "# Running solar tour — make sure space-sim-server is running first"
	./$(ADMIN_BIN) --addr $${ADDR:-http://localhost:9090} --script $(DEMO_SCRIPT)

# ── Proto ─────────────────────────────────────────────────────────────────────

.PHONY: proto
proto: ## Regenerate Go (and future TS) code from api/proto via buf
	buf generate

# ── Test ──────────────────────────────────────────────────────────────────────

.PHONY: test
test: ## Run all unit tests with race detector
	$(GO) test -race ./...

.PHONY: test-direct
test-direct: ## Run tests for sim/server packages only
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

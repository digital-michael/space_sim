# Space Sim Makefile

GO := go
GOBUILD := $(GO) build
GOTEST := $(GO) test
GOMOD := $(GO) mod

BIN_DIR := bin
APP_BIN := $(BIN_DIR)/space-sim
APP_CMD := ./cmd/space-sim
APP_SRC := $(shell find cmd/space-sim internal/space -name '*.go')

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-16s %s\n", $$1, $$2}'

.PHONY: build
build: $(APP_BIN) ## Build the application binary

$(APP_BIN): $(APP_SRC)
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(APP_BIN) $(APP_CMD)

.PHONY: run
run: build ## Build and run the application
	./$(APP_BIN)

.PHONY: test
test: ## Run unit tests
	$(GOTEST) ./...

.PHONY: tidy
tidy: ## Tidy module dependencies
	$(GOMOD) tidy

.PHONY: clean
clean: ## Remove local build outputs
	rm -rf $(BIN_DIR)
	rm -f space-sim performance_debug.log

.PHONY: json-check
json-check: ## Validate the default system JSON and build the app
	bash scripts/test_json_system.sh
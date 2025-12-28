# Claude Code Go SDK - Makefile
# Alternative to Taskfile.yml for developers who prefer Make

# Variables
PROJECT_NAME := Claude Code Go SDK
BIN_DIR := ./bin
COVERAGE_DIR := ./coverage
GO_VERSION := 1.20

# Colors for output
BLUE := \033[34m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
RESET := \033[0m

# Default target
.DEFAULT_GOAL := help

# Phony targets (not files)
.PHONY: all build build-lib examples build-examples build-basic build-advanced build-testing
.PHONY: build-budget build-plugins build-subagents
.PHONY: build-demo build-demo-streaming build-demo-basic build-dangerous-example
.PHONY: build-demo-budget build-demo-plugins build-demo-subagents
.PHONY: build-demo-sessions build-demo-mcp build-demo-retry build-demo-permissions
.PHONY: test test-lib test-dangerous test-integration test-integration-real test-local coverage
.PHONY: demo demo-streaming demo-basic demo-budget demo-plugins demo-subagents run-dangerous check-go check-claude
.PHONY: demo-sessions demo-mcp demo-retry demo-permissions
.PHONY: clean help banner

##@ Build Targets

all: banner build test ## Build and test the SDK
	@echo "$(GREEN)✅ Build and test completed$(RESET)"

build: build-lib build-examples ## Build the SDK and all examples

build-lib: ## Build the core library
	@echo "$(BLUE)🔨 Building core library...$(RESET)"
	@go build ./pkg/claude
	@echo "$(GREEN)✅ Core library built successfully$(RESET)"

examples: ## Build all example programs (alias)
	@make build-examples

build-examples: build-demo-streaming build-demo-basic build-dangerous-example ## Build all example programs
	@echo "$(BLUE)🔨 Building examples...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/basic-example ./examples/basic || echo "$(RED)❌ Basic example build failed$(RESET)"
	@go build -o $(BIN_DIR)/advanced-example ./examples/advanced || echo "$(RED)❌ Advanced example build failed$(RESET)"
	@go build -o $(BIN_DIR)/testing-example ./examples/testing || echo "$(RED)❌ Testing example build failed$(RESET)"
	@go build -o $(BIN_DIR)/budget-example ./examples/budget || echo "$(RED)❌ Budget example build failed$(RESET)"
	@go build -o $(BIN_DIR)/plugins-example ./examples/plugins || echo "$(RED)❌ Plugins example build failed$(RESET)"
	@go build -o $(BIN_DIR)/subagents-example ./examples/subagents || echo "$(RED)❌ Subagents example build failed$(RESET)"
	@echo "$(GREEN)✅ Example builds completed$(RESET)"

build-demo: build-demo-streaming ## Build the interactive demo (streaming)

build-demo-streaming: ## Build the streaming demo
	@echo "$(BLUE)🔨 Building streaming demo...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@cd examples/demo/streaming && go mod tidy && go build -o ../../../$(BIN_DIR)/demo ./cmd/demo
	@echo "$(GREEN)✅ Streaming demo built successfully$(RESET)"

build-demo-basic: ## Build the basic demo
	@echo "$(BLUE)🔨 Building basic demo...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@cd examples/demo/basic && go mod tidy && go build -o ../../../$(BIN_DIR)/demo-basic ./cmd/demo
	@echo "$(GREEN)✅ Basic demo built successfully$(RESET)"

build-demo-budget: ## Build the budget tracking demo
	@echo "$(BLUE)🔨 Building budget demo...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@cd examples/demo/budget && go mod tidy && go build -o ../../../$(BIN_DIR)/demo-budget ./cmd/demo
	@echo "$(GREEN)✅ Budget demo built successfully$(RESET)"

build-demo-plugins: ## Build the plugins demo
	@echo "$(BLUE)🔨 Building plugins demo...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@cd examples/demo/plugins && go mod tidy && go build -o ../../../$(BIN_DIR)/demo-plugins ./cmd/demo
	@echo "$(GREEN)✅ Plugins demo built successfully$(RESET)"

build-demo-subagents: ## Build the subagents demo
	@echo "$(BLUE)🔨 Building subagents demo...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@cd examples/demo/subagents && go mod tidy && go build -o ../../../$(BIN_DIR)/demo-subagents ./cmd/demo
	@echo "$(GREEN)✅ Subagents demo built successfully$(RESET)"

build-demo-sessions: ## Build the sessions demo
	@echo "$(BLUE)🔨 Building sessions demo...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@cd examples/demo/sessions && go mod tidy && go build -o ../../../$(BIN_DIR)/demo-sessions ./cmd/demo
	@echo "$(GREEN)✅ Sessions demo built successfully$(RESET)"

build-demo-mcp: ## Build the MCP demo
	@echo "$(BLUE)🔨 Building MCP demo...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@cd examples/demo/mcp && go mod tidy && go build -o ../../../$(BIN_DIR)/demo-mcp ./cmd/demo
	@echo "$(GREEN)✅ MCP demo built successfully$(RESET)"

build-demo-retry: ## Build the retry demo
	@echo "$(BLUE)🔨 Building retry demo...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@cd examples/demo/retry && go mod tidy && go build -o ../../../$(BIN_DIR)/demo-retry ./cmd/demo
	@echo "$(GREEN)✅ Retry demo built successfully$(RESET)"

build-demo-permissions: ## Build the permissions demo
	@echo "$(BLUE)🔨 Building permissions demo...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@cd examples/demo/permissions && go mod tidy && go build -o ../../../$(BIN_DIR)/demo-permissions ./cmd/demo
	@echo "$(GREEN)✅ Permissions demo built successfully$(RESET)"

build-dangerous-example: ## Build dangerous usage example
	@echo "$(BLUE)🔨 Building dangerous example...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@cd examples/dangerous_usage && go mod tidy && go build -o ../../$(BIN_DIR)/dangerous-example .
	@echo "$(GREEN)✅ Dangerous example built successfully$(RESET)"

# Individual example build targets for development
build-basic: ## Build basic example only
	@echo "$(BLUE)🔨 Building basic example...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/basic-example ./examples/basic
	@echo "$(GREEN)✅ Basic example built: $(BIN_DIR)/basic-example$(RESET)"

build-advanced: ## Build advanced example only
	@echo "$(BLUE)🔨 Building advanced example...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/advanced-example ./examples/advanced
	@echo "$(GREEN)✅ Advanced example built: $(BIN_DIR)/advanced-example$(RESET)"

build-testing: ## Build testing example only
	@echo "$(BLUE)🔨 Building testing example...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/testing-example ./examples/testing
	@echo "$(GREEN)✅ Testing example built: $(BIN_DIR)/testing-example$(RESET)"

build-budget: ## Build budget tracking example only
	@echo "$(BLUE)🔨 Building budget example...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/budget-example ./examples/budget
	@echo "$(GREEN)✅ Budget example built: $(BIN_DIR)/budget-example$(RESET)"

build-plugins: ## Build plugins example only
	@echo "$(BLUE)🔨 Building plugins example...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/plugins-example ./examples/plugins
	@echo "$(GREEN)✅ Plugins example built: $(BIN_DIR)/plugins-example$(RESET)"

build-subagents: ## Build subagents example only
	@echo "$(BLUE)🔨 Building subagents example...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/subagents-example ./examples/subagents
	@echo "$(GREEN)✅ Subagents example built: $(BIN_DIR)/subagents-example$(RESET)"

##@ Test Targets

test: ## Run core tests (dashboard mode)
	@echo "$(BLUE)🧪 Core Test Suite$(RESET)"
	@echo "$(BLUE)==================$(RESET)"
	@echo ""
	@make -s test-lib
	@echo ""

test-lib: ## Test the core library (quiet mode)
	@echo "$(BLUE)🧪 Testing core library...$(RESET)"
	@if go test ./pkg/claude > /tmp/test-core.log 2>&1; then \
		echo "$(GREEN)✅ Core library tests: PASSED$(RESET)"; \
	else \
		echo "$(RED)❌ Core library tests: FAILED$(RESET)"; \
		echo "$(YELLOW)📋 Run 'make test-lib-verbose' for details$(RESET)"; \
		exit 1; \
	fi

test-lib-verbose: ## Test the core library (verbose mode)
	@echo "$(BLUE)🧪 Testing core library (verbose)...$(RESET)"
	@go test -v ./pkg/claude

test-dangerous: ## Test dangerous package (quiet mode)
	@echo "$(YELLOW)🚨 Testing dangerous package...$(RESET)"
	@if go test ./pkg/claude/dangerous > /tmp/test-dangerous.log 2>&1; then \
		echo "$(GREEN)✅ Dangerous package tests: PASSED$(RESET)"; \
	else \
		echo "$(RED)❌ Dangerous package tests: FAILED$(RESET)"; \
		echo "$(YELLOW)📋 Run 'make test-dangerous-verbose' for details$(RESET)"; \
		exit 1; \
	fi

test-dangerous-verbose: ## Test dangerous package (verbose mode)
	@echo "$(YELLOW)🚨 Testing dangerous package (verbose)...$(RESET)"
	@go test -v ./pkg/claude/dangerous

test-integration: ## Run integration tests with mock server (quiet mode)
	@echo "$(BLUE)🔗 Running integration tests (mock server)...$(RESET)"
	@if go test ./test/integration > /tmp/test-integration.log 2>&1; then \
		echo "$(GREEN)✅ Integration tests: PASSED$(RESET)"; \
	else \
		echo "$(RED)❌ Integration tests: FAILED$(RESET)"; \
		echo "$(YELLOW)📋 Run 'make test-integration-verbose' for details$(RESET)"; \
		exit 1; \
	fi

test-integration-verbose: ## Run integration tests with mock server (verbose mode)
	@echo "$(BLUE)🔗 Running integration tests (mock server, verbose)...$(RESET)"
	@go test -v ./test/integration

test-integration-real: ## Run integration tests with real Claude CLI (quiet mode)
	@echo "$(BLUE)🔗 Running integration tests (real Claude CLI)...$(RESET)"
	@if CLAUDE_INTEGRATION_TEST=real go test ./test/integration > /tmp/test-integration-real.log 2>&1; then \
		echo "$(GREEN)✅ Real integration tests: PASSED$(RESET)"; \
	else \
		echo "$(RED)❌ Real integration tests: FAILED$(RESET)"; \
		echo "$(YELLOW)📋 Run 'make test-integration-real-verbose' for details$(RESET)"; \
		exit 1; \
	fi

test-integration-real-verbose: ## Run integration tests with real Claude CLI (verbose mode)
	@echo "$(BLUE)🔗 Running integration tests (real Claude CLI, verbose)...$(RESET)"
	@CLAUDE_INTEGRATION_TEST=real go test -v ./test/integration

test-local: ## Run all local tests (dashboard mode)
	@echo "$(BLUE)🧪 Running full test suite...$(RESET)"
	@echo "$(BLUE)=============================$(RESET)"
	@echo ""
	@make -s test-lib
	@make -s test-dangerous
	@make -s test-integration
	@echo ""
	@echo "$(GREEN)✅ All tests completed successfully$(RESET)"

test-all: test-local coverage ## Run all tests and generate coverage (dashboard mode)
	@echo ""
	@echo "$(GREEN)🎉 Complete test suite finished!$(RESET)"

test-all-verbose: test-lib-verbose test-dangerous-verbose test-integration-verbose coverage-verbose ## Run all tests with verbose output

coverage: ## Generate test coverage report (quiet mode)
	@echo "$(BLUE)📊 Generating coverage report...$(RESET)"
	@mkdir -p $(COVERAGE_DIR)
	@if go test -coverprofile=$(COVERAGE_DIR)/coverage.out ./pkg/... > /tmp/coverage.log 2>&1; then \
		echo "$(GREEN)✅ Coverage generation: COMPLETED$(RESET)"; \
		coverage_pct=$$(go tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1 | awk '{print $$3}'); \
		echo "$(BLUE)📈 Total coverage: $$coverage_pct$(RESET)"; \
		go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html; \
		echo "$(BLUE)📄 HTML report: $(COVERAGE_DIR)/coverage.html$(RESET)"; \
	else \
		echo "$(RED)❌ Coverage generation: FAILED$(RESET)"; \
		exit 1; \
	fi

coverage-verbose: ## Generate test coverage report (verbose mode)
	@echo "$(BLUE)📊 Generating coverage report (verbose)...$(RESET)"
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_DIR)/coverage.out ./pkg/...
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "$(GREEN)✅ Coverage generation completed$(RESET)"
	@echo "$(BLUE)📄 View HTML report at $(COVERAGE_DIR)/coverage.html$(RESET)"

##@ Demo and Examples

demo: demo-streaming ## Run the interactive Claude Code Go SDK demo (streaming)

demo-streaming: build-demo-streaming check-go check-claude ## Run the streaming demo
	@echo "$(BLUE)🚀 Claude Code Go SDK Demo (Streaming)$(RESET)"
	@echo "$(BLUE)=====================================$(RESET)"
	@echo ""
	@echo "$(BLUE)🎯 Starting streaming demo with real-time tool display...$(RESET)"
	@echo "$(YELLOW)   Type your responses and press Enter$(RESET)"
	@echo "$(YELLOW)   Type 'exit', 'quit', 'bye', or press Enter on empty line to exit$(RESET)"
	@echo ""
	@$(BIN_DIR)/demo

demo-basic: build-demo-basic check-go check-claude ## Run the basic demo
	@echo "$(BLUE)🚀 Claude Code Go SDK Demo (Basic)$(RESET)"
	@echo "$(BLUE)=================================$(RESET)"
	@echo ""
	@echo "$(BLUE)🎯 Starting basic demo with simple JSON output...$(RESET)"
	@echo "$(YELLOW)   Type your responses and press Enter$(RESET)"
	@echo "$(YELLOW)   Type 'exit', 'quit', 'bye', or press Enter on empty line to exit$(RESET)"
	@echo ""
	@$(BIN_DIR)/demo-basic

demo-budget: build-demo-budget check-go check-claude ## Run the budget tracking demo
	@echo "$(BLUE)🚀 Claude Code Go SDK Demo (Budget Tracking)$(RESET)"
	@echo "$(BLUE)============================================$(RESET)"
	@echo "This demo shows real-time budget tracking with:"
	@echo "  - Spending limits and warnings"
	@echo "  - Per-session cost tracking"
	@echo "  - Budget exceeded protection"
	@echo ""
	@$(BIN_DIR)/demo-budget

demo-plugins: build-demo-plugins check-go check-claude ## Run the plugins demo
	@echo "$(BLUE)🚀 Claude Code Go SDK Demo (Plugin System)$(RESET)"
	@echo "$(BLUE)==========================================$(RESET)"
	@echo "This demo shows the plugin system with:"
	@echo "  - Logging, metrics, and audit plugins"
	@echo "  - Tool filtering for security"
	@echo "  - Real-time plugin callbacks"
	@echo ""
	@$(BIN_DIR)/demo-plugins

demo-subagents: build-demo-subagents check-go check-claude ## Run the subagents demo
	@echo "$(BLUE)🚀 Claude Code Go SDK Demo (Subagent Orchestration)$(RESET)"
	@echo "$(BLUE)===================================================$(RESET)"
	@echo "This demo shows the subagent system with:"
	@echo "  - Specialized agents (security, code-review, testing)"
	@echo "  - Agent switching with @agent syntax"
	@echo "  - Session persistence and resumption"
	@echo ""
	@$(BIN_DIR)/demo-subagents

demo-sessions: build-demo-sessions check-go check-claude ## Run the sessions demo
	@echo "$(BLUE)🚀 Claude Code Go SDK Demo (Session Management)$(RESET)"
	@echo "$(BLUE)================================================$(RESET)"
	@echo "This demo shows session management with:"
	@echo "  - Custom session IDs"
	@echo "  - Session forking and resumption"
	@echo "  - Ephemeral sessions"
	@echo ""
	@$(BIN_DIR)/demo-sessions

demo-mcp: build-demo-mcp check-go check-claude ## Run the MCP demo
	@echo "$(BLUE)🚀 Claude Code Go SDK Demo (MCP Integration)$(RESET)"
	@echo "$(BLUE)=============================================$(RESET)"
	@echo "This demo shows MCP integration with:"
	@echo "  - MCP server configuration"
	@echo "  - Strict mode for isolated environments"
	@echo "  - Tool allowlisting"
	@echo ""
	@$(BIN_DIR)/demo-mcp

demo-retry: build-demo-retry check-go check-claude ## Run the retry demo
	@echo "$(BLUE)🚀 Claude Code Go SDK Demo (Retry & Error Handling)$(RESET)"
	@echo "$(BLUE)===================================================$(RESET)"
	@echo "This demo shows retry and error handling with:"
	@echo "  - Configurable retry policies"
	@echo "  - Exponential backoff with jitter"
	@echo "  - Error classification"
	@echo ""
	@$(BIN_DIR)/demo-retry

demo-permissions: build-demo-permissions check-go check-claude ## Run the permissions demo
	@echo "$(BLUE)🚀 Claude Code Go SDK Demo (Permission Control)$(RESET)"
	@echo "$(BLUE)================================================$(RESET)"
	@echo "This demo shows permission control with:"
	@echo "  - Permission modes (default, acceptEdits, bypass)"
	@echo "  - Tool allowlisting and blocklisting"
	@echo "  - Security presets"
	@echo ""
	@$(BIN_DIR)/demo-permissions

run-dangerous: build-dangerous-example check-dangerous ## Run dangerous features example (development only)
	@echo "$(YELLOW)🚨 Running Dangerous Features Example$(RESET)"
	@echo "$(YELLOW)=====================================$(RESET)"
	@echo "$(GREEN)✔️  Security requirements met$(RESET)"
	@echo ""
	@$(BIN_DIR)/dangerous-example

##@ Utility Targets

clean: ## Clean build artifacts and test cache
	@echo "$(BLUE)🧹 Cleaning build artifacts...$(RESET)"
	@rm -rf $(BIN_DIR) $(COVERAGE_DIR)
	@go clean -testcache
	@rm -rf it_works/
	@echo "$(GREEN)✅ Clean completed$(RESET)"

banner: ## Display project banner
	@echo "$(BLUE)╔══════════════════════════════════════════════════════════════╗$(RESET)"
	@echo "$(BLUE)║                    $(PROJECT_NAME)                     ║$(RESET)"
	@echo "$(BLUE)║                      Build Pipeline                          ║$(RESET)"
	@echo "$(BLUE)╚══════════════════════════════════════════════════════════════╝$(RESET)"
	@echo ""

##@ Validation Targets

check-go: ## Check Go version requirements
	@if ! command -v go &> /dev/null; then \
		echo "$(RED)❌ Error: Go is not installed or not in PATH$(RESET)"; \
		exit 1; \
	fi
	@go_version=$$(go version | awk '{print $$3}' | sed 's/go//'); \
	major_version=$$(echo $$go_version | cut -d. -f1); \
	minor_version=$$(echo $$go_version | cut -d. -f2); \
	if [ $$major_version -lt 1 ] || [ $$major_version -eq 1 -a $$minor_version -lt 20 ]; then \
		echo "$(RED)❌ Error: Go ≥1.20 is required (found: $$go_version)$(RESET)"; \
		exit 1; \
	fi
	@echo "$(GREEN)✔️  Go version: $$(go version | awk '{print $$3}' | sed 's/go//')$(RESET)"

check-claude: ## Check Claude CLI availability
	@if ! claude_path=$$(command -v claude 2>/dev/null); then \
		echo "$(RED)❌ Error: claude CLI not found in PATH$(RESET)"; \
		echo "$(YELLOW)   Please install from: https://docs.anthropic.com/en/docs/claude-code/getting-started$(RESET)"; \
		exit 1; \
	fi
	@echo "$(GREEN)✔️  Found claude CLI: $$(command -v claude)$(RESET)"

check-dangerous: ## Check dangerous operation requirements
	@if [ "$(CLAUDE_ENABLE_DANGEROUS)" != "i-accept-all-risks" ]; then \
		echo "$(RED)❌ Error: CLAUDE_ENABLE_DANGEROUS must be set to 'i-accept-all-risks'$(RESET)"; \
		echo "$(YELLOW)   export CLAUDE_ENABLE_DANGEROUS=\"i-accept-all-risks\"$(RESET)"; \
		exit 1; \
	fi
	@if [ "$(NODE_ENV)" = "production" ] || [ "$(GO_ENV)" = "production" ] || [ "$(ENVIRONMENT)" = "production" ]; then \
		echo "$(RED)❌ Error: Cannot run dangerous example in production environment$(RESET)"; \
		exit 1; \
	fi

##@ Information

help: ## Display this help message
	@echo "$(BLUE)$(PROJECT_NAME) - Available Commands$(RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make $(BLUE)<target>$(RESET)\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(BLUE)%-20s$(RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(RESET)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(YELLOW)Quick Start:$(RESET)"
	@echo "  $(BLUE)make examples$(RESET)                Build all example programs"
	@echo "  $(BLUE)make demo$(RESET)                     Run interactive demo"
	@echo "  $(BLUE)make test-local$(RESET)               Run all tests (clean dashboard)"
	@echo "  $(BLUE)make test-all$(RESET)                 Run tests + coverage (complete dashboard)"
	@echo ""
	@echo "$(YELLOW)Dashboard vs Verbose:$(RESET)"
	@echo "  $(BLUE)make test-local$(RESET)               Clean dashboard output"
	@echo "  $(BLUE)make test-all-verbose$(RESET)         Detailed test output"
	@echo ""
	@echo "$(YELLOW)Examples:$(RESET)"
	@echo "  $(BLUE)make build$(RESET)                    Build the SDK and examples"
	@echo "  $(BLUE)CLAUDE_ENABLE_DANGEROUS=\"i-accept-all-risks\" make run-dangerous$(RESET)"
	@echo ""
	@echo "$(YELLOW)Alternative:$(RESET) You can also use $(BLUE)task <command>$(RESET) (see Taskfile.yml)"

version: ## Show version information
	@echo "$(BLUE)$(PROJECT_NAME)$(RESET)"
	@echo "Go version: $$(go version)"
	@echo "Make version: $$(make --version | head -n1)"
	@if command -v task &> /dev/null; then \
		echo "Task version: $$(task --version)"; \
	else \
		echo "Task: not installed"; \
	fi
	@if command -v claude &> /dev/null; then \
		echo "Claude CLI: available"; \
	else \
		echo "Claude CLI: not found"; \
	fi

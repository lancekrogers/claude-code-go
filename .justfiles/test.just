# Test Commands
# Run `just t` to see all test commands

set dotenv-load := true

PROJECT_DIR := justfile_directory()

[private]
@default:
    @echo "Test Commands"
    @echo "============="
    @just --list --justfile {{source_file()}}

# Run all tests
all: lib dangerous

# Run core library tests
lib:
    @echo "Testing core library..."
    go test -v {{PROJECT_DIR}}/pkg/claude
    @echo "Core library tests complete"

# Run dangerous package tests
dangerous:
    @echo "Testing dangerous package..."
    go test -v {{PROJECT_DIR}}/pkg/claude/dangerous
    @echo "Dangerous package tests complete"

# Run integration tests with mock server
integration:
    @echo "Running integration tests..."
    INTEGRATION_TESTS=1 USE_MOCK_SERVER=1 go test -tags=integration {{PROJECT_DIR}}/test/integration -v
    @echo "Integration tests complete"

# Run integration tests with real Claude CLI
integration-real:
    @echo "Running real integration tests..."
    @echo "This requires Claude CLI to be installed and configured"
    INTEGRATION_TESTS=1 go test -tags=integration {{PROJECT_DIR}}/test/integration -v
    @echo "Real integration tests complete"

# Run tests with race detection
race:
    @echo "Running tests with race detection..."
    go test -race {{PROJECT_DIR}}/pkg/claude/...
    @echo "Race detection tests complete"

# Run benchmark tests
bench:
    @echo "Running benchmarks..."
    go test -bench=. {{PROJECT_DIR}}/pkg/claude/...
    @echo "Benchmarks complete"

# Run tests with memory profiling
memory:
    @echo "Running memory profiling..."
    go test -memprofile=mem.out {{PROJECT_DIR}}/pkg/claude/...
    @echo "Memory profile saved to mem.out"

# Run quick tests (no verbose)
quick:
    @echo "Running quick tests..."
    go test {{PROJECT_DIR}}/pkg/claude/...
    @echo "Quick tests complete"

# Run tests for a specific package
[no-exit-message]
pkg package:
    @echo "Testing package: {{package}}"
    go test -v {{PROJECT_DIR}}/{{package}}

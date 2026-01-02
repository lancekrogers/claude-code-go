# Utility Commands
# Run `just u` to see all utility commands

set dotenv-load := true

PROJECT_DIR := justfile_directory()
BIN_DIR := PROJECT_DIR / "bin"
COVERAGE_DIR := PROJECT_DIR / "coverage"

[private]
@default:
    @echo "Utility Commands"
    @echo "================"
    @just --list --justfile {{source_file()}}

# Clean build artifacts
clean:
    @echo "Cleaning build artifacts..."
    rm -rf {{BIN_DIR}}
    rm -rf {{COVERAGE_DIR}}
    go clean -cache -testcache
    @echo "Clean complete"

# Download and verify dependencies
deps:
    @echo "Downloading dependencies..."
    go mod download
    go mod verify
    @echo "Dependencies ready"

# Tidy dependencies
tidy:
    @echo "Tidying dependencies..."
    go mod tidy
    @echo "Dependencies tidied"

# Tidy all modules
tidy-all:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Tidying all modules..."
    while IFS= read -r -d '' modfile; do
        moddir=$(dirname "$modfile")
        echo "  - ${moddir}"
        (cd "$moddir" && go mod tidy)
    done < <(find "{{PROJECT_DIR}}" -name go.mod -print0)
    echo "All modules tidied"

# Run linters (fmt + vet)
lint:
    @echo "Running linters..."
    go fmt {{PROJECT_DIR}}/...
    go vet {{PROJECT_DIR}}/...
    @echo "Linting complete"

# Fix common issues (format + tidy)
fix: lint tidy
    @echo "Fixes applied"

# Start godoc server
docs:
    @echo "Starting godoc server at http://localhost:6060"
    @echo "Press Ctrl+C to stop"
    godoc -http=:6060

# Show version info
version:
    @echo "Go version:"
    go version
    @echo ""
    @echo "Module info:"
    go list -m

# Check Go installation
check-go:
    #!/usr/bin/env bash
    if ! command -v go &> /dev/null; then
        echo "Error: Go is not installed"
        exit 1
    fi
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo "Go version: $GO_VERSION"
    MAJOR=$(echo $GO_VERSION | cut -d. -f1)
    MINOR=$(echo $GO_VERSION | cut -d. -f2)
    if [[ $MAJOR -lt 1 ]] || [[ $MAJOR -eq 1 && $MINOR -lt 20 ]]; then
        echo "Error: Go >= 1.20 required"
        exit 1
    fi
    echo "Go version OK"

# Check Claude CLI installation
check-claude:
    #!/usr/bin/env bash
    if ! command -v claude &> /dev/null; then
        echo "Warning: Claude CLI not found in PATH"
        echo "Install from: https://docs.anthropic.com/en/docs/claude-code/getting-started"
        exit 1
    fi
    echo "Claude CLI found: $(which claude)"

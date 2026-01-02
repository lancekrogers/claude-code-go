# Demo Commands
# Run `just d` to see all demo commands

set dotenv-load := true

PROJECT_DIR := justfile_directory()
BIN_DIR := PROJECT_DIR / "bin"

[private]
@default:
    @echo "Demo Commands"
    @echo "============="
    @just --list --justfile {{source_file()}}

# Build and run streaming demo (default)
streaming: build-streaming
    @echo "Starting streaming demo..."
    @echo "Type 'exit' to quit"
    @echo ""
    {{BIN_DIR}}/demo

# Build and run basic demo
basic: build-basic
    @echo "Starting basic demo..."
    {{BIN_DIR}}/demo-basic

# Build and run budget tracking demo
budget: build-budget
    @echo "Starting budget tracking demo..."
    {{BIN_DIR}}/demo-budget

# Build and run plugins demo
plugins: build-plugins
    @echo "Starting plugins demo..."
    {{BIN_DIR}}/demo-plugins

# Build and run subagents demo
subagents: build-subagents
    @echo "Starting subagents demo..."
    {{BIN_DIR}}/demo-subagents

# Build and run sessions demo
sessions: build-sessions
    @echo "Starting sessions demo..."
    {{BIN_DIR}}/demo-sessions

# Build and run MCP demo
mcp: build-mcp
    @echo "Starting MCP demo..."
    {{BIN_DIR}}/demo-mcp

# Build and run retry demo
retry: build-retry
    @echo "Starting retry demo..."
    {{BIN_DIR}}/demo-retry

# Build and run permissions demo
permissions: build-permissions
    @echo "Starting permissions demo..."
    {{BIN_DIR}}/demo-permissions

# Build and run dangerous features example
dangerous: _check-dangerous build-dangerous
    @echo "Starting dangerous features example..."
    {{BIN_DIR}}/dangerous-example

# Build and run enhanced features example
enhanced: build-enhanced
    @echo "Starting enhanced features example..."
    {{BIN_DIR}}/enhanced-example

# Build streaming demo
build-streaming:
    @echo "Building streaming demo..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}}/examples/demo/streaming && go mod tidy && go build -o {{BIN_DIR}}/demo ./cmd/demo
    @echo "Built: {{BIN_DIR}}/demo"

# Build basic demo
build-basic:
    @echo "Building basic demo..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}}/examples/demo/basic && go mod tidy && go build -o {{BIN_DIR}}/demo-basic ./cmd/demo
    @echo "Built: {{BIN_DIR}}/demo-basic"

# Build budget demo
build-budget:
    @echo "Building budget demo..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}}/examples/demo/budget && go mod tidy && go build -o {{BIN_DIR}}/demo-budget ./cmd/demo
    @echo "Built: {{BIN_DIR}}/demo-budget"

# Build plugins demo
build-plugins:
    @echo "Building plugins demo..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}}/examples/demo/plugins && go mod tidy && go build -o {{BIN_DIR}}/demo-plugins ./cmd/demo
    @echo "Built: {{BIN_DIR}}/demo-plugins"

# Build subagents demo
build-subagents:
    @echo "Building subagents demo..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}}/examples/demo/subagents && go mod tidy && go build -o {{BIN_DIR}}/demo-subagents ./cmd/demo
    @echo "Built: {{BIN_DIR}}/demo-subagents"

# Build sessions demo
build-sessions:
    @echo "Building sessions demo..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}}/examples/demo/sessions && go mod tidy && go build -o {{BIN_DIR}}/demo-sessions ./cmd/demo
    @echo "Built: {{BIN_DIR}}/demo-sessions"

# Build MCP demo
build-mcp:
    @echo "Building MCP demo..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}}/examples/demo/mcp && go mod tidy && go build -o {{BIN_DIR}}/demo-mcp ./cmd/demo
    @echo "Built: {{BIN_DIR}}/demo-mcp"

# Build retry demo
build-retry:
    @echo "Building retry demo..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}}/examples/demo/retry && go mod tidy && go build -o {{BIN_DIR}}/demo-retry ./cmd/demo
    @echo "Built: {{BIN_DIR}}/demo-retry"

# Build permissions demo
build-permissions:
    @echo "Building permissions demo..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}}/examples/demo/permissions && go mod tidy && go build -o {{BIN_DIR}}/demo-permissions ./cmd/demo
    @echo "Built: {{BIN_DIR}}/demo-permissions"

# Build dangerous example
build-dangerous:
    @echo "Building dangerous example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}}/examples/dangerous_usage && go mod tidy && go build -o {{BIN_DIR}}/dangerous-example .
    @echo "Built: {{BIN_DIR}}/dangerous-example"

# Build enhanced example
build-enhanced:
    @echo "Building enhanced example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}}/examples/enhanced_features && go mod tidy && go build -o {{BIN_DIR}}/enhanced-example .
    @echo "Built: {{BIN_DIR}}/enhanced-example"

# Build all demos
build-all: build-streaming build-basic build-budget build-plugins build-subagents build-sessions build-mcp build-retry build-permissions build-dangerous build-enhanced
    @echo "All demos built"

# Check dangerous environment requirements
[private]
_check-dangerous:
    #!/usr/bin/env bash
    if [ "$CLAUDE_ENABLE_DANGEROUS" != "i-accept-all-risks" ]; then
        echo "Error: CLAUDE_ENABLE_DANGEROUS must be set to 'i-accept-all-risks'"
        echo "  export CLAUDE_ENABLE_DANGEROUS=\"i-accept-all-risks\""
        exit 1
    fi
    if [ "$NODE_ENV" = "production" ] || [ "$GO_ENV" = "production" ]; then
        echo "Error: Cannot run dangerous example in production"
        exit 1
    fi
    echo "Security requirements met"

# ============================================================================
# GIF Generation
# ============================================================================

# Generate GIF for a specific demo (basic, streaming, sessions, mcp, retry, permissions, budget, plugins, subagents)
gif demo-name:
    {{PROJECT_DIR}}/scripts/record-demo-gif.sh {{demo-name}}

# Generate GIFs for all demos
gif-all:
    {{PROJECT_DIR}}/scripts/record-demo-gif.sh all

# List available demos for GIF generation
gif-list:
    @echo "Available demos for GIF generation:"
    @echo "  - basic"
    @echo "  - streaming"
    @echo "  - sessions"
    @echo "  - mcp"
    @echo "  - retry"
    @echo "  - permissions"
    @echo "  - budget"
    @echo "  - plugins"
    @echo "  - subagents"
    @echo ""
    @echo "Usage: just demo gif <demo-name>"
    @echo "       just demo gif-all"

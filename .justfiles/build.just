# Build Commands
# Run `just build` to see all build commands

# invocation_directory() returns the directory from which just was called
PROJECT_DIR := invocation_directory()
BIN_DIR := PROJECT_DIR / "bin"

[private]
@default:
    @echo "Build Commands"
    @echo "=============="
    @just --list --justfile {{source_file()}}

# Build library and all examples
all: lib examples

# Build core library
lib:
    @echo "Building core library..."
    cd {{PROJECT_DIR}} && go build ./pkg/claude
    @echo "Core library built successfully"

# Build all example programs
examples: basic advanced testing budget plugins subagents sessions mcp permissions retry streaming workflow
    @echo "All examples built"

# Build basic example
basic:
    @echo "Building basic example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}} && go build -o {{BIN_DIR}}/basic-example ./examples/basic
    @echo "Built: {{BIN_DIR}}/basic-example"

# Build advanced example
advanced:
    @echo "Building advanced example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}} && go build -o {{BIN_DIR}}/advanced-example ./examples/advanced
    @echo "Built: {{BIN_DIR}}/advanced-example"

# Build testing example
testing:
    @echo "Building testing example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}} && go build -o {{BIN_DIR}}/testing-example ./examples/testing
    @echo "Built: {{BIN_DIR}}/testing-example"

# Build budget example
budget:
    @echo "Building budget example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}} && go build -o {{BIN_DIR}}/budget-example ./examples/budget
    @echo "Built: {{BIN_DIR}}/budget-example"

# Build plugins example
plugins:
    @echo "Building plugins example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}} && go build -o {{BIN_DIR}}/plugins-example ./examples/plugins
    @echo "Built: {{BIN_DIR}}/plugins-example"

# Build subagents example
subagents:
    @echo "Building subagents example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}} && go build -o {{BIN_DIR}}/subagents-example ./examples/subagents
    @echo "Built: {{BIN_DIR}}/subagents-example"

# Build sessions example
sessions:
    @echo "Building sessions example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}} && go build -o {{BIN_DIR}}/sessions-example ./examples/sessions
    @echo "Built: {{BIN_DIR}}/sessions-example"

# Build MCP configuration example
mcp:
    @echo "Building MCP example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}} && go build -o {{BIN_DIR}}/mcp-example ./examples/mcp
    @echo "Built: {{BIN_DIR}}/mcp-example"

# Build permissions example
permissions:
    @echo "Building permissions example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}} && go build -o {{BIN_DIR}}/permissions-example ./examples/permissions
    @echo "Built: {{BIN_DIR}}/permissions-example"

# Build retry example
retry:
    @echo "Building retry example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}} && go build -o {{BIN_DIR}}/retry-example ./examples/retry
    @echo "Built: {{BIN_DIR}}/retry-example"

# Build streaming example
streaming:
    @echo "Building streaming example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}} && go build -o {{BIN_DIR}}/streaming-example ./examples/streaming
    @echo "Built: {{BIN_DIR}}/streaming-example"

# Build workflow example
workflow:
    @echo "Building workflow example..."
    @mkdir -p {{BIN_DIR}}
    cd {{PROJECT_DIR}} && go build -o {{BIN_DIR}}/workflow-example ./examples/workflow
    @echo "Built: {{BIN_DIR}}/workflow-example"

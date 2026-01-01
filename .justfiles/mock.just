# Mock Server Commands
# Run `just m` to see all mock server commands

set dotenv-load := true

PROJECT_DIR := justfile_directory()
MOCK_PID_FILE := "/tmp/claude-mock-server.pid"

[private]
@default:
    @echo "Mock Server Commands"
    @echo "===================="
    @just --list --justfile {{source_file()}}

# Start mock Claude server (foreground)
start:
    @echo "Starting mock Claude server..."
    go run {{PROJECT_DIR}}/test/mockserver/main.go

# Start mock server in background
start-bg:
    #!/usr/bin/env bash
    echo "Starting mock server in background..."
    MOCK_PORT="${MOCK_SERVER_PORT:-8080}"
    go run {{PROJECT_DIR}}/test/mockserver/main.go &
    echo $! > {{MOCK_PID_FILE}}
    echo "Mock server started (PID saved to {{MOCK_PID_FILE}})"
    for _ in {1..50}; do
        if curl -sf "http://localhost:${MOCK_PORT}/health" >/dev/null; then
            exit 0
        fi
        sleep 0.1
    done
    echo "Mock server failed to start on port ${MOCK_PORT}"
    just mock stop
    exit 1

# Stop background mock server
stop:
    #!/usr/bin/env bash
    if [ -f "{{MOCK_PID_FILE}}" ]; then
        PID=$(cat "{{MOCK_PID_FILE}}")
        if kill -0 "$PID" 2>/dev/null; then
            kill "$PID"
            echo "Mock server stopped (PID: $PID)"
        else
            echo "Mock server not running"
        fi
        rm -f "{{MOCK_PID_FILE}}"
    else
        echo "No PID file found"
        # Try to find and kill any running mock server
        pkill -f "mockserver/main.go" || true
    fi

# Check mock server status
status:
    #!/usr/bin/env bash
    if [ -f "{{MOCK_PID_FILE}}" ]; then
        PID=$(cat "{{MOCK_PID_FILE}}")
        if kill -0 "$PID" 2>/dev/null; then
            echo "Mock server running (PID: $PID)"
        else
            echo "Mock server not running (stale PID file)"
        fi
    else
        echo "Mock server not running (no PID file)"
    fi

# Test mock server functionality
test:
    @echo "Testing mock server..."
    curl -s "http://localhost:${MOCK_SERVER_PORT:-8080}/health" || echo "Mock server not responding"

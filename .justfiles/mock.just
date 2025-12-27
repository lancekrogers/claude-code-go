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
    go run {{PROJECT_DIR}}/test/mock_server/main.go

# Start mock server in background
start-bg:
    @echo "Starting mock server in background..."
    go run {{PROJECT_DIR}}/test/mock_server/main.go &
    echo $! > {{MOCK_PID_FILE}}
    @echo "Mock server started (PID saved to {{MOCK_PID_FILE}})"
    sleep 2

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
        pkill -f "mock_server/main.go" || true
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
    curl -s http://localhost:8080/health || echo "Mock server not responding"

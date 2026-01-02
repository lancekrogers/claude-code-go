# Coverage Commands
# Run `just c` to see all coverage commands

set dotenv-load := true

PROJECT_DIR := justfile_directory()
COVERAGE_DIR := PROJECT_DIR / "coverage"

[private]
@default:
    @echo "Coverage Commands"
    @echo "================="
    @just --list --justfile {{source_file()}}

# Generate coverage data
run:
    @echo "Generating coverage data..."
    @mkdir -p {{COVERAGE_DIR}}
    go test -coverprofile={{COVERAGE_DIR}}/coverage.out {{PROJECT_DIR}}/pkg/claude/...
    @echo "Coverage data: {{COVERAGE_DIR}}/coverage.out"

# Generate HTML coverage report
report: run
    @echo "Generating HTML coverage report..."
    go tool cover -html={{COVERAGE_DIR}}/coverage.out -o {{COVERAGE_DIR}}/coverage.html
    @echo "Coverage report: {{COVERAGE_DIR}}/coverage.html"

# Show coverage in browser
show: report
    @echo "Opening coverage report in browser..."
    open {{COVERAGE_DIR}}/coverage.html

# Show coverage percentage
percent: run
    @echo "Coverage percentage:"
    go tool cover -func={{COVERAGE_DIR}}/coverage.out | tail -1

# Clean coverage files
clean:
    @echo "Cleaning coverage files..."
    rm -rf {{COVERAGE_DIR}}
    @echo "Coverage files cleaned"

#!/usr/bin/env bash
# Record and generate demo GIFs using asciinema + agg
# Usage: ./scripts/record-demo-gif.sh [demo-name]
# Example: ./scripts/record-demo-gif.sh basic

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="${PROJECT_DIR}/docs/gif"
DEMO_SCRIPTS_DIR="${SCRIPT_DIR}/demo-scripts"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Available demos
DEMOS=(basic streaming sessions mcp retry permissions budget plugins subagents)

print_usage() {
    echo "Usage: $0 <demo-name|all>"
    echo ""
    echo "Available demos:"
    for demo in "${DEMOS[@]}"; do
        echo "  - $demo"
    done
    echo "  - all (generate all demos)"
}

check_dependencies() {
    local missing=()

    if ! command -v asciinema &> /dev/null; then
        missing+=("asciinema")
    fi

    if ! command -v agg &> /dev/null; then
        missing+=("agg")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        echo -e "${RED}Missing dependencies: ${missing[*]}${NC}"
        echo ""
        echo "Install with:"
        echo "  brew install asciinema"
        echo "  cargo install --git https://github.com/asciinema/agg"
        exit 1
    fi

    echo -e "${GREEN}Dependencies verified${NC}"
}

record_demo() {
    local demo_name="$1"
    local demo_script="${DEMO_SCRIPTS_DIR}/${demo_name}-demo.sh"
    local cast_file="${OUTPUT_DIR}/${demo_name}.cast"
    local gif_file="${OUTPUT_DIR}/${demo_name}.gif"
    local typescript_file="${OUTPUT_DIR}/${demo_name}.typescript"

    # Check if demo script exists
    if [ ! -f "$demo_script" ]; then
        echo -e "${RED}Demo script not found: ${demo_script}${NC}"
        return 1
    fi

    # Ensure demo script is executable
    chmod +x "$demo_script"

    echo -e "${BLUE}Recording demo: ${demo_name}${NC}"

    # Use macOS script command for reliable recording without TTY issues
    # Then convert to asciicast format
    local start_time=$(python3 -c "import time; print(time.time())")

    # Record using script command (works without TTY)
    script -q "$typescript_file" "$demo_script"

    local end_time=$(python3 -c "import time; print(time.time())")
    local duration=$(python3 -c "print(${end_time} - ${start_time})")

    # Convert typescript to asciicast v2 format
    echo -e "${BLUE}Converting to asciicast format...${NC}"

    python3 - "$typescript_file" "$cast_file" "$duration" <<'PYTHON'
import sys
import json
import re

typescript_file = sys.argv[1]
cast_file = sys.argv[2]
duration = float(sys.argv[3])

# Read typescript content
with open(typescript_file, 'rb') as f:
    content = f.read()

# Clean up control sequences and split into chunks
# Remove the "Script started/done" lines from macOS script command
lines = content.decode('utf-8', errors='replace').split('\n')
if lines and lines[0].startswith('Script started'):
    lines = lines[1:]
if lines and 'Script done' in lines[-1]:
    lines = lines[:-1]

content = '\n'.join(lines)

# Create asciicast v2 format
header = {
    "version": 2,
    "width": 100,
    "height": 25,
    "timestamp": int(float(sys.argv[3])),
    "env": {"SHELL": "/bin/bash", "TERM": "xterm-256color"}
}

with open(cast_file, 'w') as f:
    f.write(json.dumps(header) + '\n')

    # Split content into chunks and add timing
    chunks = re.split(r'(\x1b\[[0-9;]*m|\n)', content)
    time_per_chunk = duration / max(len(chunks), 1)
    current_time = 0.0

    for chunk in chunks:
        if chunk:
            f.write(json.dumps([current_time, "o", chunk]) + '\n')
            current_time += time_per_chunk

print(f"Created {cast_file}")
PYTHON

    # Clean up typescript file
    rm -f "$typescript_file"

    echo -e "${BLUE}Converting to GIF...${NC}"

    # Convert to GIF with agg
    agg \
        --theme=monokai \
        --font-size=14 \
        --cols=100 \
        --rows=25 \
        --speed=1.5 \
        "$cast_file" \
        "$gif_file"

    # Report file sizes
    local cast_size=$(du -h "$cast_file" | cut -f1)
    local gif_size=$(du -h "$gif_file" | cut -f1)

    echo -e "${GREEN}Generated:${NC}"
    echo "  Cast: ${cast_file} (${cast_size})"
    echo "  GIF:  ${gif_file} (${gif_size})"
    echo ""
}

record_all() {
    echo -e "${BLUE}Recording all demos...${NC}"
    echo ""

    for demo in "${DEMOS[@]}"; do
        if [ -f "${DEMO_SCRIPTS_DIR}/${demo}-demo.sh" ]; then
            record_demo "$demo"
        else
            echo -e "${YELLOW}Skipping ${demo}: demo script not found${NC}"
        fi
    done

    echo -e "${GREEN}All demos recorded!${NC}"
}

# Main
main() {
    if [ $# -eq 0 ]; then
        print_usage
        exit 1
    fi

    local demo_name="$1"

    # Create output directory
    mkdir -p "$OUTPUT_DIR"

    # Check dependencies
    check_dependencies

    if [ "$demo_name" = "all" ]; then
        record_all
    else
        # Validate demo name
        local valid=false
        for demo in "${DEMOS[@]}"; do
            if [ "$demo" = "$demo_name" ]; then
                valid=true
                break
            fi
        done

        if [ "$valid" = false ]; then
            echo -e "${RED}Unknown demo: ${demo_name}${NC}"
            echo ""
            print_usage
            exit 1
        fi

        record_demo "$demo_name"
    fi
}

main "$@"

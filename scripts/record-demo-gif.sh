#!/usr/bin/env bash
# Record demo GIFs with VHS (https://github.com/charmbracelet/vhs)
# Each demo is scripted as a tape in scripts/demo-tapes/<name>.tape
#
# Usage: ./scripts/record-demo-gif.sh [demo-name|all]
# Example: ./scripts/record-demo-gif.sh basic

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="${PROJECT_DIR}/docs/gif"
TAPE_DIR="${SCRIPT_DIR}/demo-tapes"

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
    echo "Records REAL Go demos with VHS."
    echo "Each recording makes real API calls - keep demos short to control costs."
    echo ""
    echo "Available demos:"
    for demo in "${DEMOS[@]}"; do
        echo "  - $demo"
    done
    echo "  - all (record all demos)"
}

check_dependencies() {
    local missing=()

    if ! command -v vhs &> /dev/null; then
        missing+=("vhs")
    fi

    if ! command -v just &> /dev/null; then
        missing+=("just")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        echo -e "${RED}Missing dependencies: ${missing[*]}${NC}"
        echo ""
        echo "Install with:"
        echo "  brew install vhs just"
        exit 1
    fi

    # Verify claude CLI is available and logged in
    if ! command -v claude &> /dev/null; then
        echo -e "${RED}claude CLI not found${NC}"
        echo "Install from: https://github.com/anthropics/claude-code"
        exit 1
    fi

    echo -e "${GREEN}Dependencies verified${NC}"
}

record_demo() {
    local demo_name="$1"
    local tape="${TAPE_DIR}/${demo_name}.tape"
    local gif_file="${OUTPUT_DIR}/${demo_name}.gif"

    if [ ! -f "$tape" ]; then
        echo -e "${RED}Tape not found: ${tape}${NC}"
        return 1
    fi

    echo -e "${BLUE}Recording demo: ${demo_name}${NC}"
    echo -e "${YELLOW}Note: This uses real API credits!${NC}"

    # Tapes use repository-relative paths for Output and Source
    (cd "$PROJECT_DIR" && vhs "$tape")

    if [ ! -f "$gif_file" ]; then
        echo -e "${RED}Recording failed - no GIF created${NC}"
        return 1
    fi

    local gif_size=$(du -h "$gif_file" | cut -f1)

    echo -e "${GREEN}Generated:${NC}"
    echo "  GIF:  ${gif_file} (${gif_size})"
    echo ""
}

record_all() {
    echo -e "${BLUE}Recording all demos...${NC}"
    echo -e "${YELLOW}Warning: This will make multiple real API calls!${NC}"
    echo ""

    for demo in "${DEMOS[@]}"; do
        if [ -f "${TAPE_DIR}/${demo}.tape" ]; then
            record_demo "$demo" || echo -e "${YELLOW}Failed to record ${demo}, continuing...${NC}"
        else
            echo -e "${YELLOW}Skipping ${demo}: tape not found${NC}"
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

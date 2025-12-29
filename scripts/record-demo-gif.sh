#!/usr/bin/env bash
# Record and generate demo GIFs using asciinema + agg
# Records REAL Go demos running with actual Claude API calls
#
# Usage: ./scripts/record-demo-gif.sh [demo-name]
# Example: ./scripts/record-demo-gif.sh basic

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="${PROJECT_DIR}/docs/gif"
DEMO_INPUTS_DIR="${SCRIPT_DIR}/demo-inputs"
BIN_DIR="${PROJECT_DIR}/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Available demos - must match input files in demo-inputs/
DEMOS=(basic streaming sessions mcp retry permissions budget plugins subagents)

# Map demo names to binary names
get_binary_name() {
    local demo_name="$1"
    case "$demo_name" in
        streaming)
            echo "demo"  # streaming is the default demo
            ;;
        *)
            echo "demo-${demo_name}"
            ;;
    esac
}

# Map demo names to just build commands
get_build_command() {
    local demo_name="$1"
    case "$demo_name" in
        streaming)
            echo "just demo build-streaming"
            ;;
        *)
            echo "just demo build-${demo_name}"
            ;;
    esac
}

print_usage() {
    echo "Usage: $0 <demo-name|all>"
    echo ""
    echo "Records REAL Go demos with actual Claude API interactions."
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

    if ! command -v asciinema &> /dev/null; then
        missing+=("asciinema")
    fi

    if ! command -v agg &> /dev/null; then
        missing+=("agg")
    fi

    if ! command -v just &> /dev/null; then
        missing+=("just")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        echo -e "${RED}Missing dependencies: ${missing[*]}${NC}"
        echo ""
        echo "Install with:"
        echo "  brew install asciinema just"
        echo "  cargo install --git https://github.com/asciinema/agg"
        exit 1
    fi

    echo -e "${GREEN}Dependencies verified${NC}"
}

record_demo() {
    local demo_name="$1"
    local input_file="${DEMO_INPUTS_DIR}/${demo_name}.txt"
    local cast_file="${OUTPUT_DIR}/${demo_name}.cast"
    local gif_file="${OUTPUT_DIR}/${demo_name}.gif"
    local binary_name=$(get_binary_name "$demo_name")
    local binary_path="${BIN_DIR}/${binary_name}"
    local build_cmd=$(get_build_command "$demo_name")

    # Check if input file exists
    if [ ! -f "$input_file" ]; then
        echo -e "${RED}Input file not found: ${input_file}${NC}"
        return 1
    fi

    echo -e "${BLUE}Recording demo: ${demo_name}${NC}"

    # Build the demo first
    echo -e "${BLUE}Building demo...${NC}"
    cd "$PROJECT_DIR"
    eval "$build_cmd"

    # Verify binary exists
    if [ ! -f "$binary_path" ]; then
        echo -e "${RED}Binary not found: ${binary_path}${NC}"
        return 1
    fi

    echo -e "${BLUE}Recording with real Claude API calls...${NC}"
    echo -e "${YELLOW}Note: This uses real API credits!${NC}"

    # Record with asciinema using input file
    # The demo reads from stdin, we pipe in the input file
    cat "$input_file" | asciinema rec "$cast_file" \
        --command="$binary_path" \
        --title="Claude Code Go SDK - ${demo_name} demo" \
        --idle-time-limit=2 \
        --overwrite \
        --format=asciicast

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
    echo -e "${YELLOW}Warning: This will make multiple real API calls!${NC}"
    echo ""

    for demo in "${DEMOS[@]}"; do
        if [ -f "${DEMO_INPUTS_DIR}/${demo}.txt" ]; then
            record_demo "$demo"
        else
            echo -e "${YELLOW}Skipping ${demo}: input file not found${NC}"
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

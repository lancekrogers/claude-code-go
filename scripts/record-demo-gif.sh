#!/usr/bin/env bash
# Record and generate demo GIFs using Docker + expect + asciinema
# Runs demos in isolated Docker container for filesystem safety
#
# Usage: ./scripts/record-demo-gif.sh [demo-name]
# Example: ./scripts/record-demo-gif.sh basic

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="${PROJECT_DIR}/docs/gif"
DOCKER_DIR="${SCRIPT_DIR}/docker"
EXPECT_DIR="${SCRIPT_DIR}/demo-expect"
BIN_DIR="${PROJECT_DIR}/bin"

# Docker image name
IMAGE_NAME="claude-code-go-demo-recorder"

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
    echo "Records REAL Go demos in isolated Docker container."
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

    if ! command -v docker &> /dev/null; then
        missing+=("docker")
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
        echo "  brew install just"
        echo "  cargo install --git https://github.com/asciinema/agg"
        echo "  (Docker should be installed and running)"
        exit 1
    fi

    # Check Docker is running
    if ! docker info &> /dev/null; then
        echo -e "${RED}Docker is not running${NC}"
        exit 1
    fi

    # Check API key is set
    if [ -z "$ANTHROPIC_API_KEY" ]; then
        echo -e "${RED}ANTHROPIC_API_KEY environment variable is not set${NC}"
        exit 1
    fi

    echo -e "${GREEN}Dependencies verified${NC}"
}

build_docker_image() {
    echo -e "${BLUE}Building Docker image...${NC}"
    docker build \
        -t "$IMAGE_NAME" \
        -f "${DOCKER_DIR}/Dockerfile.demo-recorder" \
        "$PROJECT_DIR"
}

# Get binary name for a demo
get_binary_name() {
    local demo_name="$1"
    case "$demo_name" in
        streaming)
            echo "demo"
            ;;
        *)
            echo "demo-${demo_name}"
            ;;
    esac
}

# Build demo binary for Linux (cross-compile)
build_demo_for_linux() {
    local demo_name="$1"
    local binary_name=$(get_binary_name "$demo_name")
    local demo_dir="${PROJECT_DIR}/examples/demo/${demo_name}"
    local output_path="${BIN_DIR}/linux/${binary_name}"

    echo -e "${BLUE}Building ${demo_name} for Linux...${NC}"

    mkdir -p "${BIN_DIR}/linux"

    # Cross-compile for Linux
    cd "$demo_dir"
    GOOS=linux GOARCH=amd64 go build -o "$output_path" ./cmd/demo

    echo -e "${GREEN}Built: ${output_path}${NC}"
}

record_demo() {
    local demo_name="$1"
    local binary_name=$(get_binary_name "$demo_name")
    local expect_script="${EXPECT_DIR}/${demo_name}.exp"
    local cast_file="${OUTPUT_DIR}/${demo_name}.cast"
    local gif_file="${OUTPUT_DIR}/${demo_name}.gif"

    # Check expect script exists
    if [ ! -f "$expect_script" ]; then
        echo -e "${RED}Expect script not found: ${expect_script}${NC}"
        return 1
    fi

    echo -e "${BLUE}Recording demo: ${demo_name}${NC}"

    # Build demo for Linux
    build_demo_for_linux "$demo_name"

    echo -e "${BLUE}Running in Docker container...${NC}"
    echo -e "${YELLOW}Note: This uses real API credits!${NC}"

    # Run in Docker with:
    # - Binary mounted to /demo/bin/
    # - Expect script mounted to /expect/
    # - Output directory mounted to /output/
    # - API key passed as environment variable
    docker run --rm \
        -v "${BIN_DIR}/linux:/demo/bin:ro" \
        -v "${EXPECT_DIR}:/expect:ro" \
        -v "${OUTPUT_DIR}:/output" \
        -e "ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}" \
        "$IMAGE_NAME" \
        "$demo_name"

    # Check if cast file was created
    if [ ! -f "$cast_file" ]; then
        echo -e "${RED}Recording failed - no cast file created${NC}"
        return 1
    fi

    echo -e "${BLUE}Converting to GIF...${NC}"

    # Convert to GIF with agg (runs on host)
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
        if [ -f "${EXPECT_DIR}/${demo}.exp" ]; then
            record_demo "$demo" || echo -e "${YELLOW}Failed to record ${demo}, continuing...${NC}"
        else
            echo -e "${YELLOW}Skipping ${demo}: expect script not found${NC}"
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

    # Build Docker image
    build_docker_image

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

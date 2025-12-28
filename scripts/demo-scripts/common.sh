#!/usr/bin/env bash
# Common utilities for demo scripts
# Source this file in demo scripts: source "$(dirname "$0")/common.sh"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

# Speed control (fast, normal, slow)
DEMO_SPEED=${DEMO_SPEED:-"normal"}

case $DEMO_SPEED in
    "fast")
        SHORT_PAUSE=0.3
        MEDIUM_PAUSE=0.6
        LONG_PAUSE=1
        TYPING_DELAY=0.01
        ;;
    "slow")
        SHORT_PAUSE=1.5
        MEDIUM_PAUSE=3
        LONG_PAUSE=5
        TYPING_DELAY=0.08
        ;;
    *)  # normal
        SHORT_PAUSE=0.8
        MEDIUM_PAUSE=1.5
        LONG_PAUSE=2.5
        TYPING_DELAY=0.03
        ;;
esac

# Typewriter effect for realistic typing
typewriter() {
    local text="$1"
    local delay=${2:-$TYPING_DELAY}

    for (( i=0; i<${#text}; i++ )); do
        printf "%s" "${text:$i:1}"
        sleep "$delay"
    done
}

# Print a header with box
print_header() {
    local title="$1"
    local width=60
    local padding=$(( (width - ${#title} - 2) / 2 ))

    echo ""
    echo -e "${BOLD}${BLUE}╔$(printf '═%.0s' $(seq 1 $width))╗${NC}"
    printf "${BOLD}${BLUE}║${NC}"
    printf "%*s" $padding ""
    echo -n " $title "
    printf "%*s" $(( width - padding - ${#title} - 2 )) ""
    echo -e "${BOLD}${BLUE}║${NC}"
    echo -e "${BOLD}${BLUE}╚$(printf '═%.0s' $(seq 1 $width))╝${NC}"
    echo ""
}

# Print section divider
print_section() {
    local title="$1"
    echo ""
    echo -e "${YELLOW}━━━ ${title} ━━━${NC}"
    echo ""
}

# Simulate user typing a command
type_command() {
    local cmd="$1"
    echo -ne "${GREEN}\$ ${NC}"
    typewriter "$cmd"
    echo ""
    sleep $SHORT_PAUSE
}

# Simulate running a command (type then execute)
run_command() {
    local cmd="$1"
    type_command "$cmd"
    eval "$cmd"
    sleep $MEDIUM_PAUSE
}

# Show output text (simulated response)
show_output() {
    local text="$1"
    echo -e "${CYAN}${text}${NC}"
    sleep $SHORT_PAUSE
}

# Show code block
show_code() {
    local code="$1"
    echo -e "${DIM}${code}${NC}"
}

# Show success message
show_success() {
    local msg="$1"
    echo -e "${GREEN}✓ ${msg}${NC}"
}

# Show info message
show_info() {
    local msg="$1"
    echo -e "${BLUE}ℹ ${msg}${NC}"
}

# Clear screen and wait
clear_pause() {
    clear
    sleep $SHORT_PAUSE
}

# End demo with summary
end_demo() {
    echo ""
    echo -e "${BOLD}${GREEN}Demo complete!${NC}"
    echo ""
    sleep $LONG_PAUSE
}

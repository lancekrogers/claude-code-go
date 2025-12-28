#!/usr/bin/env bash
# Basic Demo - Claude Code Go SDK
source "$(dirname "$0")/common.sh"

clear
print_header "Claude Code Go SDK - Basic Demo"

show_info "The basic demo shows fundamental SDK usage patterns"
sleep $MEDIUM_PAUSE

print_section "Creating a Claude Client"

show_code 'client := claude.NewClient("claude")'
show_output "Creates a new client wrapping the claude CLI"
sleep $MEDIUM_PAUSE

print_section "Running a Simple Prompt"

show_code 'result, err := client.RunPrompt("What is 2+2?", nil)'
show_output "Executes a prompt and returns structured result"
sleep $SHORT_PAUSE

show_code 'fmt.Println(result.Result)  // "4"'
show_code 'fmt.Println(result.CostUSD) // 0.000123'
sleep $MEDIUM_PAUSE

print_section "Using RunOptions"

show_code 'opts := &claude.RunOptions{'
show_code '    Format:    claude.JSONOutput,'
show_code '    MaxTurns:  5,'
show_code '    Model:     "claude-sonnet-4-20250514",'
show_code '}'
show_output "Configure output format, turn limits, and model"
sleep $MEDIUM_PAUSE

print_section "Key Features"

show_success "Simple API wrapping claude CLI"
show_success "Structured JSON output parsing"
show_success "Cost and duration tracking"
show_success "Session management support"

end_demo

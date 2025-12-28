#!/usr/bin/env bash
# Subagents Demo - Claude Code Go SDK
source "$(dirname "$0")/common.sh"

clear
print_header "Claude Code Go SDK - Subagents Demo"

show_info "Orchestrate specialized AI agents for complex tasks"
sleep $MEDIUM_PAUSE

print_section "Subagent Types"

show_success "Explore    - Fast codebase exploration"
show_success "Plan       - Design implementation approaches"
show_success "Code       - Write and modify code"
show_success "Review     - Code review and analysis"
sleep $MEDIUM_PAUSE

print_section "Using Subagent Mode"

show_code 'opts := &claude.RunOptions{'
show_code '    SubagentMode: true,'
show_code '    SubagentType: claude.SubagentTypeExplore,'
show_code '}'
show_output "Configure subagent behavior"
sleep $MEDIUM_PAUSE

print_section "Orchestration Pattern"

show_code '// Step 1: Explore the codebase'
show_code 'exploreResult, _ := client.RunPrompt('
show_code '    "Find authentication handlers",'
show_code '    &claude.RunOptions{SubagentType: claude.SubagentTypeExplore},'
show_code ')'
sleep $SHORT_PAUSE

show_code '// Step 2: Plan the implementation'
show_code 'planResult, _ := client.RunPrompt('
show_code '    "Plan OAuth2 integration using: " + exploreResult.Result,'
show_code '    &claude.RunOptions{SubagentType: claude.SubagentTypePlan},'
show_code ')'
sleep $MEDIUM_PAUSE

print_section "Subagent Benefits"

show_success "Specialized agents for focused tasks"
show_success "Better results through task decomposition"
show_success "Parallel execution where applicable"

end_demo

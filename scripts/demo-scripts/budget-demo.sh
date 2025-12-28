#!/usr/bin/env bash
# Budget Demo - Claude Code Go SDK
source "$(dirname "$0")/common.sh"

clear
print_header "Claude Code Go SDK - Budget Demo"

show_info "Track and limit API costs across your application"
sleep $MEDIUM_PAUSE

print_section "Per-Request Budget Limit"

show_code 'opts := &claude.RunOptions{'
show_code '    MaxBudgetUSD: 0.50,  // $0.50 max per request'
show_code '}'
show_output "Stop execution if cost exceeds limit"
sleep $MEDIUM_PAUSE

print_section "Shared Budget Tracker"

show_code 'tracker := claude.NewBudgetTracker()'
show_code 'tracker.SetMaxBudget(10.00)  // $10 total budget'
sleep $SHORT_PAUSE

show_code 'opts := &claude.RunOptions{'
show_code '    BudgetTracker: tracker,'
show_code '}'
show_output "Track costs across multiple requests"
sleep $MEDIUM_PAUSE

print_section "Budget Monitoring"

show_code 'spent := tracker.TotalSpent()'
show_code 'remaining := tracker.RemainingBudget()'
show_code 'exceeded := tracker.WouldExceedBudget(0.10)'
sleep $SHORT_PAUSE

show_output "spent:     $4.23"
show_output "remaining: $5.77"
show_output "exceeded:  false"
sleep $MEDIUM_PAUSE

print_section "Cost from Results"

show_code 'result, _ := client.RunPrompt(prompt, opts)'
show_code 'fmt.Printf("Cost: $%.6f", result.CostUSD)'
show_output "Cost: $0.001234"
sleep $SHORT_PAUSE

show_success "Track costs per request and across sessions"
show_success "Set hard limits to prevent runaway costs"

end_demo

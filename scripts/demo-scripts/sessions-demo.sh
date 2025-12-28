#!/usr/bin/env bash
# Sessions Demo - Claude Code Go SDK
source "$(dirname "$0")/common.sh"

clear
print_header "Claude Code Go SDK - Sessions Demo"

show_info "Manage conversations with session IDs, forking, and persistence"
sleep $MEDIUM_PAUSE

print_section "Explicit Session ID"

show_code 'sessionID := claude.GenerateSessionID()'
show_output "Creates UUID: 550e8400-e29b-41d4-a716-446655440000"
sleep $SHORT_PAUSE

show_code 'result, _ := client.RunWithSession(prompt, sessionID, opts)'
show_output "Track conversations with explicit session IDs"
sleep $MEDIUM_PAUSE

print_section "Resume Conversations"

show_code 'result, _ := client.ResumeConversation('
show_code '    "What about Tokyo?",  // Follow-up question'
show_code '    sessionID,            // Same session'
show_code ')'
show_output "Continue previous conversation with full context"
sleep $MEDIUM_PAUSE

print_section "Fork Sessions"

show_code 'result, _ := client.ForkAndRun('
show_code '    "Try a different approach...",'
show_code '    originalSessionID,'
show_code '    opts,'
show_code ')'
show_output "Branch from existing conversation, original stays intact"
sleep $MEDIUM_PAUSE

print_section "Ephemeral Sessions"

show_code 'result, _ := client.RunEphemeral(prompt, opts)'
show_output "One-off queries, NOT saved to disk"
sleep $SHORT_PAUSE

show_success "Perfect for sensitive one-off queries"
show_success "No session persistence overhead"

end_demo

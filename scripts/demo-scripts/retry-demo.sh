#!/usr/bin/env bash
# Retry Demo - Claude Code Go SDK
source "$(dirname "$0")/common.sh"

clear
print_header "Claude Code Go SDK - Retry Demo"

show_info "Automatic retry with exponential backoff and jitter"
sleep $MEDIUM_PAUSE

print_section "Default Retry Behavior"

show_code 'opts := &claude.RunOptions{'
show_code '    MaxRetries: 3,  // Retry up to 3 times'
show_code '}'
show_output "Simple retry configuration for transient failures"
sleep $MEDIUM_PAUSE

print_section "Custom Retry Policy"

show_code 'opts := &claude.RunOptions{'
show_code '    RetryPolicy: &claude.RetryPolicy{'
show_code '        MaxAttempts:  5,'
show_code '        InitialDelay: 100 * time.Millisecond,'
show_code '        MaxDelay:     30 * time.Second,'
show_code '        Multiplier:   2.0,'
show_code '        JitterFactor: 0.2,  // +/- 20% randomization'
show_code '    },'
show_code '}'
sleep $MEDIUM_PAUSE

print_section "Backoff Progression"

show_info "Attempt 1: 100ms"
show_info "Attempt 2: 200ms (+/- jitter)"
show_info "Attempt 3: 400ms (+/- jitter)"
show_info "Attempt 4: 800ms (+/- jitter)"
show_info "Attempt 5: 1.6s (+/- jitter)"
sleep $MEDIUM_PAUSE

print_section "Retryable Errors"

show_success "Rate limiting (429)"
show_success "Server overload (503)"
show_success "Network timeouts"
show_success "Temporary failures"

end_demo

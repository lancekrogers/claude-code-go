#!/usr/bin/env bash
# Streaming Demo - Claude Code Go SDK
source "$(dirname "$0")/common.sh"

clear
print_header "Claude Code Go SDK - Streaming Demo"

show_info "Real-time message streaming for responsive applications"
sleep $MEDIUM_PAUSE

print_section "Basic Streaming"

show_code 'msgCh, errCh := client.StreamPrompt(ctx, prompt, opts)'
show_output "Returns two channels: messages and errors"
sleep $MEDIUM_PAUSE

print_section "Processing Stream Messages"

show_code 'for msg := range msgCh {'
show_code '    switch msg.Type {'
show_code '    case "system":'
show_code '        fmt.Println("Tools:", msg.Tools)'
show_code '    case "assistant":'
show_code '        fmt.Println("Response:", msg.Message)'
show_code '    case "result":'
show_code '        fmt.Println("Complete:", msg.Result)'
show_code '    }'
show_code '}'
sleep $MEDIUM_PAUSE

print_section "Message Types"

show_success "system    - Initial setup, available tools"
show_success "assistant - Claude's response chunks"
show_success "user      - Echoed user messages"
show_success "result    - Final result with cost/duration"
sleep $MEDIUM_PAUSE

print_section "Error Handling"

show_code 'go func() {'
show_code '    for err := range errCh {'
show_code '        log.Printf("Stream error: %v", err)'
show_code '    }'
show_code '}()'
show_output "Handle errors in a separate goroutine"

end_demo

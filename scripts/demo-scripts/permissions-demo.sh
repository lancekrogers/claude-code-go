#!/usr/bin/env bash
# Permissions Demo - Claude Code Go SDK
source "$(dirname "$0")/common.sh"

clear
print_header "Claude Code Go SDK - Permissions Demo"

show_info "Fine-grained control over Claude's tool access"
sleep $MEDIUM_PAUSE

print_section "Permission Modes"

show_code 'opts := &claude.RunOptions{'
show_code '    PermissionMode: claude.PermissionModeDefault,'
show_code '}'
show_output "PermissionModeDefault - Standard permission checks"
sleep $SHORT_PAUSE

show_code 'PermissionModeAcceptEdits  // Auto-approve file edits'
show_code 'PermissionModeBypassPermissions  // DANGEROUS - skip all'
sleep $MEDIUM_PAUSE

print_section "Tool Whitelisting"

show_code 'opts := &claude.RunOptions{'
show_code '    AllowedTools: []string{'
show_code '        "Read",'
show_code '        "Grep",'
show_code '        "Bash(git status:*)",'
show_code '        "Bash(go test:*)",'
show_code '    },'
show_code '}'
show_output "Only allow specific tools and commands"
sleep $MEDIUM_PAUSE

print_section "Tool Blacklisting"

show_code 'DisallowedTools: []string{'
show_code '    "Bash(rm:*)",'
show_code '    "Bash(sudo:*)",'
show_code '}'
show_output "Block dangerous operations"
sleep $MEDIUM_PAUSE

print_section "Pre-built Callbacks"

show_success "ReadOnlyCallback()     - Read, Grep, Glob only"
show_success "SafeBashCallback()     - Block dangerous commands"
show_success "FilePathCallback()     - Restrict file access paths"
show_success "ChainCallbacks()       - Combine multiple callbacks"

end_demo

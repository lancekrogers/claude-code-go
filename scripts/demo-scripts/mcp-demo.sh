#!/usr/bin/env bash
# MCP Demo - Claude Code Go SDK
source "$(dirname "$0")/common.sh"

clear
print_header "Claude Code Go SDK - MCP Demo"

show_info "Model Context Protocol - Extend Claude with external tools"
sleep $MEDIUM_PAUSE

print_section "Building MCP Configuration"

show_code 'builder := claude.NewMCPConfigBuilder()'
show_output "Fluent API for building MCP server configs"
sleep $SHORT_PAUSE

show_code 'builder.AddHTTPServer("api", "https://api.example.com/mcp")'
show_code 'builder.AddSSEServer("events", "https://stream.example.com")'
show_code 'builder.AddStdioServer("db", "sqlite-mcp", "--db", "data.db")'
sleep $MEDIUM_PAUSE

print_section "Server Types"

show_success "HTTP  - REST API endpoints"
show_success "SSE   - Server-Sent Events streaming"
show_success "Stdio - Local process via stdin/stdout"
sleep $MEDIUM_PAUSE

print_section "Environment Variables"

show_code 'builder.WithEnv("api", map[string]string{'
show_code '    "API_KEY": os.Getenv("API_KEY"),'
show_code '})'
show_output "Inject credentials securely from environment"
sleep $MEDIUM_PAUSE

print_section "Using with Claude"

show_code 'opts := &claude.RunOptions{'
show_code '    MCPConfigPath: "/path/to/mcp.json",'
show_code '    AllowedTools: []string{"mcp__api__search"},'
show_code '}'
show_output "Load config and whitelist MCP tools"
sleep $SHORT_PAUSE

show_code 'result, _ := client.RunPrompt("Search for...", opts)'

end_demo

#!/usr/bin/env bash
# Plugins Demo - Claude Code Go SDK
source "$(dirname "$0")/common.sh"

clear
print_header "Claude Code Go SDK - Plugins Demo"

show_info "Extend SDK behavior with the plugin system"
sleep $MEDIUM_PAUSE

print_section "Built-in Plugins"

show_success "LoggingPlugin     - Log all prompts/results"
show_success "MetricsPlugin     - Collect usage statistics"
show_success "ToolFilterPlugin  - Filter tool access"
show_success "AuditPlugin       - Security audit trail"
sleep $MEDIUM_PAUSE

print_section "Using Plugins"

show_code 'client := claude.NewClient("claude")'
show_code ''
show_code 'loggingPlugin := claude.NewLoggingPlugin(logger)'
show_code 'metricsPlugin := claude.NewMetricsPlugin()'
show_code ''
show_code 'client.RegisterPlugin(loggingPlugin)'
show_code 'client.RegisterPlugin(metricsPlugin)'
sleep $MEDIUM_PAUSE

print_section "Plugin Interface"

show_code 'type Plugin interface {'
show_code '    Name() string'
show_code '    BeforeRun(ctx, prompt, opts) error'
show_code '    AfterRun(ctx, result, err) error'
show_code '    Shutdown() error'
show_code '}'
sleep $SHORT_PAUSE
show_output "Hooks for pre/post processing and cleanup"
sleep $MEDIUM_PAUSE

print_section "Plugin Manager"

show_code 'pm := client.PluginManager()'
show_code 'pm.EnablePlugin("logging")'
show_code 'pm.DisablePlugin("metrics")'
show_code 'pm.Shutdown()  // Clean up all plugins'

end_demo

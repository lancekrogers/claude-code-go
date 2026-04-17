package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

// CustomSecurityPlugin demonstrates creating a custom plugin
type CustomSecurityPlugin struct {
	claude.BasePlugin
	blockedPaths []string
	auditLog     []string
}

func NewSecurityPlugin(blockedPaths []string) *CustomSecurityPlugin {
	return &CustomSecurityPlugin{
		BasePlugin: claude.BasePlugin{
			PluginName:    "security",
			PluginVersion: "1.0.0",
		},
		blockedPaths: blockedPaths,
		auditLog:     make([]string, 0),
	}
}

func (sp *CustomSecurityPlugin) OnToolCall(ctx context.Context, toolName string, input claude.ToolInput) error {
	// Log all tool calls
	entry := fmt.Sprintf("[%s] Tool: %s", time.Now().Format(time.RFC3339), toolName)
	sp.auditLog = append(sp.auditLog, entry)

	// Block access to sensitive paths
	if toolName == "Read" || toolName == "Write" || toolName == "Edit" {
		for _, blocked := range sp.blockedPaths {
			if input.FilePath == blocked {
				return fmt.Errorf("access denied: %s is a protected path", blocked)
			}
		}
	}

	return nil
}

func (sp *CustomSecurityPlugin) GetAuditLog() []string {
	return sp.auditLog
}

func main() {
	cc := claude.NewClient("claude")

	// Example 1: Using built-in LoggingPlugin
	fmt.Println("Example 1: Logging Plugin")
	fmt.Println("=========================")

	pm := claude.NewPluginManager()

	// Add logging plugin
	loggingPlugin := claude.NewLoggingPlugin(func(format string, args ...interface{}) {
		log.Printf("[LOG] "+format, args...)
	})
	pm.Register(loggingPlugin, &claude.PluginConfig{
		Enabled:  true,
		Priority: 10, // Lower = runs first
	})

	// Initialize plugins
	ctx := context.Background()
	if err := pm.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize plugins: %v", err)
	}
	defer pm.Shutdown(ctx)

	opts := &claude.RunOptions{
		Format:        claude.JSONOutput,
		PluginManager: pm,
	}

	result, err := cc.RunPrompt("What is 5+5?", opts)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Result: %s\n", result.Result)
	}

	// Example 2: Metrics Plugin
	fmt.Println("\n\nExample 2: Metrics Plugin")
	fmt.Println("=========================")

	pm2 := claude.NewPluginManager()
	metricsPlugin := claude.NewMetricsPlugin()
	pm2.Register(metricsPlugin, nil)
	pm2.Initialize(ctx)

	// Simulate some tool calls
	pm2.OnToolCall(ctx, "Bash", claude.ToolInput{Command: "ls -la"})
	pm2.OnToolCall(ctx, "Bash", claude.ToolInput{Command: "git status"})
	pm2.OnToolCall(ctx, "Read", claude.ToolInput{FilePath: "/etc/hosts"})
	pm2.OnMessage(ctx, claude.Message{Type: "assistant"})
	pm2.OnComplete(ctx, &claude.ClaudeResult{CostUSD: 0.05, NumTurns: 3})

	metrics := metricsPlugin.GetMetrics()
	fmt.Printf("Tool calls: %v\n", metrics["tool_calls"])
	fmt.Printf("Message count: %d\n", metrics["message_count"])
	fmt.Printf("Total cost: $%.4f\n", metrics["total_cost"])
	fmt.Printf("Executions: %d\n", metrics["execution_count"])

	// Example 3: Tool Filter Plugin
	fmt.Println("\n\nExample 3: Tool Filter Plugin")
	fmt.Println("=============================")

	pm3 := claude.NewPluginManager()
	filterPlugin := claude.NewToolFilterPlugin(map[string]string{
		"Bash":  "Shell commands are disabled for safety",
		"Write": "Write operations are not allowed",
	})
	pm3.Register(filterPlugin, nil)

	// Test blocked tool
	err = pm3.OnToolCall(ctx, "Bash", claude.ToolInput{Command: "rm -rf /"})
	if err != nil {
		fmt.Printf("Blocked (expected): %v\n", err)
	}

	// Test allowed tool
	err = pm3.OnToolCall(ctx, "Read", claude.ToolInput{FilePath: "main.go"})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Read tool allowed")
	}

	// Example 4: Audit Plugin
	fmt.Println("\n\nExample 4: Audit Plugin")
	fmt.Println("=======================")

	pm4 := claude.NewPluginManager()
	auditPlugin := claude.NewAuditPlugin(100) // Keep last 100 records
	pm4.Register(auditPlugin, nil)

	// Simulate activity
	pm4.OnToolCall(ctx, "Read", claude.ToolInput{
		FilePath: "config.yaml",
		Raw:      map[string]interface{}{"file_path": "config.yaml"},
	})
	pm4.OnToolCall(ctx, "Grep", claude.ToolInput{
		Pattern: "password",
		Raw:     map[string]interface{}{"pattern": "password"},
	})

	records := auditPlugin.GetRecords()
	fmt.Printf("Audit records: %d\n", len(records))
	for _, r := range records {
		fmt.Printf("  - %s: %v\n", r.ToolName, r.Input)
	}

	// Example 5: Custom Security Plugin
	fmt.Println("\n\nExample 5: Custom Security Plugin")
	fmt.Println("==================================")

	pm5 := claude.NewPluginManager()
	securityPlugin := NewSecurityPlugin([]string{
		"/etc/passwd",
		"/etc/shadow",
		"~/.ssh/id_rsa",
	})
	pm5.Register(securityPlugin, &claude.PluginConfig{
		Enabled:  true,
		Priority: 1, // Run first for security
	})

	// Test security plugin
	err = pm5.OnToolCall(ctx, "Read", claude.ToolInput{FilePath: "/etc/passwd"})
	if err != nil {
		fmt.Printf("Blocked (expected): %v\n", err)
	}

	err = pm5.OnToolCall(ctx, "Read", claude.ToolInput{FilePath: "main.go"})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Safe file access allowed")
	}

	fmt.Printf("Audit log entries: %d\n", len(securityPlugin.GetAuditLog()))

	// Example 6: Chaining Multiple Plugins
	fmt.Println("\n\nExample 6: Plugin Chain")
	fmt.Println("========================")

	pm6 := claude.NewPluginManager()
	pm6.Register(claude.NewLoggingPlugin(log.Printf), &claude.PluginConfig{Priority: 10})
	pm6.Register(claude.NewMetricsPlugin(), &claude.PluginConfig{Priority: 20})
	pm6.Register(claude.NewAuditPlugin(50), &claude.PluginConfig{Priority: 30})

	fmt.Printf("Registered plugins: %v\n", pm6.List())
	fmt.Printf("Plugin count: %d\n", pm6.Count())

	// Disable a plugin
	pm6.SetEnabled("logging", false)
	fmt.Println("Disabled logging plugin")

	fmt.Println("\nPlugin examples complete!")
}

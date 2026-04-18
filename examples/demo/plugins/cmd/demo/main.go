package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

var (
	pluginManager *claude.PluginManager
	metricsPlugin *claude.MetricsPlugin
	auditPlugin   *claude.AuditPlugin
)

func isExitCommand(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	exitCommands := []string{"exit", "quit", "bye", "q", ":q", "done", "stop"}
	for _, cmd := range exitCommands {
		if input == cmd {
			return true
		}
	}
	return false
}

func displayMetrics() {
	metrics := metricsPlugin.GetMetrics()
	fmt.Println("\n📊 Plugin Metrics Dashboard:")
	fmt.Println("   ┌─────────────────────────────────────┐")
	fmt.Printf("   │ Tool Calls:      %-18v │\n", metrics["tool_calls"])
	fmt.Printf("   │ Messages:        %-18d │\n", metrics["message_count"])
	fmt.Printf("   │ Executions:      %-18d │\n", metrics["execution_count"])
	fmt.Printf("   │ Total Cost:      $%-17.6f │\n", metrics["total_cost"])
	fmt.Println("   └─────────────────────────────────────┘")
}

func displayAuditLog() {
	records := auditPlugin.GetRecords()
	if len(records) == 0 {
		fmt.Println("\n📋 Audit Log: (empty)")
		return
	}
	fmt.Println("\n📋 Audit Log (last 5):")
	start := 0
	if len(records) > 5 {
		start = len(records) - 5
	}
	for _, r := range records[start:] {
		ts := time.Unix(0, r.Timestamp*int64(time.Millisecond))
		fmt.Printf("   • [%s] %s\n", ts.Format("15:04:05"), r.ToolName)
	}
}

func displayStreamingMessage(msg claude.Message) {
	switch msg.Type {
	case "system":
		if msg.Subtype == "init" {
			fmt.Printf("🔄 Session initialized: %s\n", msg.SessionID[:8])
		}
	case "assistant":
		var content map[string]interface{}
		if err := json.Unmarshal(msg.Message, &content); err == nil {
			if contentArray, ok := content["content"].([]interface{}); ok {
				for _, item := range contentArray {
					if itemMap, ok := item.(map[string]interface{}); ok {
						if itemMap["type"] == "text" {
							if text, ok := itemMap["text"].(string); ok && strings.TrimSpace(text) != "" {
								fmt.Printf("💬 %s\n", text)
							}
						} else if itemMap["type"] == "tool_use" {
							if name, ok := itemMap["name"].(string); ok {
								fmt.Printf("🔧 Tool: %s\n", name)
							}
						}
					}
				}
			}
		}
	case "result":
		if msg.IsError {
			fmt.Printf("❌ Error: %s\n", msg.Result)
		} else {
			fmt.Printf("✅ Complete - Cost: $%.6f | Turns: %d\n", msg.CostUSD, msg.NumTurns)
		}
	}
}

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║        Claude Code Go SDK - Plugin System Demo             ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Initialize plugin manager
	pluginManager = claude.NewPluginManager()
	ctx := context.Background()

	// Register logging plugin
	loggingPlugin := claude.NewLoggingPlugin(func(format string, args ...interface{}) {
		fmt.Printf("   [LOG] "+format+"\n", args...)
	})
	pluginManager.Register(loggingPlugin, &claude.PluginConfig{
		Enabled:  true,
		Priority: 10,
	})

	// Register metrics plugin
	metricsPlugin = claude.NewMetricsPlugin()
	pluginManager.Register(metricsPlugin, &claude.PluginConfig{
		Enabled:  true,
		Priority: 20,
	})

	// Register audit plugin
	auditPlugin = claude.NewAuditPlugin(100)
	pluginManager.Register(auditPlugin, &claude.PluginConfig{
		Enabled:  true,
		Priority: 30,
	})

	// Register tool filter plugin (blocks dangerous commands)
	filterPlugin := claude.NewToolFilterPlugin(map[string]string{
		"Write": "Write operations blocked for demo safety",
	})
	pluginManager.Register(filterPlugin, &claude.PluginConfig{
		Enabled:  true,
		Priority: 1, // Run first for security
	})

	// Initialize all plugins
	if err := pluginManager.Initialize(ctx); err != nil {
		fmt.Printf("❌ Plugin initialization failed: %v\n", err)
		return
	}
	defer pluginManager.Shutdown(ctx)

	fmt.Println("📦 Registered Plugins:")
	for _, name := range pluginManager.List() {
		fmt.Printf("   • %s\n", name)
	}
	fmt.Println()

	fmt.Println("🔒 Security: Write operations are blocked by ToolFilterPlugin")
	fmt.Println("📊 Metrics and audit logging are active")
	fmt.Println()

	cc := claude.NewClient("claude")

	opts := &claude.RunOptions{
		Format:        claude.StreamJSONOutput,
		SystemPrompt:  "You are a helpful assistant. Keep responses concise.",
		PluginManager: pluginManager,
		AllowedTools: []string{
			"Read(*)",
			"Bash(ls*)",
			"Bash(pwd)",
			"Bash(echo*)",
			"Bash(cat*)",
			"Glob(*)",
		},
	}

	var sessionID string
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("🚀 Interactive session started. Commands:")
	fmt.Println("   • 'metrics' - Show plugin metrics")
	fmt.Println("   • 'audit'   - Show audit log")
	fmt.Println("   • 'exit'    - End session")
	fmt.Println()

	for {
		fmt.Print(">>> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" || isExitCommand(input) {
			break
		}

		// Handle special commands
		if input == "metrics" {
			displayMetrics()
			continue
		}
		if input == "audit" {
			displayAuditLog()
			continue
		}

		// Continue or start conversation
		if sessionID != "" {
			opts.ResumeID = sessionID
		}

		messageCh, errCh := cc.StreamPrompt(ctx, input, opts)

	processLoop:
		for {
			select {
			case msg, ok := <-messageCh:
				if !ok {
					break processLoop
				}
				displayStreamingMessage(msg)
				if msg.SessionID != "" {
					sessionID = msg.SessionID
				}
				if msg.Type == "result" {
					break processLoop
				}
			case err := <-errCh:
				if err != nil {
					fmt.Printf("❌ Error: %v\n", err)
					break processLoop
				}
			}
		}

		// Brief pause then show quick metrics
		time.Sleep(100 * time.Millisecond)
		metrics := metricsPlugin.GetMetrics()
		fmt.Printf("   📈 Metrics: %v tool calls | %d messages\n",
			metrics["tool_calls"], metrics["message_count"])
		fmt.Println()
	}

	// Final summary
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     Session Summary                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	displayMetrics()
	displayAuditLog()
	fmt.Println("\nDemo completed!")
}

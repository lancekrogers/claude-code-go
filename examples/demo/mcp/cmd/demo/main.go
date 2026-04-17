package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

var (
	configFile   string
	strictMode   bool
	allowedTools []string
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

func displayMCPStatus() {
	fmt.Println("\n┌─────────────────────────────────────────────────────────────┐")

	// MCP Status
	if configFile != "" {
		fmt.Println("│ MCP Status: 🟢 Configured                                   │")
		fmt.Printf("│ Config File: %-47s │\n", truncatePath(configFile))
	} else {
		fmt.Println("│ MCP Status: ⚪ Not configured                                │")
		fmt.Println("│ (Use /example to create sample config)                      │")
	}

	// Strict Mode
	strictIcon := "⚪"
	strictDesc := "off"
	if strictMode {
		strictIcon = "🔒"
		strictDesc = "on (MCP servers only)"
	}
	fmt.Printf("│ Strict Mode: %s %-44s │\n", strictIcon, strictDesc)

	// Tool Allowlist
	if len(allowedTools) > 0 {
		fmt.Printf("│ Allowed Tools: %-45d │\n", len(allowedTools))
		// Show first 3 tools
		maxShow := 3
		for i, tool := range allowedTools {
			if i >= maxShow {
				remaining := len(allowedTools) - maxShow
				fmt.Printf("│   ... and %d more%-42s │\n", remaining, "")
				break
			}
			// Truncate long tool names
			displayTool := tool
			if len(displayTool) > 50 {
				displayTool = displayTool[:47] + "..."
			}
			fmt.Printf("│   • %-56s │\n", displayTool)
		}
	} else {
		fmt.Println("│ Allowed Tools: All tools enabled                            │")
	}

	fmt.Println("└─────────────────────────────────────────────────────────────┘")
}

func truncatePath(path string) string {
	if len(path) > 48 {
		return "..." + path[len(path)-45:]
	}
	if path == "" {
		return "(none)"
	}
	return path
}

func displayHelp() {
	fmt.Println("\n📚 MCP Commands:")
	fmt.Println("  /config <path>  - Load MCP config from file")
	fmt.Println("  /example        - Create and use example MCP config")
	fmt.Println("  /strict on|off  - Toggle strict MCP mode")
	fmt.Println("  /tools          - List allowed MCP tools")
	fmt.Println("  /allow <tool>   - Add tool to allowlist")
	fmt.Println("  /clear          - Clear MCP configuration")
	fmt.Println("  /status         - Show current MCP status")
	fmt.Println("  /help           - Show this help")
	fmt.Println("  exit            - Exit the demo")
}

func checkPrerequisites() error {
	// Check for npx
	fmt.Print("🔍 Checking for npx... ")
	_, err := exec.LookPath("npx")
	if err != nil {
		fmt.Println("❌")
		return fmt.Errorf("npx not found - install Node.js first")
	}
	fmt.Println("✅")

	// Check for Node.js
	fmt.Print("🔍 Checking Node.js version... ")
	cmd := exec.Command("node", "--version")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("❌")
		return fmt.Errorf("node not found")
	}
	fmt.Printf("✅ %s", strings.TrimSpace(string(output)))

	fmt.Println("✅ Prerequisites met")
	return nil
}

func createExampleConfig() (string, error) {
	// Create a filesystem MCP server config
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"filesystem": map[string]interface{}{
				"command": "npx",
				"args":    []string{"-y", "@modelcontextprotocol/server-filesystem", "."},
			},
		},
	}

	// Write to temp file
	tmpDir := os.TempDir()
	configPath := filepath.Join(tmpDir, "claude-mcp-demo.json")

	f, err := os.Create(configPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return "", err
	}

	return configPath, nil
}

func handleCommand(cmd string) bool {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}

	switch strings.ToLower(parts[0]) {
	case "/config":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /config <path>")
			return true
		}
		path := strings.Join(parts[1:], " ")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("✗ Config file not found: %s\n", path)
			return true
		}
		configFile = path
		fmt.Printf("✓ Loaded MCP config: %s\n", path)
		return true

	case "/example":
		// Check prerequisites but don't block on failure
		fmt.Println("\n📝 Creating MCP configuration...")
		if err := checkPrerequisites(); err != nil {
			fmt.Println("⚠️  Note: Prerequisites not met (for actual MCP server usage):")
			fmt.Printf("   %v\n", err)
			fmt.Println("   (Creating config anyway for demonstration)")
		} else {
			fmt.Println("✅ Prerequisites verified")
		}

		// Always create the config, regardless of prerequisites
		path, err := createExampleConfig()
		if err != nil {
			fmt.Printf("✗ Failed to create example config: %v\n", err)
			return true
		}

		configFile = path
		allowedTools = []string{
			"mcp__filesystem__list_directory",
			"mcp__filesystem__read_file",
		}
		fmt.Printf("✓ Created example config at: %s\n", path)
		fmt.Println("  📦 Server: filesystem (via npx)")
		fmt.Println("  🔧 Command: npx -y @modelcontextprotocol/server-filesystem .")
		fmt.Println("  🛠️  Configured tools:")
		fmt.Println("     • mcp__filesystem__list_directory")
		fmt.Println("     • mcp__filesystem__read_file")
		fmt.Println("\n💡 Try: 'List files in the current directory'")
		return true

	case "/strict":
		if len(parts) < 2 {
			strictMode = !strictMode
		} else {
			switch strings.ToLower(parts[1]) {
			case "on", "true", "yes", "1":
				strictMode = true
			case "off", "false", "no", "0":
				strictMode = false
			default:
				fmt.Println("✗ Usage: /strict on|off")
				return true
			}
		}
		fmt.Printf("✓ Strict mode: %v\n", strictMode)
		if strictMode {
			fmt.Println("  (Only MCP servers from config will be used)")
		}
		return true

	case "/tools":
		fmt.Println("\n🔧 Allowed MCP Tools:")
		if len(allowedTools) == 0 {
			fmt.Println("  ⚠️  No allowlist configured - all tools are permitted")
			fmt.Println("\n💡 Tip: Use /allow <tool> to restrict to specific tools")
		} else {
			for i, tool := range allowedTools {
				// Parse tool name for better display
				if strings.HasPrefix(tool, "mcp__") {
					serverName := extractServerName(tool)
					toolName := strings.TrimPrefix(tool, "mcp__"+serverName+"__")
					fmt.Printf("  %d. 🔌 %s (server: %s)\n", i+1, toolName, serverName)
				} else {
					fmt.Printf("  %d. 🛠️  %s (built-in)\n", i+1, tool)
				}
			}
		}
		return true

	case "/allow":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /allow <tool_name>")
			return true
		}
		tool := parts[1]
		allowedTools = append(allowedTools, tool)
		fmt.Printf("✓ Added to allowlist: %s\n", tool)
		return true

	case "/clear":
		configFile = ""
		strictMode = false
		allowedTools = nil
		fmt.Println("✓ Cleared MCP configuration")
		return true

	case "/status":
		displayMCPStatus()
		return true

	case "/help":
		displayHelp()
		return true
	}
	return false
}

func extractServerName(toolName string) string {
	// MCP tool names follow pattern: mcp__<server>__<tool>
	// Example: mcp__filesystem__list_directory -> filesystem
	parts := strings.Split(toolName, "__")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "unknown"
}

func displayStreamingMessage(msg claude.Message) {
	switch msg.Type {
	case "system":
		if msg.Subtype == "init" {
			fmt.Printf("🔄 Session: %s\n", msg.SessionID[:8])
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
								// Highlight MCP tools with server info
								if strings.HasPrefix(name, "mcp__") {
									serverName := extractServerName(name)
									toolName := strings.TrimPrefix(name, "mcp__"+serverName+"__")
									fmt.Printf("🔌 MCP Tool: %s (server: %s)\n", toolName, serverName)
								} else {
									fmt.Printf("🛠️  SDK Tool: %s (built-in)\n", name)
								}
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
			fmt.Printf("📊 Cost: $%.6f | Duration: %.1fs | Turns: %d\n",
				msg.CostUSD, float64(msg.DurationMS)/1000.0, msg.NumTurns)
		}
	}
}

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║       Claude Code Go SDK - MCP Integration Demo            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This demo showcases Model Context Protocol (MCP) integration:")
	fmt.Println("  • Load MCP server configurations")
	fmt.Println("  • Strict mode for isolated MCP environments")
	fmt.Println("  • Tool allowlisting for security")
	fmt.Println("  • Real-time MCP tool execution visibility")
	fmt.Println()
	fmt.Println("💡 Use /example to set up a demo filesystem MCP server")
	fmt.Println("   (Requires: npx and @modelcontextprotocol/server-filesystem)")
	fmt.Println()
	displayHelp()
	displayMCPStatus()

	cc := claude.NewClient("claude")
	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)

	var sessionID string

	for {
		fmt.Print("\n>>> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if isExitCommand(input) {
			break
		}
		if strings.HasPrefix(input, "/") {
			if handleCommand(input) {
				continue
			}
		}

		// Build options with MCP configuration
		opts := &claude.RunOptions{
			Format:       claude.StreamJSONOutput,
			SystemPrompt: "You are a helpful assistant with access to MCP tools. Use them when appropriate.",
		}

		// Add MCP configuration
		if configFile != "" {
			opts.MCPConfigs = []string{configFile}
		}
		if strictMode {
			opts.StrictMCPConfig = true
		}
		if len(allowedTools) > 0 {
			opts.AllowedTools = allowedTools
		} else {
			// Default tools if no MCP config
			opts.AllowedTools = []string{"Read(*)", "Bash(ls*)", "Bash(pwd)"}
		}

		// Resume session if we have one
		if sessionID != "" {
			opts.ResumeID = sessionID
		}

		messageCh, errCh := cc.StreamPrompt(ctx, input, opts)

		// Process streaming messages
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

		displayMCPStatus()
	}

	// Cleanup temp config if we created one
	if strings.Contains(configFile, "claude-mcp-demo.json") {
		os.Remove(configFile)
	}

	// Final summary
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     MCP Demo Summary                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("   Config used:    %s\n", truncatePath(configFile))
	fmt.Printf("   Strict mode:    %v\n", strictMode)
	fmt.Printf("   Tools allowed:  %d\n", len(allowedTools))
	fmt.Println("\nDemo completed!")
}

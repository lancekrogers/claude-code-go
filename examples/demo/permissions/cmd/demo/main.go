package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

var (
	permissionMode   claude.PermissionMode
	allowedTools     []string
	disallowedTools  []string
	callbackEnabled  bool
	callbackDecision string // "allow", "deny", or "ask"
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

func displayPermissionStatus() {
	fmt.Println("\n┌─────────────────────────────────────────────────────────────┐")
	fmt.Printf("│ Mode: %-12s | Callback: %-6v | Decision: %-10s │\n",
		permissionMode, callbackEnabled, callbackDecision)
	fmt.Printf("│ Allowed: %-9d | Disallowed: %-35d │\n",
		len(allowedTools), len(disallowedTools))
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
}

func displayHelp() {
	fmt.Println("\n📚 Permission Commands:")
	fmt.Println("  /mode default     - Standard permission checks")
	fmt.Println("  /mode acceptEdits - Auto-approve file edits")
	fmt.Println("  /mode bypass      - Bypass all checks (dangerous!)")
	fmt.Println("  /allow <pattern>  - Add to allowed tools list")
	fmt.Println("  /deny <pattern>   - Add to disallowed tools list")
	fmt.Println("  /clear allowed    - Clear allowed tools list")
	fmt.Println("  /clear denied     - Clear disallowed tools list")
	fmt.Println("  /callback on|off  - Enable/disable permission callback")
	fmt.Println("  /decision <d>     - Set callback decision (allow/deny/ask)")
	fmt.Println("  /tools            - Show current tool permissions")
	fmt.Println("  /presets          - Show preset tool configurations")
	fmt.Println("  /preset <name>    - Apply a preset configuration")
	fmt.Println("  /status           - Show current permission status")
	fmt.Println("  /help             - Show this help")
	fmt.Println("  exit              - Exit the demo")
}

func displayPresets() {
	fmt.Println("\n🔧 Available Presets:")
	fmt.Println("  readonly  - Only read operations (Read, Bash(ls/cat/pwd))")
	fmt.Println("  safe      - Safe operations (no writes, no network)")
	fmt.Println("  git       - Git operations only")
	fmt.Println("  full      - All standard tools")
	fmt.Println("  minimal   - Absolute minimum (Read only)")
}

func applyPreset(name string) bool {
	switch strings.ToLower(name) {
	case "readonly":
		allowedTools = []string{
			"Read(*)",
			"Bash(ls*)",
			"Bash(cat*)",
			"Bash(pwd)",
			"Bash(head*)",
			"Bash(tail*)",
		}
		disallowedTools = []string{
			"Write(*)",
			"Edit(*)",
			"Bash(rm*)",
			"Bash(mv*)",
		}
		fmt.Println("✓ Applied 'readonly' preset")
		return true

	case "safe":
		allowedTools = []string{
			"Read(*)",
			"Bash(ls*)",
			"Bash(pwd)",
			"Bash(echo*)",
			"Bash(date)",
		}
		disallowedTools = []string{
			"Write(*)",
			"Edit(*)",
			"Bash(curl*)",
			"Bash(wget*)",
			"Bash(rm*)",
		}
		fmt.Println("✓ Applied 'safe' preset")
		return true

	case "git":
		allowedTools = []string{
			"Bash(git status*)",
			"Bash(git log*)",
			"Bash(git diff*)",
			"Bash(git branch*)",
			"Read(*)",
		}
		disallowedTools = []string{
			"Bash(git push*)",
			"Bash(git reset --hard*)",
			"Bash(git force*)",
		}
		fmt.Println("✓ Applied 'git' preset")
		return true

	case "full":
		allowedTools = []string{
			"Read(*)",
			"Write(*)",
			"Edit(*)",
			"Bash(*)",
			"Glob(*)",
			"Grep(*)",
		}
		disallowedTools = nil
		fmt.Println("✓ Applied 'full' preset")
		return true

	case "minimal":
		allowedTools = []string{"Read(*)"}
		disallowedTools = []string{"Bash(*)", "Write(*)", "Edit(*)"}
		fmt.Println("✓ Applied 'minimal' preset")
		return true

	default:
		fmt.Printf("✗ Unknown preset: %s\n", name)
		return false
	}
}

func handleCommand(cmd string) bool {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}

	switch strings.ToLower(parts[0]) {
	case "/mode":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /mode default|acceptEdits|bypass")
			return true
		}
		switch strings.ToLower(parts[1]) {
		case "default":
			permissionMode = claude.PermissionModeDefault
			fmt.Println("✓ Mode: default (standard permission checks)")
		case "acceptedits":
			permissionMode = claude.PermissionModeAcceptEdits
			fmt.Println("✓ Mode: acceptEdits (auto-approve file edits)")
		case "bypass":
			permissionMode = claude.PermissionModeBypassPermissions
			fmt.Println("⚠️  Mode: bypass (ALL permissions bypassed - DANGEROUS!)")
		default:
			fmt.Println("✗ Unknown mode. Use: default, acceptEdits, or bypass")
		}
		return true

	case "/allow":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /allow <pattern>")
			fmt.Println("   Examples: /allow Read(*)")
			fmt.Println("             /allow Bash(git*)")
			return true
		}
		pattern := strings.Join(parts[1:], " ")
		allowedTools = append(allowedTools, pattern)
		fmt.Printf("✓ Added to allowed: %s\n", pattern)
		return true

	case "/deny":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /deny <pattern>")
			return true
		}
		pattern := strings.Join(parts[1:], " ")
		disallowedTools = append(disallowedTools, pattern)
		fmt.Printf("✓ Added to disallowed: %s\n", pattern)
		return true

	case "/clear":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /clear allowed|denied")
			return true
		}
		switch strings.ToLower(parts[1]) {
		case "allowed":
			allowedTools = nil
			fmt.Println("✓ Cleared allowed tools list")
		case "denied", "disallowed":
			disallowedTools = nil
			fmt.Println("✓ Cleared disallowed tools list")
		default:
			fmt.Println("✗ Usage: /clear allowed|denied")
		}
		return true

	case "/callback":
		if len(parts) < 2 {
			callbackEnabled = !callbackEnabled
		} else {
			switch strings.ToLower(parts[1]) {
			case "on", "true", "yes", "1":
				callbackEnabled = true
			case "off", "false", "no", "0":
				callbackEnabled = false
			default:
				fmt.Println("✗ Usage: /callback on|off")
				return true
			}
		}
		fmt.Printf("✓ Permission callback: %v\n", callbackEnabled)
		return true

	case "/decision":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /decision allow|deny|ask")
			return true
		}
		switch strings.ToLower(parts[1]) {
		case "allow":
			callbackDecision = "allow"
			fmt.Println("✓ Callback will auto-allow all requests")
		case "deny":
			callbackDecision = "deny"
			fmt.Println("✓ Callback will auto-deny all requests")
		case "ask":
			callbackDecision = "ask"
			fmt.Println("✓ Callback will prompt for each request")
		default:
			fmt.Println("✗ Unknown decision. Use: allow, deny, or ask")
		}
		return true

	case "/tools":
		fmt.Println("\n🔧 Tool Permissions:")
		fmt.Println("  Allowed:")
		if len(allowedTools) == 0 {
			fmt.Println("    (none specified - all allowed)")
		}
		for _, t := range allowedTools {
			fmt.Printf("    ✓ %s\n", t)
		}
		fmt.Println("  Disallowed:")
		if len(disallowedTools) == 0 {
			fmt.Println("    (none specified)")
		}
		for _, t := range disallowedTools {
			fmt.Printf("    ✗ %s\n", t)
		}
		return true

	case "/presets":
		displayPresets()
		return true

	case "/preset":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /preset <name>")
			displayPresets()
			return true
		}
		applyPreset(parts[1])
		return true

	case "/status":
		displayPermissionStatus()
		return true

	case "/help":
		displayHelp()
		return true
	}
	return false
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
								fmt.Printf("🔧 Tool requested: %s\n", name)
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
	fmt.Println("║       Claude Code Go SDK - Permission Control Demo         ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This demo showcases permission and tool control features:")
	fmt.Println("  • Permission modes (default, acceptEdits, bypass)")
	fmt.Println("  • Tool allowlisting with glob patterns")
	fmt.Println("  • Tool blocklisting for security")
	fmt.Println("  • Preset configurations for common use cases")
	fmt.Println()

	// Initialize defaults
	permissionMode = claude.PermissionModeDefault
	callbackEnabled = false
	callbackDecision = "allow"

	// Start with readonly preset
	applyPreset("readonly")

	displayHelp()
	displayPermissionStatus()

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

		// Build options with permission configuration
		opts := &claude.RunOptions{
			Format:         claude.StreamJSONOutput,
			SystemPrompt:   "You are a helpful assistant. Keep responses concise.",
			MaxTurns:       3,
			PermissionMode: permissionMode,
		}

		// Set tool permissions
		if len(allowedTools) > 0 {
			opts.AllowedTools = allowedTools
		}
		if len(disallowedTools) > 0 {
			opts.DisallowedTools = disallowedTools
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

		displayPermissionStatus()
	}

	// Final summary
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                Permission Demo Summary                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("   Permission Mode:  %s\n", permissionMode)
	fmt.Printf("   Allowed Tools:    %d patterns\n", len(allowedTools))
	fmt.Printf("   Disallowed Tools: %d patterns\n", len(disallowedTools))
	fmt.Printf("   Callback Enabled: %v\n", callbackEnabled)
	fmt.Println("\nDemo completed!")
}

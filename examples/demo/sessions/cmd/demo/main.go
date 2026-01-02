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

// Session modes for the demo
const (
	ModeNormal    = "normal"
	ModeCustomID  = "custom"
	ModeFork      = "fork"
	ModeEphemeral = "ephemeral"
)

var (
	currentMode      = ModeNormal
	currentSessionID string
	sessionHistory   []string
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

func displaySessionInfo() {
	fmt.Println("\n┌─────────────────────────────────────────────────────────────┐")
	fmt.Printf("│ Mode: %-10s | Session: %-30s │\n", currentMode, truncateID(currentSessionID))
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
}

func truncateID(id string) string {
	if len(id) > 30 {
		return id[:12] + "..." + id[len(id)-12:]
	}
	if id == "" {
		return "(new)"
	}
	return id
}

func displayHelp() {
	fmt.Println("\n📚 Session Commands:")
	fmt.Println("  /normal    - Switch to normal session mode (default)")
	fmt.Println("  /custom    - Use a custom-generated session ID")
	fmt.Println("  /fork      - Fork from current session into a new one")
	fmt.Println("  /ephemeral - Use ephemeral sessions (no disk persistence)")
	fmt.Println("  /new       - Start a brand new session")
	fmt.Println("  /history   - Show session history")
	fmt.Println("  /status    - Show current session status")
	fmt.Println("  /help      - Show this help")
	fmt.Println("  exit       - Exit the demo")
}

func handleCommand(cmd string) bool {
	switch strings.ToLower(strings.TrimSpace(cmd)) {
	case "/normal":
		currentMode = ModeNormal
		fmt.Println("✓ Switched to normal session mode")
		return true
	case "/custom":
		currentMode = ModeCustomID
		newID := claude.GenerateSessionID()
		currentSessionID = newID
		sessionHistory = append(sessionHistory, newID)
		fmt.Printf("✓ Switched to custom ID mode. Generated ID: %s\n", truncateID(newID))
		return true
	case "/fork":
		if currentSessionID == "" {
			fmt.Println("✗ No current session to fork from")
			return true
		}
		currentMode = ModeFork
		fmt.Printf("✓ Next message will fork from session: %s\n", truncateID(currentSessionID))
		return true
	case "/ephemeral":
		currentMode = ModeEphemeral
		currentSessionID = "" // Clear any existing session
		fmt.Println("✓ Switched to ephemeral mode (sessions not saved to disk)")
		return true
	case "/new":
		currentSessionID = ""
		currentMode = ModeNormal
		fmt.Println("✓ Starting fresh session")
		return true
	case "/history":
		fmt.Println("\n📜 Session History:")
		if len(sessionHistory) == 0 {
			fmt.Println("  (no sessions yet)")
		}
		for i, sid := range sessionHistory {
			marker := "  "
			if sid == currentSessionID {
				marker = "→ "
			}
			fmt.Printf("%s%d. %s\n", marker, i+1, truncateID(sid))
		}
		return true
	case "/status":
		displaySessionInfo()
		fmt.Printf("  Sessions created: %d\n", len(sessionHistory))
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
			fmt.Printf("🔄 Session initialized: %s\n", truncateID(msg.SessionID))
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
								fmt.Printf("🔧 Using: %s\n", name)
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
	fmt.Println("║       Claude Code Go SDK - Session Management Demo         ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This demo showcases the SDK's session management features:")
	fmt.Println("  • Custom session IDs with UUID generation")
	fmt.Println("  • Session forking (branch conversations)")
	fmt.Println("  • Ephemeral sessions (no disk persistence)")
	fmt.Println("  • Session resumption and history tracking")
	fmt.Println()
	displayHelp()
	displaySessionInfo()

	cc := claude.NewClient("claude")
	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)

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

		// Build options based on current mode
		opts := &claude.RunOptions{
			Format:       claude.StreamJSONOutput,
			SystemPrompt: "You are a helpful assistant. Keep responses concise.",
			AllowedTools: []string{"Read(*)", "Bash(ls*)", "Bash(pwd)"},
			MaxTurns:     3,
		}

		switch currentMode {
		case ModeCustomID:
			if currentSessionID != "" {
				opts.SessionID = currentSessionID
			}
		case ModeFork:
			if currentSessionID != "" {
				opts.ResumeID = currentSessionID
				opts.ForkSession = true
			}
			// Reset to normal after forking
			currentMode = ModeNormal
		case ModeEphemeral:
			opts.NoSessionPersistence = true
		case ModeNormal:
			if currentSessionID != "" {
				opts.ResumeID = currentSessionID
			}
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
				// Track session ID
				if msg.SessionID != "" && msg.SessionID != currentSessionID {
					currentSessionID = msg.SessionID
					// Add to history if new
					found := false
					for _, s := range sessionHistory {
						if s == msg.SessionID {
							found = true
							break
						}
					}
					if !found {
						sessionHistory = append(sessionHistory, msg.SessionID)
					}
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

		displaySessionInfo()
	}

	// Final summary
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     Session Summary                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("   Sessions created: %d\n", len(sessionHistory))
	fmt.Printf("   Current session:  %s\n", truncateID(currentSessionID))
	fmt.Printf("   Final mode:       %s\n", currentMode)
	fmt.Println("\nDemo completed!")
}

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
	agentManager *claude.SubagentManager
	currentAgent string
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

func displayAgents() {
	fmt.Println("\n🤖 Available Agents:")
	fmt.Println("   ┌─────────────────────────────────────────────────────────────┐")
	for name, desc := range agentManager.GetAgentDescriptions() {
		marker := "  "
		if name == currentAgent {
			marker = "→ "
		}
		// Truncate description to fit
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		fmt.Printf("   │ %s%-12s │ %-44s │\n", marker, name, desc)
	}
	fmt.Println("   └─────────────────────────────────────────────────────────────┘")
}

func displayStreamingMessage(msg claude.Message) {
	switch msg.Type {
	case "system":
		if msg.Subtype == "init" {
			fmt.Printf("🔄 Agent session: %s\n", msg.SessionID[:8])
		}
	case "assistant":
		var content map[string]interface{}
		if err := json.Unmarshal(msg.Message, &content); err == nil {
			if contentArray, ok := content["content"].([]interface{}); ok {
				for _, item := range contentArray {
					if itemMap, ok := item.(map[string]interface{}); ok {
						if itemMap["type"] == "text" {
							if text, ok := itemMap["text"].(string); ok && strings.TrimSpace(text) != "" {
								fmt.Printf("💬 [%s]: %s\n", currentAgent, text)
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
			fmt.Printf("✅ [%s] Complete - Cost: $%.6f | Turns: %d\n",
				currentAgent, msg.CostUSD, msg.NumTurns)
			// Store session for potential resumption
			agentManager.SetSession(currentAgent, msg.SessionID)
		}
	}
}

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║       Claude Code Go SDK - Subagent Orchestration Demo     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	cc := claude.NewClient("claude")
	agentManager = claude.NewSubagentManager(cc)

	// Register pre-built specialized agents
	agents := map[string]*claude.SubagentConfig{
		"security":    claude.SecurityReviewerAgent(),
		"code-review": claude.CodeReviewerAgent(),
		"testing":     claude.TestAnalystAgent(),
		"performance": claude.PerformanceAnalystAgent(),
		"docs":        claude.DocumentationAgent(),
	}

	for name, config := range agents {
		if err := agentManager.RegisterAgent(name, config); err != nil {
			fmt.Printf("❌ Failed to register %s: %v\n", name, err)
			return
		}
	}

	currentAgent = "code-review" // Default agent

	fmt.Printf("📦 Registered %d specialized agents\n", agentManager.AgentCount())
	displayAgents()
	fmt.Println()

	fmt.Println("🚀 Commands:")
	fmt.Println("   • @<agent> <prompt>  - Switch agent and send prompt")
	fmt.Println("   • agents             - List available agents")
	fmt.Println("   • resume             - Continue previous conversation")
	fmt.Println("   • exit               - End session")
	fmt.Println()
	fmt.Printf("📍 Current agent: %s\n", currentAgent)
	fmt.Println()

	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf("[%s] >>> ", currentAgent)
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" || isExitCommand(input) {
			break
		}

		// Handle special commands
		if input == "agents" {
			displayAgents()
			continue
		}

		if input == "resume" {
			if sessionID, ok := agentManager.GetSession(currentAgent); ok {
				fmt.Printf("📂 Resuming %s session: %s\n", currentAgent, sessionID[:8])
				fmt.Print("Continue with: ")
				if !scanner.Scan() {
					break
				}
				prompt := strings.TrimSpace(scanner.Text())
				if prompt == "" {
					continue
				}
				result, err := agentManager.ResumeAgent(ctx, currentAgent, prompt, nil)
				if err != nil {
					fmt.Printf("❌ Resume error: %v\n", err)
				} else {
					fmt.Printf("💬 [%s]: %s\n", currentAgent, result.Result)
					fmt.Printf("✅ Cost: $%.6f\n", result.CostUSD)
				}
			} else {
				fmt.Printf("⚠️  No previous session for %s\n", currentAgent)
			}
			continue
		}

		// Check for agent switch: @agent-name prompt
		if strings.HasPrefix(input, "@") {
			parts := strings.SplitN(input[1:], " ", 2)
			agentName := parts[0]
			if _, ok := agentManager.GetAgent(agentName); ok {
				currentAgent = agentName
				fmt.Printf("🔄 Switched to: %s\n", currentAgent)
				if len(parts) > 1 {
					input = parts[1]
				} else {
					continue
				}
			} else {
				fmt.Printf("⚠️  Unknown agent: %s\n", agentName)
				displayAgents()
				continue
			}
		}

		// Run the current agent with streaming
		messageCh, errCh := agentManager.StreamAgent(ctx, currentAgent, input, nil)

	processLoop:
		for {
			select {
			case msg, ok := <-messageCh:
				if !ok {
					break processLoop
				}
				displayStreamingMessage(msg)
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
		fmt.Println()
	}

	// Final summary
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     Session Summary                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("   Agents used: %d\n", agentManager.AgentCount())
	fmt.Println("   Sessions stored:")
	for _, name := range agentManager.ListAgents() {
		if sessionID, ok := agentManager.GetSession(name); ok {
			fmt.Printf("     • %s: %s\n", name, sessionID[:8])
		}
	}
	fmt.Println("\nDemo completed!")
}

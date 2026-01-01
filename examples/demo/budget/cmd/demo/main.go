package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

var budgetTracker *claude.BudgetTracker

func isExitCommand(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	exitCommands := []string{
		"exit", "quit", "bye", "q", ":q", "done", "stop",
	}
	for _, cmd := range exitCommands {
		if input == cmd {
			return true
		}
	}
	return false
}

func displayBudgetStatus() {
	spent := budgetTracker.TotalSpent()
	remaining := budgetTracker.RemainingBudget()
	config := budgetTracker.Config()

	pct := 0.0
	if config.MaxBudgetUSD > 0 {
		pct = (spent / config.MaxBudgetUSD) * 100
	}

	bar := renderProgressBar(pct, 20)
	fmt.Printf("💰 Budget: $%.4f / $%.2f [%s] %.1f%% | Remaining: $%.4f\n",
		spent, config.MaxBudgetUSD, bar, pct, remaining)
}

func renderProgressBar(pct float64, width int) string {
	filled := int(pct / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func displayStreamingMessage(msg claude.Message) {
	switch msg.Type {
	case "system":
		if msg.Subtype == "init" {
			fmt.Printf("🔄 Session %s started\n", msg.SessionID[:8])
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
			// Track the cost
			budgetTracker.AddSpend(msg.SessionID, msg.CostUSD)
			fmt.Printf("📊 Cost: $%.6f | Duration: %.1fs | Turns: %d\n",
				msg.CostUSD, float64(msg.DurationMS)/1000.0, msg.NumTurns)
			displayBudgetStatus()
		}
	}
}

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║       Claude Code Go SDK - Budget Tracking Demo            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Initialize budget tracker with callbacks
	budgetTracker = claude.NewBudgetTracker(&claude.BudgetConfig{
		MaxBudgetUSD:     0.50, // $0.50 limit for demo
		WarningThreshold: 0.5,  // Warn at 50%
		OnBudgetWarning: func(current, max float64) {
			pct := (current / max) * 100
			fmt.Printf("\n⚠️  WARNING: Budget at %.1f%% ($%.4f / $%.2f)\n", pct, current, max)
		},
		OnBudgetExceeded: func(current, max float64) {
			fmt.Printf("\n🚨 BUDGET EXCEEDED: $%.4f > $%.2f limit!\n", current, max)
		},
	})

	fmt.Println("📋 Budget Configuration:")
	fmt.Printf("   • Maximum budget: $%.2f\n", 0.50)
	fmt.Printf("   • Warning threshold: 50%%\n")
	fmt.Println()
	displayBudgetStatus()
	fmt.Println()

	cc := claude.NewClient("claude")
	ctx := context.Background()

	// Initial prompt
	fmt.Println("🚀 Starting budget-tracked conversation...")
	fmt.Println("   Ask Claude to do tasks and watch your budget in real-time!")
	fmt.Println()

	opts := &claude.RunOptions{
		Format:       claude.StreamJSONOutput,
		SystemPrompt: "You are a helpful assistant. Keep responses concise to minimize costs.",
		AllowedTools: []string{
			"Read(*)",
			"Bash(ls*)",
			"Bash(pwd)",
			"Bash(echo*)",
		},
		MaxTurns: 3,
	}

	var sessionID string
	scanner := bufio.NewScanner(os.Stdin)

	for {
		// Check budget before accepting input
		if !budgetTracker.CanSpend(0.05) { // Estimate ~$0.05 per request
			fmt.Println("\n🛑 Budget exhausted! Cannot process more requests.")
			fmt.Printf("   Total spent: $%.4f\n", budgetTracker.TotalSpent())
			break
		}

		fmt.Print("\n>>> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" || isExitCommand(input) {
			break
		}

		// Continue or start conversation
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
					if err == claude.ErrBudgetExceeded {
						fmt.Println("\n🛑 Request blocked: Would exceed budget!")
					} else {
						log.Printf("Error: %v", err)
					}
					break processLoop
				}
			}
		}
	}

	// Final summary
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     Session Summary                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("   Total spent: $%.6f\n", budgetTracker.TotalSpent())
	fmt.Printf("   Remaining:   $%.6f\n", budgetTracker.RemainingBudget())
	displayBudgetStatus()
	fmt.Println("\nDemo completed!")
}

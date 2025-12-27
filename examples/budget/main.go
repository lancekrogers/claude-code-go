package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

func main() {
	client := claude.NewClient("claude")

	// Example 1: Basic budget tracking
	fmt.Println("Example 1: Basic Budget Tracking")
	fmt.Println("================================")

	tracker := claude.NewBudgetTracker(&claude.BudgetConfig{
		MaxBudgetUSD:     1.00, // $1.00 limit
		WarningThreshold: 0.5,  // Warn at 50%
		OnBudgetWarning: func(current, max float64) {
			log.Printf("WARNING: Budget at %.1f%% ($%.4f / $%.2f)",
				(current/max)*100, current, max)
		},
		OnBudgetExceeded: func(current, max float64) {
			log.Printf("EXCEEDED: Budget exceeded! $%.4f > $%.2f", current, max)
		},
	})

	opts := &claude.RunOptions{
		Format:        claude.JSONOutput,
		BudgetTracker: tracker,
		MaxTurns:      1,
	}

	// Run a few prompts and track spending
	prompts := []string{
		"What is 2+2?",
		"What is the capital of France?",
		"Name three colors",
	}

	for i, prompt := range prompts {
		fmt.Printf("\nPrompt %d: %s\n", i+1, prompt)

		// Check if we can afford another request (estimate ~$0.01-0.10 per request)
		if !tracker.CanSpend(0.10) {
			fmt.Println("Stopping: Would exceed budget")
			break
		}

		result, err := client.RunPrompt(prompt, opts)
		if err != nil {
			if err == claude.ErrBudgetExceeded {
				fmt.Println("Budget exceeded, stopping")
				break
			}
			log.Printf("Error: %v", err)
			continue
		}

		// Manually track spending (in production, integrate into execution)
		tracker.AddSpend(result.SessionID, result.CostUSD)

		fmt.Printf("Cost: $%.6f | Total: $%.6f | Remaining: $%.6f\n",
			result.CostUSD, tracker.TotalSpent(), tracker.RemainingBudget())
	}

	// Example 2: Multi-session tracking
	fmt.Println("\n\nExample 2: Multi-Session Budget Tracking")
	fmt.Println("=========================================")

	multiTracker := claude.NewBudgetTracker(&claude.BudgetConfig{
		MaxBudgetUSD: 5.00,
	})

	// Simulate multiple sessions
	sessions := []struct {
		id   string
		cost float64
	}{
		{"session-1", 0.50},
		{"session-2", 1.25},
		{"session-3", 0.75},
	}

	for _, s := range sessions {
		if err := multiTracker.AddSpend(s.id, s.cost); err != nil {
			fmt.Printf("Session %s: Budget error - %v\n", s.id, err)
		} else {
			fmt.Printf("Session %s: Spent $%.2f | Session total: $%.2f | Overall: $%.2f\n",
				s.id, s.cost, multiTracker.SessionSpent(s.id), multiTracker.TotalSpent())
		}
	}

	// Example 3: Budget with context timeout
	fmt.Println("\n\nExample 3: Budget with Context")
	fmt.Println("===============================")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	budgetOpts := &claude.RunOptions{
		Format:        claude.JSONOutput,
		BudgetTracker: claude.NewBudgetTracker(&claude.BudgetConfig{MaxBudgetUSD: 0.50}),
		Timeout:       10 * time.Second,
	}

	result, err := client.RunPromptCtx(ctx, "Say hello briefly", budgetOpts)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Result: %s\n", result.Result)
		fmt.Printf("Cost: $%.6f\n", result.CostUSD)
	}

	fmt.Println("\nBudget tracking examples complete!")
}

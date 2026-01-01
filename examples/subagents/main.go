package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

func main() {
	cc := claude.NewClient("claude")

	// Example 1: Using Pre-built Subagent Templates
	fmt.Println("Example 1: Pre-built Subagent Templates")
	fmt.Println("========================================")

	manager := claude.NewSubagentManager(cc)

	// Register pre-built specialized agents
	if err := manager.RegisterAgent("security", claude.SecurityReviewerAgent()); err != nil {
		log.Fatalf("Failed to register security agent: %v", err)
	}
	if err := manager.RegisterAgent("code-review", claude.CodeReviewerAgent()); err != nil {
		log.Fatalf("Failed to register code-review agent: %v", err)
	}
	if err := manager.RegisterAgent("testing", claude.TestAnalystAgent()); err != nil {
		log.Fatalf("Failed to register testing agent: %v", err)
	}
	if err := manager.RegisterAgent("performance", claude.PerformanceAnalystAgent()); err != nil {
		log.Fatalf("Failed to register performance agent: %v", err)
	}
	if err := manager.RegisterAgent("docs", claude.DocumentationAgent()); err != nil {
		log.Fatalf("Failed to register docs agent: %v", err)
	}

	fmt.Printf("Registered %d agents: %v\n", manager.AgentCount(), manager.ListAgents())

	// Show agent descriptions (useful for routing decisions)
	fmt.Println("\nAgent Descriptions:")
	for name, desc := range manager.GetAgentDescriptions() {
		fmt.Printf("  - %s: %s\n", name, desc[:min(80, len(desc))]+"...")
	}

	// Example 2: Custom Subagent Configuration
	fmt.Println("\n\nExample 2: Custom Subagent Configuration")
	fmt.Println("=========================================")

	customAgent := &claude.SubagentConfig{
		Description: "Go language expert for idiomatic Go code review and guidance.",
		Prompt: `You are a Go language expert with deep knowledge of Go idioms,
best practices, and the standard library. Focus on:
- Idiomatic Go patterns (error handling, interfaces, goroutines)
- Standard library usage over third-party dependencies
- Effective use of context for cancellation and timeouts
- Proper error wrapping with fmt.Errorf and %w
- Struct embedding and composition over inheritance

Provide practical examples and explain the "Go way" of solving problems.`,
		Tools:    []string{"Read", "Grep", "Glob"},
		Model:    "sonnet",
		MaxTurns: 5,
	}

	if err := manager.RegisterAgent("go-expert", customAgent); err != nil {
		log.Fatalf("Failed to register go-expert agent: %v", err)
	}

	fmt.Printf("Total agents after custom: %d\n", manager.AgentCount())

	// Example 3: Running a Subagent
	fmt.Println("\n\nExample 3: Running a Subagent")
	fmt.Println("==============================")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run the code review agent on a simple prompt
	result, err := manager.RunAgent(ctx, "code-review", "What are the key principles of clean code?", nil)
	if err != nil {
		log.Printf("Error running code-review agent: %v", err)
	} else {
		fmt.Printf("Code Review Agent Response:\n%s\n", result.Result)
		fmt.Printf("Cost: $%.6f | Turns: %d\n", result.CostUSD, result.NumTurns)

		// Store session for potential continuation
		manager.SetSession("code-review", result.SessionID)
	}

	// Example 4: Session Management (Conversation Continuity)
	fmt.Println("\n\nExample 4: Session Management")
	fmt.Println("==============================")

	// Check for existing session
	if sessionID, ok := manager.GetSession("code-review"); ok {
		fmt.Printf("Found existing session for code-review: %s\n", sessionID[:20]+"...")

		// Resume the conversation
		result, err := manager.ResumeAgent(ctx, "code-review",
			"Can you give me a specific example of that?", nil)
		if err != nil {
			log.Printf("Error resuming agent: %v", err)
		} else {
			fmt.Printf("Continued Response:\n%s\n", result.Result)
		}
	}

	// Example 5: Streaming Agent Output
	fmt.Println("\n\nExample 5: Streaming Agent Output")
	fmt.Println("==================================")

	streamCtx, streamCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer streamCancel()

	msgCh, errCh := manager.StreamAgent(streamCtx, "go-expert",
		"Explain error handling best practices in Go briefly", nil)

	fmt.Println("Streaming response:")
	for msg := range msgCh {
		switch msg.Type {
		case "assistant":
			// Assistant messages contain the raw message content
			if len(msg.Message) > 0 {
				fmt.Printf("[assistant] %s\n", string(msg.Message))
			}
		case "result":
			fmt.Printf("\n[Complete] Result: %s\n", msg.Result)
			fmt.Printf("[Complete] Cost: $%.6f\n", msg.CostUSD)
		}
	}

	// Check for errors
	if err := <-errCh; err != nil {
		log.Printf("Stream error: %v", err)
	}

	// Example 6: Batch Agent Registration
	fmt.Println("\n\nExample 6: Batch Agent Registration")
	fmt.Println("====================================")

	manager2 := claude.NewSubagentManager(cc)

	agents := map[string]*claude.SubagentConfig{
		"python-expert": {
			Description: "Python language expert for pythonic code patterns.",
			Prompt:      "You are a Python expert. Focus on PEP 8, type hints, and pythonic idioms.",
			Tools:       []string{"Read", "Grep", "Glob"},
			Model:       "haiku",
		},
		"rust-expert": {
			Description: "Rust language expert for memory safety and ownership patterns.",
			Prompt:      "You are a Rust expert. Focus on ownership, borrowing, lifetimes, and zero-cost abstractions.",
			Tools:       []string{"Read", "Grep", "Glob"},
			Model:       "sonnet",
		},
		"typescript-expert": {
			Description: "TypeScript expert for type-safe JavaScript development.",
			Prompt:      "You are a TypeScript expert. Focus on type safety, generics, and modern TS patterns.",
			Tools:       []string{"Read", "Grep", "Glob"},
			Model:       "haiku",
		},
	}

	if err := manager2.RegisterAgents(agents); err != nil {
		log.Fatalf("Failed to register agents: %v", err)
	}

	fmt.Printf("Batch registered %d language expert agents\n", manager2.AgentCount())

	// Example 7: Agent Lifecycle Management
	fmt.Println("\n\nExample 7: Agent Lifecycle Management")
	fmt.Println("======================================")

	// Get a specific agent config
	if config, ok := manager.GetAgent("security"); ok {
		fmt.Printf("Security agent uses model: %s\n", config.Model)
		fmt.Printf("Security agent tools: %v\n", config.Tools)
	}

	// Unregister an agent
	manager.UnregisterAgent("docs")
	fmt.Printf("After unregistering docs: %d agents\n", manager.AgentCount())

	// Clear all sessions
	manager.ClearAllSessions()
	fmt.Println("Cleared all agent sessions")

	// Example 8: Parent Options Inheritance
	fmt.Println("\n\nExample 8: Parent Options Inheritance")
	fmt.Println("======================================")

	// Create parent options with budget tracking and permissions
	parentOpts := &claude.RunOptions{
		MaxTurns:       3,
		BudgetTracker:  claude.NewBudgetTracker(&claude.BudgetConfig{MaxBudgetUSD: 1.00}),
		PermissionMode: claude.PermissionModeDefault,
	}

	// Subagent will inherit budget tracker and permission settings
	fmt.Println("Subagent inherits:")
	fmt.Println("  - Budget tracking (from parent)")
	fmt.Println("  - Permission mode (from parent)")
	fmt.Println("  - MCP config path (from parent)")
	fmt.Println("  - Max turns (uses subagent's if specified, otherwise parent's)")
	fmt.Println("  - Model (uses subagent's if specified, otherwise parent's)")

	// This would run with inherited settings:
	// result, err := manager.RunAgent(ctx, "security", "Review this code", parentOpts)
	_ = parentOpts // Demonstrate the concept without actual execution

	fmt.Println("\nSubagent examples complete!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Package main demonstrates session management features in the Claude Code Go SDK.
// This example shows how to:
// 1. Create sessions with explicit SessionIDs (UUID format)
// 2. Fork existing sessions to create branches
// 3. Use ephemeral sessions that are not persisted to disk
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

func main() {
	// Create a new Claude client
	client := claude.NewClient("claude")

	// =========================================================================
	// Example 1: Explicit Session ID
	// =========================================================================
	// Use a specific UUID to identify and track your conversation.
	// This allows you to organize sessions logically and resume them later.
	fmt.Println("=== Example 1: Explicit Session ID ===")
	fmt.Println("Creating a session with a specific UUID for tracking...")

	// Generate a new session ID (UUID v4 format)
	sessionID := claude.GenerateSessionID()
	fmt.Printf("Session ID: %s\n\n", sessionID)

	// Run a prompt with the explicit session ID
	result, err := client.RunWithSession(
		"What is the capital of France? Keep your answer brief.",
		sessionID,
		&claude.RunOptions{
			Format: claude.JSONOutput,
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in Example 1: %v\n", err)
	} else {
		fmt.Printf("Response: %s\n", result.Result)
		fmt.Printf("Session ID from response: %s\n", result.SessionID)
		fmt.Printf("Cost: $%.6f\n", result.CostUSD)
	}
	fmt.Println()

	// =========================================================================
	// Example 2: Resuming a Session
	// =========================================================================
	// Continue a previous conversation using its session ID.
	// The context from the previous conversation is maintained.
	fmt.Println("=== Example 2: Resuming a Session ===")
	fmt.Println("Resuming the session to continue the conversation...")

	resumeResult, err := client.ResumeConversation(
		"What is its population? (referring to the city you just mentioned)",
		sessionID,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in Example 2: %v\n", err)
	} else {
		fmt.Printf("Response: %s\n", resumeResult.Result)
		fmt.Printf("Session ID: %s\n", resumeResult.SessionID)
	}
	fmt.Println()

	// =========================================================================
	// Example 3: Forking a Session
	// =========================================================================
	// Create a branch from an existing conversation. The original session
	// remains intact while you explore a different direction.
	fmt.Println("=== Example 3: Forking a Session ===")
	fmt.Println("Forking the session to explore a different direction...")

	forkResult, err := client.ForkAndRun(
		"What about Tokyo's population instead?",
		sessionID,
		&claude.RunOptions{
			Format: claude.JSONOutput,
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in Example 3: %v\n", err)
	} else {
		fmt.Printf("Response: %s\n", forkResult.Result)
		fmt.Printf("Forked Session ID: %s\n", forkResult.SessionID)
		fmt.Printf("Original Session ID was: %s\n", sessionID)
		fmt.Println("Note: The forked session has a different ID than the original")
	}
	fmt.Println()

	// =========================================================================
	// Example 4: Ephemeral Session (No Persistence)
	// =========================================================================
	// Run prompts without saving the session to disk. Useful for one-off
	// queries where you don't need to resume the conversation later.
	fmt.Println("=== Example 4: Ephemeral Session ===")
	fmt.Println("Running an ephemeral session that won't be saved...")

	ephemeralResult, err := client.RunEphemeral(
		"What is 2 + 2? Keep your answer brief.",
		&claude.RunOptions{
			Format: claude.JSONOutput,
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in Example 4: %v\n", err)
	} else {
		fmt.Printf("Response: %s\n", ephemeralResult.Result)
		fmt.Printf("Cost: $%.6f\n", ephemeralResult.CostUSD)
		fmt.Println("Note: This session will NOT be saved to disk")
	}
	fmt.Println()

	// =========================================================================
	// Example 5: Continue Last Session
	// =========================================================================
	// Continue the most recent conversation without specifying an ID.
	// Useful for quick follow-ups.
	fmt.Println("=== Example 5: Continue Last Session ===")
	fmt.Println("Continuing the most recent conversation...")

	continueResult, err := client.ContinueConversation(
		"Can you summarize what we discussed?",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in Example 5: %v\n", err)
	} else {
		fmt.Printf("Response: %s\n", continueResult.Result)
		fmt.Printf("Session ID: %s\n", continueResult.SessionID)
	}
	fmt.Println()

	// =========================================================================
	// Example 6: Session with Context and Timeout
	// =========================================================================
	// Combine session management with context cancellation for robust
	// production usage.
	fmt.Println("=== Example 6: Session with Context and Timeout ===")
	fmt.Println("Running a session with timeout control...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	newSessionID := claude.GenerateSessionID()
	ctxResult, err := client.RunWithSessionCtx(
		ctx,
		"List 3 programming languages.",
		newSessionID,
		&claude.RunOptions{
			Format: claude.JSONOutput,
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in Example 6: %v\n", err)
	} else {
		fmt.Printf("Response: %s\n", ctxResult.Result)
		fmt.Printf("Session ID: %s\n", ctxResult.SessionID)
		fmt.Printf("Completed in: %d ms\n", ctxResult.DurationMS)
	}
	fmt.Println()

	// =========================================================================
	// Summary
	// =========================================================================
	fmt.Println("=== Session Management Summary ===")
	fmt.Println(`Session management features:

1. SessionID        - Explicit UUID for tracking conversations
2. ResumeConversation - Resume a specific session by ID
3. ForkSession      - Branch from existing session with new ID
4. NoSessionPersistence - Ephemeral sessions (not saved)
5. ContinueConversation - Continue the most recent session

Use cases:
- Multi-turn conversations: Use SessionID + ResumeConversation
- Explore alternatives: Use ForkSession to branch
- One-off queries: Use RunEphemeral for no persistence
- Quick follow-ups: Use ContinueConversation

Best practices:
- Generate UUIDs with claude.GenerateSessionID()
- Store session IDs if you need to resume later
- Use ephemeral sessions for sensitive one-off queries
- Use context with timeout for production robustness`)
}

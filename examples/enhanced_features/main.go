package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

func main() {
	// Create client
	cc := claude.NewClient("claude")

	// Example 1: Enhanced tool permissions with granular control
	fmt.Println("🔧 Example 1: Enhanced Tool Permissions")
	opts := &claude.RunOptions{
		Format: claude.JSONOutput,
		AllowedTools: []string{
			"Bash(git log:*)",  // Allow git log with any arguments
			"Bash(git status)", // Allow only git status
			"Read",             // Allow all file reading
			"Write(src/**)",    // Allow writing only to src directory
		},
		DisallowedTools: []string{
			"Bash(rm:*)", // Explicitly block rm commands
		},
		ModelAlias: "sonnet", // Use model alias instead of full name
		Timeout:    30 * time.Second,
		Verbose:    true,
	}

	result, err := cc.RunPromptEnhanced("List recent git commits and explain the changes", opts)
	if err != nil {
		// Enhanced error handling
		if claudeErr, ok := err.(*claude.ClaudeError); ok {
			fmt.Printf("❌ Error Type: %s\n", claudeErr.Type)
			fmt.Printf("📄 Message: %s\n", claudeErr.Message)
			if suggestion, exists := claudeErr.Details["suggestion"]; exists {
				fmt.Printf("💡 Suggestion: %s\n", suggestion)
			}
			fmt.Printf("🔄 Retryable: %t\n", claudeErr.IsRetryable())
		} else {
			fmt.Printf("❌ Error: %v\n", err)
		}
		return
	}

	fmt.Printf("✅ Success! Cost: $%.4f, Duration: %dms\n", result.CostUSD, result.DurationMS)
	fmt.Printf("📊 Turns: %d\n", result.NumTurns)
	fmt.Printf("📝 Response: %s\n\n", result.Result)

	// Example 2: Retry logic with rate limit handling
	fmt.Println("🔄 Example 2: Retry Logic")
	retryPolicy := &claude.RetryPolicy{
		MaxRetries:    3,
		BaseDelay:     500 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err = cc.RunPromptWithRetryCtx(ctx, "Analyze this project structure", opts, retryPolicy)
	if err != nil {
		if claudeErr, ok := err.(*claude.ClaudeError); ok {
			fmt.Printf("❌ Failed after retries - Type: %s\n", claudeErr.Type)
			fmt.Printf("📄 Message: %s\n", claudeErr.Message)
		} else {
			fmt.Printf("❌ Error: %v\n", err)
		}
		return
	}

	fmt.Printf("✅ Success with retries! Cost: $%.4f\n", result.CostUSD)

	// Example 3: Validation before execution
	fmt.Println("🔍 Example 3: Input Validation")
	invalidOpts := &claude.RunOptions{
		AllowedTools: []string{
			"InvalidTool()", // This will trigger validation error
		},
		ModelAlias: "invalid-model",  // This will also fail validation
		Timeout:    -5 * time.Second, // Negative timeout
	}

	if err := claude.ValidateOptions(invalidOpts); err != nil {
		if claudeErr, ok := err.(*claude.ClaudeError); ok {
			fmt.Printf("❌ Validation Failed - Type: %s\n", claudeErr.Type)
			fmt.Printf("📄 Message: %s\n", claudeErr.Message)
			if field, exists := claudeErr.Details["field"]; exists {
				fmt.Printf("🏷️  Field: %s\n", field)
			}
		}
	}

	// Example 4: Safe permissions for production
	fmt.Println("🛡️  Example 4: Production-Safe Configuration")
	safeOpts := &claude.RunOptions{
		Format: claude.JSONOutput,
		AllowedTools: []string{
			"Read",                    // Safe read-only operations
			"Bash(git status)",        // Safe git operations
			"Bash(git log:--oneline)", // Limited git log
		},
		DisallowedTools: []string{
			"Bash(rm:*)",   // Block destructive operations
			"Bash(sudo:*)", // Block privilege escalation
			"Write",        // Block all writes (could be more specific)
		},
		ModelAlias: "sonnet",
		Timeout:    60 * time.Second,
	}

	fmt.Println("✅ Safe configuration validated!")
	if err := claude.ValidateOptions(safeOpts); err != nil {
		log.Printf("Validation error: %v", err)
	} else {
		fmt.Println("🔒 Configuration is production-ready")
	}
}

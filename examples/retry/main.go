// Package main demonstrates retry logic and error handling in the Claude Code Go SDK.
// This example shows how to:
// 1. Configure RetryPolicy for automatic retries
// 2. Detect and handle different error types
// 3. Extract retry-after information from rate limit errors
// 4. Implement custom error recovery patterns
package main

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

func main() {
	// =========================================================================
	// Example 1: Default Retry Policy
	// =========================================================================
	// The SDK provides a sensible default retry policy for handling transient errors.
	fmt.Println("=== Example 1: Default Retry Policy ===")

	defaultPolicy := claude.DefaultRetryPolicy()
	fmt.Printf("Max Retries:    %d\n", defaultPolicy.MaxRetries)
	fmt.Printf("Base Delay:     %v\n", defaultPolicy.BaseDelay)
	fmt.Printf("Max Delay:      %v\n", defaultPolicy.MaxDelay)
	fmt.Printf("Backoff Factor: %.1f\n", defaultPolicy.BackoffFactor)
	fmt.Println()

	// =========================================================================
	// Example 2: Custom Retry Policy
	// =========================================================================
	// Configure retry behavior for your specific use case.
	fmt.Println("=== Example 2: Custom Retry Policy ===")

	// Aggressive retry for critical operations
	aggressivePolicy := &claude.RetryPolicy{
		MaxRetries:    5,                     // More retry attempts
		BaseDelay:     50 * time.Millisecond, // Faster initial retry
		MaxDelay:      10 * time.Second,      // Longer max wait
		BackoffFactor: 1.5,                   // Slower backoff
	}
	fmt.Println("Aggressive Policy (for critical operations):")
	fmt.Printf("  Max Retries: %d, Base Delay: %v, Max Delay: %v\n",
		aggressivePolicy.MaxRetries, aggressivePolicy.BaseDelay, aggressivePolicy.MaxDelay)

	// Conservative retry for background tasks
	conservativePolicy := &claude.RetryPolicy{
		MaxRetries:    2,                // Fewer attempts
		BaseDelay:     1 * time.Second,  // Longer initial wait
		MaxDelay:      30 * time.Second, // Longer max wait
		BackoffFactor: 3.0,              // Aggressive backoff
	}
	fmt.Println("Conservative Policy (for background tasks):")
	fmt.Printf("  Max Retries: %d, Base Delay: %v, Max Delay: %v\n",
		conservativePolicy.MaxRetries, conservativePolicy.BaseDelay, conservativePolicy.MaxDelay)
	fmt.Println()

	// =========================================================================
	// Example 3: Error Type Detection
	// =========================================================================
	// The SDK categorizes errors for appropriate handling.
	fmt.Println("=== Example 3: Error Type Detection ===")

	errorTypes := []claude.ErrorType{
		claude.ErrorAuthentication,
		claude.ErrorRateLimit,
		claude.ErrorPermission,
		claude.ErrorCommand,
		claude.ErrorNetwork,
		claude.ErrorMCP,
		claude.ErrorValidation,
		claude.ErrorTimeout,
		claude.ErrorSession,
	}

	fmt.Println("Error Types and Retryability:")
	for _, errType := range errorTypes {
		fmt.Printf("  %-15s: Retryable=%v\n", errType.String(), errType.IsRetryable())
	}
	fmt.Println()

	// =========================================================================
	// Example 4: Simulated Error Handling
	// =========================================================================
	// Demonstrate how to handle different error scenarios.
	fmt.Println("=== Example 4: Simulated Error Handling ===")

	// Simulate different error scenarios
	simulatedErrors := []*claude.ClaudeError{
		{
			Type:    claude.ErrorRateLimit,
			Message: "Rate limit exceeded",
			Details: map[string]interface{}{"retry_after": 30},
		},
		{
			Type:    claude.ErrorNetwork,
			Message: "Connection refused",
		},
		{
			Type:    claude.ErrorAuthentication,
			Message: "Invalid API key",
		},
		{
			Type:    claude.ErrorMCP,
			Message: "MCP server connection timeout",
		},
	}

	for _, err := range simulatedErrors {
		fmt.Printf("\nError: %s\n", err.Error())
		fmt.Printf("  Type:      %s\n", err.Type.String())
		fmt.Printf("  Retryable: %v\n", err.IsRetryable())
		if delay := err.RetryDelay(); delay > 0 {
			fmt.Printf("  Retry After: %d seconds\n", delay)
		}
	}
	fmt.Println()

	// =========================================================================
	// Example 5: Using RunPromptWithRetry
	// =========================================================================
	// Execute prompts with automatic retry handling.
	fmt.Println("=== Example 5: Using RunPromptWithRetry ===")

	cc := claude.NewClient("claude")

	// Example with default retry policy
	fmt.Println("Code example with default retry:")
	fmt.Println(`  result, err := client.RunPromptWithRetry(
      "Your prompt here",
      &claude.RunOptions{Format: claude.JSONOutput},
      nil, // Uses DefaultRetryPolicy()
  )
  if err != nil {
      // All retries exhausted or non-retryable error
      handleError(err)
  }`)

	// Example with custom retry policy
	fmt.Println("Code example with custom retry policy:")
	fmt.Println(`  customPolicy := &claude.RetryPolicy{
      MaxRetries:    5,
      BaseDelay:     100 * time.Millisecond,
      MaxDelay:      10 * time.Second,
      BackoffFactor: 2.0,
  }
  result, err := client.RunPromptWithRetry(
      "Your prompt here",
      &claude.RunOptions{Format: claude.JSONOutput},
      customPolicy,
  )`)
	_ = cc
	fmt.Println()

	// =========================================================================
	// Example 6: Context-Aware Retry with Timeout
	// =========================================================================
	// Combine retry logic with context cancellation.
	fmt.Println("=== Example 6: Context-Aware Retry ===")

	fmt.Println("Using RunPromptWithRetryCtx for timeout control:")
	fmt.Println(`  ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
  defer cancel()

  result, err := client.RunPromptWithRetryCtx(
      ctx,
      "Long-running prompt",
      &claude.RunOptions{Format: claude.JSONOutput},
      customPolicy,
  )
  if err != nil {
      if errors.Is(err, context.DeadlineExceeded) {
          log.Println("Operation timed out after all retries")
      }
  }`)
	fmt.Println()

	// =========================================================================
	// Example 7: Custom Error Recovery Pattern
	// =========================================================================
	// Implement sophisticated error handling with fallback strategies.
	fmt.Println("=== Example 7: Custom Error Recovery Pattern ===")

	// Demonstrate a custom error handling pattern
	handleWithRecovery := func(err error) {
		var claudeErr *claude.ClaudeError
		if !errors.As(err, &claudeErr) {
			fmt.Println("  Non-Claude error, cannot classify")
			return
		}

		switch claudeErr.Type {
		case claude.ErrorRateLimit:
			delay := claudeErr.RetryDelay()
			fmt.Printf("  Rate limited! Wait %d seconds before retry\n", delay)
			// In real code: time.Sleep(time.Duration(delay) * time.Second)

		case claude.ErrorAuthentication:
			fmt.Println("  Authentication failed! Check API key")
			// In real code: refreshAPIKey() or notify admin

		case claude.ErrorNetwork:
			fmt.Println("  Network error! Check connectivity")
			// In real code: switch to backup endpoint or retry

		case claude.ErrorMCP:
			if claudeErr.IsRetryable() {
				fmt.Println("  MCP connection error! Server might be down")
				// In real code: retry or use fallback MCP config
			} else {
				fmt.Println("  MCP config error! Fix configuration")
				// In real code: log and notify, don't retry
			}

		case claude.ErrorTimeout:
			fmt.Println("  Operation timed out! Consider smaller prompt")
			// In real code: simplify prompt or increase timeout

		case claude.ErrorPermission:
			fmt.Println("  Permission denied! Update tool permissions")
			// In real code: notify user to update permissions

		default:
			fmt.Printf("  Unhandled error type: %s\n", claudeErr.Type)
		}
	}

	fmt.Println("Demonstrating error recovery for each type:")
	for _, err := range simulatedErrors {
		handleWithRecovery(err)
	}
	fmt.Println()

	// =========================================================================
	// Example 8: Enhanced Mode with Built-in Retry
	// =========================================================================
	// RunPromptEnhanced automatically includes retry logic.
	fmt.Println("=== Example 8: Enhanced Mode with Built-in Retry ===")

	fmt.Println("RunPromptEnhanced includes automatic retry:")
	fmt.Println(`  // Enhanced mode includes: validation, timeout support, and default retry
  result, err := client.RunPromptEnhanced(
      "Your prompt",
      &claude.RunOptions{
          Format:  claude.JSONOutput,
          Timeout: 30 * time.Second,
      },
  )
  // Automatically retries on transient errors with DefaultRetryPolicy`)

	fmt.Println("With context for cancellation:")
	fmt.Println(`  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  result, err := client.RunPromptEnhancedCtx(ctx, "Your prompt", opts)`)
	fmt.Println()

	// =========================================================================
	// Example 9: Exponential Backoff Demonstration
	// =========================================================================
	// Visualize how backoff works.
	fmt.Println("=== Example 9: Exponential Backoff Visualization ===")

	policy := claude.DefaultRetryPolicy()
	fmt.Printf("Policy: BaseDelay=%v, MaxDelay=%v, BackoffFactor=%.1f\n\n",
		policy.BaseDelay, policy.MaxDelay, policy.BackoffFactor)

	fmt.Println("Attempt | Delay (without jitter)")
	fmt.Println("--------|------------------------")
	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		// Note: We can't directly call calculateBackoff as it's private
		// This simulates the calculation using math.Pow for exponential growth
		delay := time.Duration(float64(policy.BaseDelay) * math.Pow(policy.BackoffFactor, float64(attempt)))
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}
		if attempt == 0 {
			fmt.Printf("   %d    | Immediate (no delay)\n", attempt)
		} else {
			fmt.Printf("   %d    | %v\n", attempt, delay)
		}
	}
	fmt.Println("\nNote: Actual delays include ±20% jitter to prevent thundering herd")
	fmt.Println()

	// =========================================================================
	// Summary
	// =========================================================================
	fmt.Println("=== Retry and Error Handling Summary ===")
	fmt.Println(`Error Types (claude.ErrorType):
- ErrorAuthentication  - API key issues (not retryable)
- ErrorRateLimit       - Rate limiting (retryable with delay)
- ErrorPermission      - Tool access denied (not retryable)
- ErrorCommand         - CLI errors (not retryable)
- ErrorNetwork         - Connectivity issues (retryable)
- ErrorMCP             - MCP server errors (conditionally retryable)
- ErrorValidation      - Input validation (not retryable)
- ErrorTimeout         - Operation timeout (retryable)
- ErrorSession         - Session errors (not retryable)

RetryPolicy Configuration:
- MaxRetries     - Maximum retry attempts
- BaseDelay      - Initial delay between retries
- MaxDelay       - Maximum delay (caps exponential growth)
- BackoffFactor  - Multiplier for exponential backoff

ClaudeError Methods:
- Error()        - Get error message
- IsRetryable()  - Check if retry makes sense
- RetryDelay()   - Get recommended wait time
- Unwrap()       - Access underlying error

Retry Functions:
- DefaultRetryPolicy()         - Get sensible defaults
- RunPromptWithRetry()         - Execute with retry
- RunPromptWithRetryCtx()      - With context support
- RunPromptEnhanced()          - All features including retry
- RunPromptEnhancedCtx()       - Enhanced with context

Best Practices:
1. Always check IsRetryable() before retrying
2. Use RetryDelay() for rate limit errors
3. Set appropriate timeouts with context
4. Use conservative retry for background tasks
5. Log retry attempts for debugging
6. Have fallback strategies for critical paths`)
}

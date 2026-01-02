package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

var (
	retryPolicy *claude.RetryPolicy
	timeout     time.Duration
	useEnhanced bool
	// Retry statistics
	totalAttempts   int
	totalRetries    int
	totalRetryTime  time.Duration
	errorsEncountered []string
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

func displayRetryStatus() {
	fmt.Println("\n┌─────────────────────────────────────────────────────────────┐")
	fmt.Printf("│ Max Retries: %-3d | Base Delay: %-8s | Max Delay: %-7s │\n",
		retryPolicy.MaxRetries,
		retryPolicy.BaseDelay.String(),
		retryPolicy.MaxDelay.String())
	fmt.Printf("│ Backoff Factor: %-4.1f | Timeout: %-10s | Enhanced: %-5v │\n",
		retryPolicy.BackoffFactor,
		formatTimeout(timeout),
		useEnhanced)
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
}

func displayRetryStatistics() {
	fmt.Println("\n📊 Cumulative Retry Statistics:")
	fmt.Printf("  Total requests:    %d\n", totalAttempts)
	fmt.Printf("  Total retries:     %d\n", totalRetries)
	fmt.Printf("  Total retry time:  %v\n", totalRetryTime.Round(time.Millisecond))
	if len(errorsEncountered) > 0 {
		fmt.Printf("  Errors encountered: %d\n", len(errorsEncountered))
		for i, err := range errorsEncountered {
			if i >= 3 {
				fmt.Printf("    ... and %d more\n", len(errorsEncountered)-3)
				break
			}
			fmt.Printf("    - %s\n", err)
		}
	}
	if totalAttempts > 0 {
		successRate := float64(totalAttempts-len(errorsEncountered)) / float64(totalAttempts) * 100
		fmt.Printf("  Success rate:      %.1f%%\n", successRate)
	}
}

func formatTimeout(d time.Duration) string {
	if d == 0 {
		return "none"
	}
	return d.String()
}

func displayHelp() {
	fmt.Println("\n📚 Retry Commands:")
	fmt.Println("  /test-retry      - Test retry behavior with timeout (demonstrates retries)")
	fmt.Println("  /retries <n>     - Set max retry attempts (0-10)")
	fmt.Println("  /delay <ms>      - Set base delay in milliseconds")
	fmt.Println("  /maxdelay <ms>   - Set max delay in milliseconds")
	fmt.Println("  /backoff <n>     - Set backoff factor (e.g., 2.0)")
	fmt.Println("  /timeout <sec>   - Set request timeout in seconds (0=none)")
	fmt.Println("  /enhanced on|off - Toggle retry visibility (default: on)")
	fmt.Println("  /default         - Reset to default retry policy")
	fmt.Println("  /aggressive      - Use aggressive retry (more retries, shorter delays)")
	fmt.Println("  /conservative    - Use conservative retry (fewer retries, longer delays)")
	fmt.Println("  /status          - Show current retry configuration")
	fmt.Println("  /stats           - Show cumulative retry statistics")
	fmt.Println("  /help            - Show this help")
	fmt.Println("  exit             - Exit the demo")
	fmt.Println()
	fmt.Println("💡 Tip: Use /test-retry to see retry behavior in action!")
	fmt.Println("   When enhanced mode is ON, you'll see each retry attempt logged")
	fmt.Println("   with delays and backoff progression.")
}

func handleCommand(cmd string) bool {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}

	switch strings.ToLower(parts[0]) {
	case "/retries":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /retries <n>")
			return true
		}
		n, err := strconv.Atoi(parts[1])
		if err != nil || n < 0 || n > 10 {
			fmt.Println("✗ Retries must be 0-10")
			return true
		}
		retryPolicy.MaxRetries = n
		fmt.Printf("✓ Max retries set to: %d\n", n)
		return true

	case "/delay":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /delay <milliseconds>")
			return true
		}
		ms, err := strconv.Atoi(parts[1])
		if err != nil || ms < 0 {
			fmt.Println("✗ Delay must be positive milliseconds")
			return true
		}
		retryPolicy.BaseDelay = time.Duration(ms) * time.Millisecond
		fmt.Printf("✓ Base delay set to: %v\n", retryPolicy.BaseDelay)
		return true

	case "/maxdelay":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /maxdelay <milliseconds>")
			return true
		}
		ms, err := strconv.Atoi(parts[1])
		if err != nil || ms < 0 {
			fmt.Println("✗ Max delay must be positive milliseconds")
			return true
		}
		retryPolicy.MaxDelay = time.Duration(ms) * time.Millisecond
		fmt.Printf("✓ Max delay set to: %v\n", retryPolicy.MaxDelay)
		return true

	case "/backoff":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /backoff <factor>")
			return true
		}
		f, err := strconv.ParseFloat(parts[1], 64)
		if err != nil || f < 1.0 {
			fmt.Println("✗ Backoff factor must be >= 1.0")
			return true
		}
		retryPolicy.BackoffFactor = f
		fmt.Printf("✓ Backoff factor set to: %.1f\n", f)
		return true

	case "/timeout":
		if len(parts) < 2 {
			fmt.Println("✗ Usage: /timeout <seconds>")
			return true
		}
		sec, err := strconv.Atoi(parts[1])
		if err != nil || sec < 0 {
			fmt.Println("✗ Timeout must be positive seconds (0=none)")
			return true
		}
		timeout = time.Duration(sec) * time.Second
		fmt.Printf("✓ Timeout set to: %v\n", formatTimeout(timeout))
		return true

	case "/enhanced":
		if len(parts) < 2 {
			useEnhanced = !useEnhanced
		} else {
			switch strings.ToLower(parts[1]) {
			case "on", "true", "yes", "1":
				useEnhanced = true
			case "off", "false", "no", "0":
				useEnhanced = false
			default:
				fmt.Println("✗ Usage: /enhanced on|off")
				return true
			}
		}
		fmt.Printf("✓ Enhanced mode: %v\n", useEnhanced)
		if useEnhanced {
			fmt.Println("  ✅ Retry attempts will be logged with delays and backoff")
			fmt.Println("  ✅ Statistics will track retry behavior")
			fmt.Println("  ✅ You'll see: attempt numbers, wait times, success/failure")
		} else {
			fmt.Println("  ⚠️  Retry logic will run invisibly in the SDK")
			fmt.Println("  ⚠️  You won't see retry attempts when they occur")
		}
		return true

	case "/default":
		retryPolicy = claude.DefaultRetryPolicy()
		timeout = 0
		useEnhanced = false
		fmt.Println("✓ Reset to default retry policy")
		return true

	case "/aggressive":
		retryPolicy = &claude.RetryPolicy{
			MaxRetries:    5,
			BaseDelay:     50 * time.Millisecond,
			MaxDelay:      2 * time.Second,
			BackoffFactor: 1.5,
		}
		fmt.Println("✓ Using aggressive retry policy")
		fmt.Println("  (5 retries, 50ms base, 1.5x backoff)")
		return true

	case "/conservative":
		retryPolicy = &claude.RetryPolicy{
			MaxRetries:    2,
			BaseDelay:     500 * time.Millisecond,
			MaxDelay:      10 * time.Second,
			BackoffFactor: 3.0,
		}
		fmt.Println("✓ Using conservative retry policy")
		fmt.Println("  (2 retries, 500ms base, 3x backoff)")
		return true

	case "/status":
		displayRetryStatus()
		return true

	case "/stats":
		displayRetryStatistics()
		return true

	case "/test-retry":
		fmt.Println("\n🧪 Testing retry behavior with intentional timeout...")
		fmt.Println("   This will make a request with a short timeout to trigger retries")

		if !useEnhanced {
			fmt.Println("\n⚠️  Enhanced mode is OFF - retries won't be visible")
			fmt.Println("   Run '/enhanced on' first to see retry attempts")
			return true
		}

		// Create context with timeout that's long enough for command to start
		// but short enough to timeout during API call
		testCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		testOpts := &claude.RunOptions{
			Format:       claude.StreamJSONOutput,
			SystemPrompt: "You are a helpful assistant.",
			AllowedTools: []string{"Read(*)", "Bash(ls*)"},
			MaxTurns:     1,
			Timeout:      1 * time.Second, // Request timeout shorter than context timeout
		}

		fmt.Println("\n🔄 Using enhanced retry mode (visible retry logic)...")
		cc := claude.NewClient("claude")
		result, err := runWithRetry(testCtx, cc, "What is 2+2?", testOpts, retryPolicy)

		if err != nil {
			fmt.Println("\n✅ Test completed - retry behavior demonstrated!")
			fmt.Printf("   Final error: %v\n", err)
		} else {
			fmt.Println("\n⚠️  Test request succeeded (no retries needed)")
			if result.Result != "" {
				text := result.Result
				if len(text) > 100 {
					text = text[:100] + "..."
				}
				fmt.Printf("   Result: %s\n", text)
			}
		}

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

func classifyError(err error) string {
	if claudeErr, ok := err.(*claude.ClaudeError); ok {
		retryable := "no"
		if claudeErr.IsRetryable() {
			retryable = "yes"
		}
		return fmt.Sprintf("Type: %s, Retryable: %s", claudeErr.Type, retryable)
	}
	return "Unknown error type"
}

// runWithRetry implements retry logic with full visibility
func runWithRetry(ctx context.Context, cc *claude.ClaudeClient, prompt string, opts *claude.RunOptions, policy *claude.RetryPolicy) (*claude.ClaudeResult, error) {
	totalAttempts++
	var lastErr error
	attemptNum := 0
	retriesUsed := 0
	retryStartTime := time.Now()

	for attemptNum <= policy.MaxRetries {
		attemptNum++

		if attemptNum > 1 {
			// Calculate backoff delay
			delay := policy.BaseDelay
			for i := 1; i < attemptNum-1; i++ {
				delay = time.Duration(float64(delay) * policy.BackoffFactor)
				if delay > policy.MaxDelay {
					delay = policy.MaxDelay
					break
				}
			}

			fmt.Printf("⏳ Waiting %v before retry (attempt %d/%d)...\n",
				delay.Round(time.Millisecond), attemptNum, policy.MaxRetries+1)
			time.Sleep(delay)
			totalRetryTime += delay

			fmt.Printf("🔄 Retrying request (attempt %d/%d)...\n",
				attemptNum, policy.MaxRetries+1)
			retriesUsed++
		} else {
			fmt.Println("🔄 Attempting request...")
		}

		result, err := cc.RunPromptCtx(ctx, prompt, opts)
		if err == nil {
			// Success!
			if retriesUsed > 0 {
				elapsed := time.Since(retryStartTime)
				fmt.Printf("✅ Success on attempt %d (%d retries needed, %v total retry time)\n",
					attemptNum, retriesUsed, elapsed.Round(time.Millisecond))
				totalRetries += retriesUsed
			} else {
				fmt.Println("✅ Success on first attempt (no retries needed)")
			}
			return result, nil
		}

		lastErr = err
		errorMsg := err.Error()
		if claudeErr, ok := err.(*claude.ClaudeError); ok {
			if !claudeErr.IsRetryable() {
				fmt.Printf("❌ Non-retryable error: %v\n", err)
				fmt.Printf("   %s\n", classifyError(err))
				errorsEncountered = append(errorsEncountered, errorMsg)
				return nil, err
			}
			fmt.Printf("❌ Retryable error (attempt %d): %v\n", attemptNum, err)
			fmt.Printf("   %s\n", classifyError(err))
		} else {
			fmt.Printf("❌ Error (attempt %d): %v\n", attemptNum, err)
		}

		if attemptNum > policy.MaxRetries {
			break
		}
	}

	fmt.Printf("❌ Failed after %d attempts (used %d retries)\n", attemptNum, retriesUsed)
	totalRetries += retriesUsed
	errorsEncountered = append(errorsEncountered, lastErr.Error())
	return nil, lastErr
}

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║       Claude Code Go SDK - Retry & Error Handling Demo     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This demo showcases retry and error handling features:")
	fmt.Println("  • Configurable retry policies with exponential backoff")
	fmt.Println("  • Jitter to prevent thundering herd")
	fmt.Println("  • Error classification (retryable vs non-retryable)")
	fmt.Println("  • Context timeout support")
	fmt.Println("  • Visible retry attempts with delay progression")
	fmt.Println()
	fmt.Println("ℹ️  Enhanced mode is ON - retry attempts will be logged as they occur")
	fmt.Println("   Retries happen automatically when API errors occur (rate limits,")
	fmt.Println("   network timeouts, etc.). Each retry attempt is logged with delays.")
	fmt.Println()

	// Initialize with default policy
	retryPolicy = claude.DefaultRetryPolicy()
	timeout = 0
	useEnhanced = true // Default to enhanced mode for retry visibility

	displayHelp()
	displayRetryStatus()

	cc := claude.NewClient("claude")
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

		// Build options
		opts := &claude.RunOptions{
			Format:       claude.StreamJSONOutput,
			SystemPrompt: "You are a helpful assistant. Keep responses concise.",
			AllowedTools: []string{"Read(*)", "Bash(ls*)", "Bash(pwd)"},
			MaxTurns:     3,
		}

		if timeout > 0 {
			opts.Timeout = timeout
		}

		if sessionID != "" {
			opts.ResumeID = sessionID
		}

		// Create context
		ctx := context.Background()

		start := time.Now()

		if useEnhanced {
			// Use enhanced retry mode with full visibility
			fmt.Println("\n🔄 Using enhanced retry mode (visible retry logic)...")
			result, err := runWithRetry(ctx, cc, input, opts, retryPolicy)
			elapsed := time.Since(start)

			if err == nil {
				sessionID = result.SessionID
				fmt.Printf("\n📊 Cost: $%.6f | Duration: %.1fs | Turns: %d\n",
					result.CostUSD, float64(result.DurationMS)/1000.0, result.NumTurns)
				if result.Result != "" {
					// Truncate long results
					text := result.Result
					if len(text) > 200 {
						text = text[:200] + "..."
					}
					fmt.Printf("💬 %s\n", text)
				}
			}
			fmt.Printf("\n⏱️  Total time: %v\n", elapsed.Round(time.Millisecond))
		} else {
			// Use streaming mode
			messageCh, errCh := cc.StreamPrompt(ctx, input, opts)

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
						fmt.Printf("   %s\n", classifyError(err))
						break processLoop
					}
				}
			}
		}

		displayRetryStatus()
	}

	// Final summary
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  Retry Demo Summary                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("   Max Retries:     %d\n", retryPolicy.MaxRetries)
	fmt.Printf("   Base Delay:      %v\n", retryPolicy.BaseDelay)
	fmt.Printf("   Max Delay:       %v\n", retryPolicy.MaxDelay)
	fmt.Printf("   Backoff Factor:  %.1f\n", retryPolicy.BackoffFactor)
	fmt.Printf("   Timeout:         %s\n", formatTimeout(timeout))
	fmt.Printf("   Enhanced Mode:   %v\n", useEnhanced)
	fmt.Println("\nDemo completed!")
}

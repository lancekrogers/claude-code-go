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

func formatTimeout(d time.Duration) string {
	if d == 0 {
		return "none"
	}
	return d.String()
}

func displayHelp() {
	fmt.Println("\n📚 Retry Commands:")
	fmt.Println("  /retries <n>     - Set max retry attempts (0-10)")
	fmt.Println("  /delay <ms>      - Set base delay in milliseconds")
	fmt.Println("  /maxdelay <ms>   - Set max delay in milliseconds")
	fmt.Println("  /backoff <n>     - Set backoff factor (e.g., 2.0)")
	fmt.Println("  /timeout <sec>   - Set request timeout in seconds (0=none)")
	fmt.Println("  /enhanced on|off - Toggle enhanced mode (auto-retry)")
	fmt.Println("  /default         - Reset to default retry policy")
	fmt.Println("  /aggressive      - Use aggressive retry (more retries, shorter delays)")
	fmt.Println("  /conservative    - Use conservative retry (fewer retries, longer delays)")
	fmt.Println("  /status          - Show current retry configuration")
	fmt.Println("  /help            - Show this help")
	fmt.Println("  exit             - Exit the demo")
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
			fmt.Println("  (Uses RunPromptWithRetry with automatic retry logic)")
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
	fmt.Println("  • Enhanced mode with automatic retry")
	fmt.Println()

	// Initialize with default policy
	retryPolicy = claude.DefaultRetryPolicy()
	timeout = 0
	useEnhanced = false

	displayHelp()
	displayRetryStatus()

	client := claude.NewClient("claude")
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
			// Use the enhanced retry method
			fmt.Println("🔄 Using enhanced mode with automatic retry...")
			result, err := client.RunPromptWithRetryCtx(ctx, input, opts, retryPolicy)
			elapsed := time.Since(start)

			if err != nil {
				fmt.Printf("❌ Error after retries: %v\n", err)
				fmt.Printf("   %s\n", classifyError(err))
			} else {
				sessionID = result.SessionID
				fmt.Printf("✅ Success!\n")
				fmt.Printf("📊 Cost: $%.6f | Duration: %.1fs | Turns: %d\n",
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
			fmt.Printf("⏱️  Total time: %v\n", elapsed.Round(time.Millisecond))
		} else {
			// Use streaming mode
			messageCh, errCh := client.StreamPrompt(ctx, input, opts)

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

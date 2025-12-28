// Package main demonstrates streaming response handling in the Claude Code Go SDK.
// This example shows how to:
// 1. Use StreamPrompt for real-time message processing
// 2. Handle the message and error channels
// 3. Process different message types
// 4. Implement context cancellation for streaming
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

func main() {
	// =========================================================================
	// Example 1: Basic Streaming
	// =========================================================================
	// StreamPrompt returns two channels: messages and errors.
	// Messages arrive as Claude generates the response.
	fmt.Println("=== Example 1: Basic Streaming Pattern ===")

	os.Stdout.WriteString(`
Basic streaming pattern:

  client := claude.NewClient("claude")
  ctx := context.Background()

  // StreamPrompt returns two channels
  messageCh, errCh := client.StreamPrompt(ctx, "Your prompt", &claude.RunOptions{})

  // Process messages as they arrive
  for msg := range messageCh {
      fmt.Printf("Type: %s\n", msg.Type)
      if msg.Type == "result" {
          fmt.Printf("Final result: %s\n", msg.Result)
          break
      }
  }

  // Check for errors
  for err := range errCh {
      log.Printf("Error: %v\n", err)
  }
`)

	// =========================================================================
	// Example 2: Message Types
	// =========================================================================
	// Claude sends different message types during streaming.
	fmt.Println("=== Example 2: Message Types ===")

	fmt.Println(`Message types you may receive:

  Type: "system"
    - Initial system message with available tools and MCP servers
    - Contains: Tools[], MCPServers[]

  Type: "assistant"
    - Claude's response content
    - Contains: Message (raw JSON of assistant message)

  Type: "user"
    - Echoed user messages (rare in streaming)
    - Contains: Message

  Type: "result"
    - Final result when conversation completes
    - Contains: Result, CostUSD, DurationMS, NumTurns, SessionID, IsError`)

	// =========================================================================
	// Example 3: Processing Messages
	// =========================================================================
	// Handle each message type appropriately.
	fmt.Println("=== Example 3: Processing Messages ===")

	os.Stdout.WriteString(`
Complete message processing pattern:

  func processMessages(messageCh <-chan claude.Message) {
      for msg := range messageCh {
          switch msg.Type {
          case "system":
              fmt.Printf("System: %d tools available\n", len(msg.Tools))
              for _, server := range msg.MCPServers {
                  fmt.Printf("  MCP Server: %s (%s)\n", server.Name, server.Status)
              }

          case "assistant":
              // Parse the raw message JSON if needed
              var content map[string]interface{}
              if err := json.Unmarshal(msg.Message, &content); err == nil {
                  fmt.Printf("Assistant: %v\n", content)
              }

          case "result":
              fmt.Printf("=== Completed ===\n")
              fmt.Printf("Result: %s\n", msg.Result)
              fmt.Printf("Cost: $%.6f\n", msg.CostUSD)
              fmt.Printf("Duration: %dms\n", msg.DurationMS)
              fmt.Printf("Turns: %d\n", msg.NumTurns)
              fmt.Printf("Session: %s\n", msg.SessionID)
              if msg.IsError {
                  fmt.Println("Note: Result indicates an error occurred")
              }
              return // Exit on result

          default:
              fmt.Printf("Unknown type: %s\n", msg.Type)
          }
      }
  }
`)

	// =========================================================================
	// Example 4: Concurrent Error Handling
	// =========================================================================
	// Handle errors in a separate goroutine.
	fmt.Println("=== Example 4: Concurrent Error Handling ===")

	os.Stdout.WriteString(`
Handle errors concurrently:

  messageCh, errCh := client.StreamPrompt(ctx, prompt, opts)

  // Start error handler in background
  go func() {
      for err := range errCh {
          log.Printf("Stream error: %v\n", err)
          // Optionally cancel context on critical errors
          if isCriticalError(err) {
              cancel()
          }
      }
  }()

  // Process messages in main goroutine
  for msg := range messageCh {
      // ... process messages
  }
`)

	// =========================================================================
	// Example 5: Context Cancellation
	// =========================================================================
	// Cancel streaming mid-response.
	fmt.Println("=== Example 5: Context Cancellation ===")

	os.Stdout.WriteString(`
Cancel streaming with context:

  ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
  defer cancel()

  messageCh, errCh := client.StreamPrompt(ctx, prompt, opts)

  go func() {
      for err := range errCh {
          if err == context.DeadlineExceeded {
              log.Println("Streaming timed out")
          } else if err == context.Canceled {
              log.Println("Streaming was cancelled")
          } else {
              log.Printf("Stream error: %v\n", err)
          }
      }
  }()

  messageCount := 0
  for msg := range messageCh {
      messageCount++
      // Cancel after receiving 10 messages
      if messageCount >= 10 {
          cancel()
          break
      }
  }
`)

	// =========================================================================
	// Example 6: Signal Handling for Graceful Shutdown
	// =========================================================================
	// Handle OS signals to gracefully stop streaming.
	fmt.Println("=== Example 6: Signal Handling ===")

	fmt.Println(`Graceful shutdown with signals:

  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  // Setup signal handling
  sigCh := make(chan os.Signal, 1)
  signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

  go func() {
      <-sigCh
      fmt.Println("\nReceived interrupt, stopping stream...")
      cancel()
  }()

  messageCh, errCh := client.StreamPrompt(ctx, prompt, opts)
  // ... process as usual`)

	// =========================================================================
	// Example 7: Real-Time Progress Display
	// =========================================================================
	// Display streaming progress to the user.
	fmt.Println("=== Example 7: Real-Time Progress Display ===")

	os.Stdout.WriteString(`
Display streaming progress:

  func streamWithProgress(client *claude.ClaudeClient, prompt string) {
      ctx := context.Background()
      opts := &claude.RunOptions{Format: claude.StreamJSONOutput}

      messageCh, errCh := client.StreamPrompt(ctx, prompt, opts)
      startTime := time.Now()

      go func() {
          for err := range errCh {
              fmt.Fprintf(os.Stderr, "\rError: %v\n", err)
          }
      }()

      messageCount := 0
      for msg := range messageCh {
          messageCount++
          elapsed := time.Since(startTime)

          // Update progress line (overwrite previous)
          fmt.Printf("\r[%s] Messages: %d, Type: %s    ",
              elapsed.Round(time.Millisecond), messageCount, msg.Type)

          if msg.Type == "result" {
              fmt.Printf("\n\nCompleted in %s\n", elapsed)
              fmt.Printf("Cost: $%.6f\n", msg.CostUSD)
              break
          }
      }
  }
`)

	// =========================================================================
	// Example 8: Live Demo (Simulated)
	// =========================================================================
	// Demonstrate what streaming looks like with simulated messages.
	fmt.Println("=== Example 8: Simulated Streaming Demo ===")

	// Create simulated messages to show the flow
	simulatedMessages := []claude.Message{
		{Type: "system", Tools: []string{"Read", "Write", "Bash"}},
		{Type: "assistant", Message: json.RawMessage(`{"content": "Starting analysis..."}`)},
		{Type: "assistant", Message: json.RawMessage(`{"content": "Processing your request..."}`)},
		{Type: "assistant", Message: json.RawMessage(`{"content": "Almost done..."}`)},
		{
			Type:       "result",
			Result:     "Here is your completed response!",
			CostUSD:    0.000542,
			DurationMS: 1523,
			NumTurns:   1,
			SessionID:  "abc123-def456",
		},
	}

	fmt.Println("Simulating streaming output:")
	fmt.Println()
	for i, msg := range simulatedMessages {
		time.Sleep(200 * time.Millisecond) // Simulate delay

		switch msg.Type {
		case "system":
			fmt.Printf("[%d] SYSTEM: %d tools available\n", i+1, len(msg.Tools))
		case "assistant":
			var content map[string]interface{}
			json.Unmarshal(msg.Message, &content)
			fmt.Printf("[%d] ASSISTANT: %v\n", i+1, content["content"])
		case "result":
			fmt.Printf("[%d] RESULT: %s\n", i+1, msg.Result)
			fmt.Printf("    Cost: $%.6f, Duration: %dms\n", msg.CostUSD, msg.DurationMS)
		}
	}
	fmt.Println()

	// =========================================================================
	// Example 9: Complete Working Example
	// =========================================================================
	fmt.Println("=== Example 9: Complete Working Example ===")

	os.Stdout.WriteString(`
Complete example you can run:

  package main

  import (
      "context"
      "fmt"
      "os"
      "os/signal"
      "syscall"
      "time"

      "github.com/lancekrogers/claude-code-go/pkg/claude"
  )

  func main() {
      client := claude.NewClient("claude")

      ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
      defer cancel()

      // Handle Ctrl+C
      sigCh := make(chan os.Signal, 1)
      signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
      go func() {
          <-sigCh
          fmt.Println("\nCancelling...")
          cancel()
      }()

      messageCh, errCh := client.StreamPrompt(ctx, "Write a haiku about Go", nil)

      // Error handler
      go func() {
          for err := range errCh {
              fmt.Fprintf(os.Stderr, "Error: %v\n", err)
          }
      }()

      // Message processor
      for msg := range messageCh {
          if msg.Type == "result" {
              fmt.Printf("\n%s\n", msg.Result)
              fmt.Printf("\nCost: $%.6f\n", msg.CostUSD)
              break
          }
      }
  }
`)

	// Avoid unused import warnings
	_ = os.Stdin
	_ = signal.Notify
	_ = syscall.SIGINT
	_ = context.Background
	_ = time.Second

	// =========================================================================
	// Summary
	// =========================================================================
	fmt.Println("=== Streaming Summary ===")
	fmt.Println(`StreamPrompt Returns:
- messageCh (<-chan Message) - Receives messages as they arrive
- errCh (<-chan error)       - Receives any errors during streaming

Message Fields:
- Type       - "system", "assistant", "user", or "result"
- Subtype    - Additional type information
- Message    - Raw JSON message content
- SessionID  - Conversation session ID
- Result     - Final result text (for "result" type)
- CostUSD    - Total cost (for "result" type)
- DurationMS - Total duration (for "result" type)
- NumTurns   - Number of turns (for "result" type)
- IsError    - Whether result is an error (for "result" type)

Best Practices:
1. Always handle the error channel (use goroutine)
2. Use context for timeout and cancellation
3. Check msg.Type == "result" for completion
4. Handle signals for graceful shutdown
5. Close resources when streaming ends
6. Consider buffering for high-frequency messages`)
}

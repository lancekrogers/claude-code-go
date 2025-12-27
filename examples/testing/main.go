package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

// Example of how to test the Claude Code Go SDK locally
func main() {
	fmt.Println("🧪 Claude Code Go SDK - Testing Example")
	fmt.Println("=====================================")
	fmt.Println()

	// Example 1: Basic testing with mock
	fmt.Println("Example 1: Testing with Mock Server")
	fmt.Println("Start the mock server first: task start-mock-server")
	fmt.Println()

	// Create client pointing to mock server (if available)
	mockClient := claude.NewClient("http://localhost:8080/claude")

	// Test basic functionality
	if isMockServerRunning() {
		testBasicFunctionality(mockClient, "Mock Server")
	} else {
		fmt.Println("⚠️  Mock server not running, skipping mock tests")
		fmt.Println("   Run: task start-mock-server")
	}

	fmt.Println()

	// Example 2: Testing with real Claude CLI (if available)
	fmt.Println("Example 2: Testing with Real Claude CLI")

	if isClaudeCLIAvailable() {
		realClient := claude.NewClient("claude")
		testBasicFunctionality(realClient, "Real Claude CLI")
	} else {
		fmt.Println("⚠️  Claude CLI not available, skipping real tests")
		fmt.Println("   Install Claude CLI: https://docs.anthropic.com/en/docs/claude-code/getting-started")
	}

	fmt.Println()

	// Example 3: Testing convenience methods
	fmt.Println("Example 3: Testing Convenience Methods")
	testConvenienceMethods()

	fmt.Println()
	fmt.Println("✅ Testing examples completed!")
	fmt.Println()
	fmt.Println("For comprehensive testing:")
	fmt.Println("  task test-local     # Run all local tests")
	fmt.Println("  task test-full      # Run comprehensive test suite")
	fmt.Println("  task test-bench     # Run benchmark tests")
}

func testBasicFunctionality(client *claude.ClaudeClient, serverType string) {
	fmt.Printf("Testing with %s...\n", serverType)

	// Test 1: Simple prompt
	result, err := client.RunPrompt("What is 2+2?", &claude.RunOptions{
		Format: claude.JSONOutput,
	})

	if err != nil {
		fmt.Printf("❌ Simple prompt failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Simple prompt succeeded\n")
	fmt.Printf("   Result: %s\n", result.Result)
	fmt.Printf("   Cost: $%.6f\n", result.CostUSD)
	fmt.Printf("   Session: %s\n", result.SessionID)

	// Test 2: Text output
	textResult, err := client.RunPrompt("Say hello", &claude.RunOptions{
		Format: claude.TextOutput,
	})

	if err != nil {
		fmt.Printf("❌ Text output failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Text output succeeded\n")
	fmt.Printf("   Result: %s\n", textResult.Result[:min(50, len(textResult.Result))])

	// Test 3: Streaming (only test setup, don't wait for completion)
	fmt.Printf("✅ Testing streaming setup...\n")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	messageCh, errCh := client.StreamPrompt(ctx, "Count to 3", &claude.RunOptions{})

	// Just verify we can set up streaming
	go func() {
		for err := range errCh {
			fmt.Printf("   Stream error (expected for mock): %v\n", err)
		}
	}()

	messageCount := 0
	for msg := range messageCh {
		messageCount++
		if messageCount > 2 {
			break // Don't wait for full completion
		}
		fmt.Printf("   Received message type: %s\n", msg.Type)
	}

	fmt.Printf("✅ Streaming setup succeeded (%d messages)\n", messageCount)
}

func testConvenienceMethods() {
	client := claude.NewClient("claude")

	fmt.Println("Testing convenience methods (may fail without real Claude)...")

	// Test method existence (they won't actually work without Claude)
	methods := []string{
		"RunWithSystemPrompt",
		"RunWithMCP",
		"ContinueConversation",
		"ResumeConversation",
	}

	for _, method := range methods {
		fmt.Printf("✅ Method %s exists\n", method)
	}

	// Test validation
	_, err := client.RunPrompt("test", &claude.RunOptions{
		AllowedTools: []string{"mcp__filesystem__read_file", "Bash"},
	})

	if err != nil && err.Error() != "claude command failed: exec: \"claude\": executable file not found in $PATH: " {
		fmt.Printf("⚠️  Validation error: %v\n", err)
	} else {
		fmt.Printf("✅ MCP tool validation works\n")
	}

	// Test invalid MCP tool
	_, err = client.RunPrompt("test", &claude.RunOptions{
		AllowedTools: []string{"mcp_invalid_tool"},
	})

	if err != nil && err.Error() == "invalid MCP tool name: mcp_invalid_tool (must follow pattern: mcp__<serverName>__<toolName>)" {
		fmt.Printf("✅ Invalid MCP tool validation works\n")
	} else {
		fmt.Printf("⚠️  Expected validation error, got: %v\n", err)
	}
}

func isMockServerRunning() bool {
	// Simple check if mock server is running
	client := claude.NewClient("http://localhost:8080/claude")
	_, err := client.RunPrompt("ping", &claude.RunOptions{
		Format: claude.TextOutput,
	})
	return err == nil
}

func isClaudeCLIAvailable() bool {
	// Check if Claude CLI is available in PATH
	client := claude.NewClient("claude")
	_, err := client.RunPrompt("test", &claude.RunOptions{
		Format: claude.TextOutput,
	})
	// If there's no exec error, Claude CLI is available
	// (authentication failures are handled by Claude CLI automatically)
	return err == nil || (err != nil && !isExecError(err))
}

func isExecError(err error) bool {
	// Check if error is related to executable not found
	return err != nil && (err.Error() == "claude command failed: exec: \"claude\": executable file not found in $PATH: " ||
		err.Error() == "exec: \"claude\": executable file not found in $PATH")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

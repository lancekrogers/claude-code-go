package integration

import (
	"strings"
	"testing"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
	"github.com/lancekrogers/claude-code-go/test/utils"
)

func TestBasicPrompt(t *testing.T) {
	utils.SkipIfNoClaudeCLI(t)

	client := utils.NewTestClient(t)

	result, err := client.RunPrompt("What is 2+2?", &claude.RunOptions{
		Format: claude.JSONOutput,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Result == "" {
		t.Error("Expected non-empty result")
	}

	if result.SessionID == "" {
		t.Error("Expected non-empty session ID")
	}

	// Validate cost tracking
	if result.CostUSD < 0 {
		t.Errorf("Expected non-negative cost, got %f", result.CostUSD)
	}

	// Validate timing
	if result.DurationMS <= 0 {
		t.Errorf("Expected positive duration, got %d", result.DurationMS)
	}

	t.Logf("Result: %s", result.Result)
	t.Logf("Cost: $%.6f", result.CostUSD)
	t.Logf("Session: %s", result.SessionID)
}

func TestTextOutput(t *testing.T) {
	utils.SkipIfNoClaudeCLI(t)

	client := utils.NewTestClient(t)

	result, err := client.RunPrompt("Say hello", &claude.RunOptions{
		Format: claude.TextOutput,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Result == "" {
		t.Error("Expected non-empty result")
	}

	// Text output should contain a greeting
	if !strings.Contains(strings.ToLower(result.Result), "hello") {
		t.Errorf("Expected result to contain 'hello', got: %s", result.Result)
	}
}

func TestSystemPrompt(t *testing.T) {
	utils.SkipIfNoClaudeCLI(t)

	client := utils.NewTestClient(t)

	result, err := client.RunWithSystemPrompt(
		"What language am I?",
		"You are a helpful assistant that always responds in Spanish.",
		nil,
	)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Result == "" {
		t.Error("Expected non-empty result")
	}

	t.Logf("System prompt result: %s", result.Result)
}

func TestConvenienceMethods(t *testing.T) {
	utils.SkipIfNoClaudeCLI(t)

	client := utils.NewTestClient(t)

	t.Run("RunWithSystemPrompt", func(t *testing.T) {
		result, err := client.RunWithSystemPrompt(
			"Count to 3",
			"Always respond with just numbers, no other text.",
			nil,
		)
		if err != nil {
			t.Fatalf("RunWithSystemPrompt failed: %v", err)
		}
		if result.Result == "" {
			t.Error("Expected non-empty result")
		}
	})

	// Note: We can't easily test ContinueConversation and ResumeConversation
	// without a persistent session, so we'll test that they don't crash
	t.Run("ContinueConversation", func(t *testing.T) {
		_, err := client.ContinueConversation("Hello")
		// This might fail if there's no previous conversation, that's OK
		// We're just testing that the method exists and doesn't panic
		if err != nil {
			t.Logf("ContinueConversation failed as expected: %v", err)
		}
	})
}

func TestValidation(t *testing.T) {
	client := utils.NewTestClient(t)

	t.Run("ValidMCPTool", func(t *testing.T) {
		// Skip MCP tool testing with real Claude CLI unless MCP config is provided
		if !utils.IsMockServerMode() {
			t.Skip("Skipping MCP tool test with real Claude CLI (would require MCP configuration)")
		}

		_, err := client.RunPrompt("test", &claude.RunOptions{
			Format:       claude.JSONOutput,
			AllowedTools: []string{"mcp__filesystem__read_file", "Bash"},
		})

		// This should not fail due to validation (may fail for other reasons like mock server)
		if err != nil && strings.Contains(err.Error(), "invalid MCP tool name") {
			t.Errorf("Valid MCP tool failed validation: %v", err)
		} else {
			t.Logf("Valid MCP tool test passed (err: %v)", err)
		}
	})

	t.Run("InvalidMCPTool", func(t *testing.T) {
		_, err := client.RunPrompt("test", &claude.RunOptions{
			Format:       claude.JSONOutput,
			AllowedTools: []string{"mcp__invalid"},
		})

		// This should fail due to validation
		if err != nil && strings.Contains(err.Error(), "invalid MCP tool name") {
			t.Logf("Validation correctly caught invalid MCP tool: %v", err)
		} else {
			// In mock server mode, this might not trigger validation
			if utils.IsMockServerMode() {
				t.Logf("Mock server mode - validation may not trigger (err: %v)", err)
			} else {
				t.Errorf("Expected validation error for invalid MCP tool name, got: %v", err)
			}
		}
	})
}

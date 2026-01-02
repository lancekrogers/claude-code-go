package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

// NewTestClient creates a Claude client for testing
func NewTestClient(t *testing.T) *claude.ClaudeClient {
	return claude.NewClient(GetTestClaudePath(t))
}

// GetTestClaudePath returns the path to Claude CLI for testing
func GetTestClaudePath(t *testing.T) string {
	if path := os.Getenv("CLAUDE_CODE_PATH"); path != "" {
		return path
	}

	// Check if mock server is preferred
	if os.Getenv("USE_MOCK_SERVER") == "1" {
		return CreateMockClaudeScript(t)
	}

	return "claude" // Default Claude Code CLI
}

// GetMockServerPath returns the path to the mock server endpoint
func GetMockServerPath() string {
	port := os.Getenv("MOCK_SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	return "http://localhost:" + port + "/claude"
}

// SkipIfNoClaudeCLI skips the test if Claude Code CLI is not available
func SkipIfNoClaudeCLI(t *testing.T) {
	if os.Getenv("USE_MOCK_SERVER") == "1" {
		return // Mock server tests don't need Claude CLI
	}

	claudePath := GetTestClaudePath(t)
	if _, err := exec.LookPath(claudePath); err != nil {
		t.Skipf("Skipping test: Claude Code CLI not found at '%s'. Install Claude Code CLI or use mock server with USE_MOCK_SERVER=1", claudePath)
	}

	// Test if Claude CLI is working (it will handle auth automatically)
	cmd := exec.Command(claudePath, "--help")
	if err := cmd.Run(); err != nil {
		t.Skip("Skipping test: Claude Code CLI not working. Please ensure it's properly installed.")
	}
}

// RequireClaudeCLI fails the test if Claude Code CLI is not available
func RequireClaudeCLI(t *testing.T) {
	if os.Getenv("USE_MOCK_SERVER") == "1" {
		return // Mock server tests don't need Claude CLI
	}

	claudePath := GetTestClaudePath(t)
	if _, err := exec.LookPath(claudePath); err != nil {
		t.Fatalf("Test requires Claude Code CLI at '%s'. Install from https://docs.anthropic.com/en/docs/claude-code/getting-started", claudePath)
	}

	// Test if Claude CLI is working (it will handle auth automatically)
	cmd := exec.Command(claudePath, "--help")
	if err := cmd.Run(); err != nil {
		t.Fatal("Claude Code CLI not working. Please ensure it's properly installed.")
	}
}

// IsIntegrationTest returns true if integration tests should run
func IsIntegrationTest() bool {
	return os.Getenv("INTEGRATION_TESTS") == "1"
}

// GetTestTimeout returns the test timeout duration
func GetTestTimeout() string {
	if timeout := os.Getenv("TEST_TIMEOUT"); timeout != "" {
		return timeout
	}
	return "30s"
}

// IsMockServerMode returns true if using mock server
func IsMockServerMode() bool {
	return os.Getenv("USE_MOCK_SERVER") == "1"
}

// CreateMockClaudeScript creates a shell script that forwards to the mock server
func CreateMockClaudeScript(t *testing.T) string {
	tempDir := t.TempDir()
	mockScript := filepath.Join(tempDir, "mock-claude.sh")

	content := `#!/bin/bash
# Mock Claude CLI that forwards to HTTP server
PORT=${MOCK_SERVER_PORT:-8080}
curl -s -X POST "http://localhost:$PORT/claude" -d "$*"
`

	if err := os.WriteFile(mockScript, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}

	return mockScript
}

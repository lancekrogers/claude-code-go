package dangerous

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

func TestNewDangerousClient_RequiresEnvironmentVariable(t *testing.T) {
	// Clear environment
	originalEnv := os.Getenv("CLAUDE_ENABLE_DANGEROUS")
	defer os.Setenv("CLAUDE_ENABLE_DANGEROUS", originalEnv)
	os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")

	_, err := NewDangerousClient("claude")
	if err == nil {
		t.Error("Expected error when CLAUDE_ENABLE_DANGEROUS is not set")
	}

	expectedMsg := "dangerous client requires CLAUDE_ENABLE_DANGEROUS=i-accept-all-risks"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestNewDangerousClient_BlocksProduction(t *testing.T) {
	// Set required dangerous env var
	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")
	defer os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")

	testCases := []struct {
		envVar string
		value  string
	}{
		{"NODE_ENV", "production"},
		{"GO_ENV", "production"},
		{"ENVIRONMENT", "production"},
		{"ENVIRONMENT", "prod"},
	}

	for _, tc := range testCases {
		t.Run(tc.envVar+"="+tc.value, func(t *testing.T) {
			original := os.Getenv(tc.envVar)
			defer os.Setenv(tc.envVar, original)

			os.Setenv(tc.envVar, tc.value)

			_, err := NewDangerousClient("claude")
			if err == nil {
				t.Errorf("Expected error when %s=%s", tc.envVar, tc.value)
			}

			if !strings.Contains(err.Error(), "forbidden in production") {
				t.Errorf("Expected production error, got: %v", err)
			}
		})
	}
}

func TestNewDangerousClient_AllowsDevelopment(t *testing.T) {
	// Set required environment variables
	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")
	os.Setenv("NODE_ENV", "development")
	defer os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")
	defer os.Unsetenv("NODE_ENV")

	client, err := NewDangerousClient("claude")
	if err != nil {
		t.Errorf("Expected no error in development, got: %v", err)
	}

	if client == nil {
		t.Error("Expected client to be created")
	}

	if !client.securityGate.confirmed {
		t.Error("Expected security gate to be confirmed")
	}

	if !client.securityGate.productionCheck {
		t.Error("Expected production check to pass")
	}
}

func TestDangerousClient_SecurityGateChecks(t *testing.T) {
	// Create client with proper setup
	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")
	os.Setenv("NODE_ENV", "development")
	defer os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")
	defer os.Unsetenv("NODE_ENV")

	client, err := NewDangerousClient("mock-claude")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test that methods check security gate
	envVars := map[string]string{"TEST": "value"}
	err = client.SET_ENVIRONMENT_VARIABLES(envVars)
	if err != nil {
		t.Errorf("Expected SET_ENVIRONMENT_VARIABLES to work with confirmed gate: %v", err)
	}

	err = client.ENABLE_MCP_DEBUG()
	if err != nil {
		t.Errorf("Expected ENABLE_MCP_DEBUG to work with confirmed gate: %v", err)
	}

	// Test that warnings are tracked
	warnings := client.GetSecurityWarnings()
	if len(warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d: %v", len(warnings), warnings)
	}

	// Test reset
	client.ResetDangerousSettings()
	warnings = client.GetSecurityWarnings()
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings after reset, got %d: %v", len(warnings), warnings)
	}
}

func TestDangerousClient_EnvironmentVariableValidation(t *testing.T) {
	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")
	os.Setenv("NODE_ENV", "development")
	defer os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")
	defer os.Unsetenv("NODE_ENV")

	client, err := NewDangerousClient("mock-claude")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test setting sensitive environment variables (should work but warn)
	sensitiveVars := map[string]string{
		"API_PASSWORD": "secret",
		"SECRET_KEY":   "key123",
		"ACCESS_TOKEN": "token456",
		"PATH":         "/custom/path",
	}

	err = client.SET_ENVIRONMENT_VARIABLES(sensitiveVars)
	if err != nil {
		t.Errorf("Expected sensitive vars to be set with warnings: %v", err)
	}

	// Verify variables were stored
	if len(client.envVars) != 4 {
		t.Errorf("Expected 4 env vars stored, got %d", len(client.envVars))
	}

	if client.envVars["API_PASSWORD"] != "secret" {
		t.Errorf("Expected env var to be stored correctly")
	}
}

func TestSecurityGate_UnconfirmedGateBlocks(t *testing.T) {
	// Create a client with unconfirmed gate (simulate internal state)
	client := &DangerousClient{
		ClaudeClient: claude.NewClient("mock-claude"),
		securityGate: &SecurityGate{confirmed: false},
		envVars:      make(map[string]string),
	}

	// Test that operations fail with unconfirmed gate
	_, err := client.BYPASS_ALL_PERMISSIONS("test", nil)
	if err == nil || err.Error() != "security gate not confirmed" {
		t.Errorf("Expected security gate error, got: %v", err)
	}

	err = client.SET_ENVIRONMENT_VARIABLES(map[string]string{"TEST": "value"})
	if err == nil || err.Error() != "security gate not confirmed" {
		t.Errorf("Expected security gate error, got: %v", err)
	}

	err = client.ENABLE_MCP_DEBUG()
	if err == nil || err.Error() != "security gate not confirmed" {
		t.Errorf("Expected security gate error, got: %v", err)
	}
}

func TestDangerousClient_BYPASS_ALL_PERMISSIONS_Variants(t *testing.T) {
	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")
	os.Setenv("NODE_ENV", "development")
	defer os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")
	defer os.Unsetenv("NODE_ENV")

	client, err := NewDangerousClient("mock-claude")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("BYPASS_ALL_PERMISSIONS with nil opts", func(t *testing.T) {
		// This will fail because mock-claude doesn't exist, but we can verify
		// it properly handles nil options
		_, err := client.BYPASS_ALL_PERMISSIONS("test prompt", nil)
		if err == nil {
			t.Error("Expected error from non-existent binary")
		}
	})

	t.Run("BYPASS_ALL_PERMISSIONS with options", func(t *testing.T) {
		opts := &claude.RunOptions{
			Verbose: true,
		}
		_, err := client.BYPASS_ALL_PERMISSIONS("test prompt", opts)
		if err == nil {
			t.Error("Expected error from non-existent binary")
		}
	})
}

func TestDangerousClient_BYPASS_ALL_PERMISSIONS_CTX(t *testing.T) {
	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")
	os.Setenv("NODE_ENV", "development")
	defer os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")
	defer os.Unsetenv("NODE_ENV")

	client, err := NewDangerousClient("mock-claude")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("with context and nil opts", func(t *testing.T) {
		ctx := context.Background()
		_, err := client.BYPASS_ALL_PERMISSIONS_CTX(ctx, "test", nil)
		if err == nil {
			t.Error("Expected error from non-existent binary")
		}
	})

	t.Run("with canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		_, err := client.BYPASS_ALL_PERMISSIONS_CTX(ctx, "test", nil)
		if err == nil {
			t.Error("Expected error from canceled context")
		}
	})

	t.Run("with unconfirmed security gate", func(t *testing.T) {
		unconfirmedClient := &DangerousClient{
			ClaudeClient: claude.NewClient("mock-claude"),
			securityGate: &SecurityGate{confirmed: false},
			envVars:      make(map[string]string),
		}
		_, err := unconfirmedClient.BYPASS_ALL_PERMISSIONS_CTX(context.Background(), "test", nil)
		if err == nil || err.Error() != "security gate not confirmed" {
			t.Errorf("Expected security gate error, got: %v", err)
		}
	})
}

func TestDangerousClient_DANGEROUS_RunWithEnvironment(t *testing.T) {
	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")
	os.Setenv("NODE_ENV", "development")
	defer os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")
	defer os.Unsetenv("NODE_ENV")

	client, err := NewDangerousClient("mock-claude")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("with environment variables", func(t *testing.T) {
		envVars := map[string]string{
			"CUSTOM_VAR": "value",
		}
		_, err := client.DANGEROUS_RunWithEnvironment("test", nil, envVars)
		if err == nil {
			t.Error("Expected error from non-existent binary")
		}
		// Verify env vars were set
		if client.envVars["CUSTOM_VAR"] != "value" {
			t.Error("Expected environment variable to be set")
		}
	})

	t.Run("with unconfirmed gate", func(t *testing.T) {
		unconfirmedClient := &DangerousClient{
			ClaudeClient: claude.NewClient("mock-claude"),
			securityGate: &SecurityGate{confirmed: false},
			envVars:      make(map[string]string),
		}
		_, err := unconfirmedClient.DANGEROUS_RunWithEnvironment("test", nil, map[string]string{})
		if err == nil || err.Error() != "security gate not confirmed" {
			t.Errorf("Expected security gate error, got: %v", err)
		}
	})
}

func TestDangerousClient_DANGEROUS_RunWithEnvironmentCtx(t *testing.T) {
	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")
	os.Setenv("NODE_ENV", "development")
	defer os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")
	defer os.Unsetenv("NODE_ENV")

	client, err := NewDangerousClient("mock-claude")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("with context and environment", func(t *testing.T) {
		ctx := context.Background()
		envVars := map[string]string{
			"TEST_VAR": "test_value",
		}
		_, err := client.DANGEROUS_RunWithEnvironmentCtx(ctx, "test", nil, envVars)
		if err == nil {
			t.Error("Expected error from non-existent binary")
		}
	})

	t.Run("with canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := client.DANGEROUS_RunWithEnvironmentCtx(ctx, "test", nil, map[string]string{})
		if err == nil {
			t.Error("Expected error from canceled context")
		}
	})
}

func TestDangerousClient_GetSecurityWarnings_MCPDebug(t *testing.T) {
	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")
	os.Setenv("NODE_ENV", "development")
	defer os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")
	defer os.Unsetenv("NODE_ENV")

	client, err := NewDangerousClient("mock-claude")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Initially no warnings
	warnings := client.GetSecurityWarnings()
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings initially, got %d", len(warnings))
	}

	// Enable MCP debug
	err = client.ENABLE_MCP_DEBUG()
	if err != nil {
		t.Fatalf("Failed to enable MCP debug: %v", err)
	}

	warnings = client.GetSecurityWarnings()
	if len(warnings) != 1 {
		t.Errorf("Expected 1 warning after MCP debug, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "MCP debug") {
		t.Errorf("Expected MCP debug warning, got: %s", warnings[0])
	}

	// Add env vars
	err = client.SET_ENVIRONMENT_VARIABLES(map[string]string{"TEST": "value"})
	if err != nil {
		t.Fatalf("Failed to set env vars: %v", err)
	}

	warnings = client.GetSecurityWarnings()
	if len(warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(warnings))
	}
}

func TestDangerousClient_runWithDangerousFlags_MCPDebug(t *testing.T) {
	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")
	os.Setenv("NODE_ENV", "development")
	defer os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")
	defer os.Unsetenv("NODE_ENV")

	client, err := NewDangerousClient("mock-claude")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Enable MCP debug
	err = client.ENABLE_MCP_DEBUG()
	if err != nil {
		t.Fatalf("Failed to enable MCP debug: %v", err)
	}

	// Run with MCP debug enabled (will fail due to mock binary)
	_, err = client.BYPASS_ALL_PERMISSIONS("test", nil)
	if err == nil {
		t.Error("Expected error from mock binary")
	}
	// At least the mcpDebug flag was set and will be included in args
	if !client.mcpDebug {
		t.Error("Expected mcpDebug to be true")
	}
}

func TestDangerousClient_OptionsValidation(t *testing.T) {
	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")
	os.Setenv("NODE_ENV", "development")
	defer os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")
	defer os.Unsetenv("NODE_ENV")

	client, err := NewDangerousClient("mock-claude")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("with various options", func(t *testing.T) {
		opts := &claude.RunOptions{
			Format:       claude.JSONOutput,
			Verbose:      true,
			SystemPrompt: "Test system prompt",
		}
		_, err := client.BYPASS_ALL_PERMISSIONS("test", opts)
		// Will error because binary doesn't exist
		if err == nil {
			t.Error("Expected error from non-existent binary")
		}
	})
}

func TestNewDangerousClient_AllProductionEnvChecks(t *testing.T) {
	originalDangerous := os.Getenv("CLAUDE_ENABLE_DANGEROUS")
	defer func() {
		if originalDangerous != "" {
			os.Setenv("CLAUDE_ENABLE_DANGEROUS", originalDangerous)
		} else {
			os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")
		}
	}()

	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")

	// Test each production environment variable individually
	productionEnvVars := []struct {
		envVar string
		value  string
	}{
		{"NODE_ENV", "production"},
		{"GO_ENV", "production"},
		{"ENVIRONMENT", "production"},
		{"ENVIRONMENT", "prod"},
	}

	for _, tc := range productionEnvVars {
		t.Run(tc.envVar+"="+tc.value, func(t *testing.T) {
			// Clear all production env vars first
			os.Unsetenv("NODE_ENV")
			os.Unsetenv("GO_ENV")
			os.Unsetenv("ENVIRONMENT")

			os.Setenv(tc.envVar, tc.value)
			defer os.Unsetenv(tc.envVar)

			_, err := NewDangerousClient("claude")
			if err == nil {
				t.Errorf("Expected error when %s=%s", tc.envVar, tc.value)
			}
			if !strings.Contains(err.Error(), tc.envVar) {
				t.Errorf("Expected error to mention %s, got: %v", tc.envVar, err)
			}
		})
	}
}

func TestDangerousClient_EmptyEnvironmentVariables(t *testing.T) {
	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")
	os.Setenv("NODE_ENV", "development")
	defer os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")
	defer os.Unsetenv("NODE_ENV")

	client, err := NewDangerousClient("mock-claude")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Set empty env vars map
	err = client.SET_ENVIRONMENT_VARIABLES(map[string]string{})
	if err != nil {
		t.Errorf("Expected no error for empty env vars: %v", err)
	}

	// Verify no warnings for empty map
	warnings := client.GetSecurityWarnings()
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings for empty env vars, got %d", len(warnings))
	}
}

func TestDangerousClient_ResetClearsAll(t *testing.T) {
	os.Setenv("CLAUDE_ENABLE_DANGEROUS", "i-accept-all-risks")
	os.Setenv("NODE_ENV", "development")
	defer os.Unsetenv("CLAUDE_ENABLE_DANGEROUS")
	defer os.Unsetenv("NODE_ENV")

	client, err := NewDangerousClient("mock-claude")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Set up some dangerous settings
	client.SET_ENVIRONMENT_VARIABLES(map[string]string{"TEST": "value"})
	client.ENABLE_MCP_DEBUG()

	if len(client.GetSecurityWarnings()) != 2 {
		t.Error("Expected 2 warnings before reset")
	}

	// Reset
	client.ResetDangerousSettings()

	// Verify all cleared
	if len(client.envVars) != 0 {
		t.Errorf("Expected empty env vars after reset, got %d", len(client.envVars))
	}
	if client.mcpDebug {
		t.Error("Expected mcpDebug to be false after reset")
	}
	if len(client.GetSecurityWarnings()) != 0 {
		t.Errorf("Expected 0 warnings after reset, got %d", len(client.GetSecurityWarnings()))
	}
}

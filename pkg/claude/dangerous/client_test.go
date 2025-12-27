package dangerous

import (
	"os"
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

			if !containsString(err.Error(), "forbidden in production") {
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

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					findInString(s, substr))))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

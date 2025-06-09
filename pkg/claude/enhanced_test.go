package claude

import (
	"testing"
	"time"
)

func TestPreprocessOptions(t *testing.T) {
	tests := []struct {
		name        string
		opts        *RunOptions
		expectError bool
		errorType   ErrorType
	}{
		{
			name: "Valid options with enhanced permissions",
			opts: &RunOptions{
				AllowedTools: []string{
					"Bash(git log:*)",
					"Write(src/**)",
					"Read",
				},
				ModelAlias: "sonnet",
				Timeout:    30 * time.Second,
			},
			expectError: false,
		},
		{
			name: "Invalid tool permission format",
			opts: &RunOptions{
				AllowedTools: []string{
					"InvalidTool()",
				},
			},
			expectError: true,
			errorType:   ErrorValidation,
		},
		{
			name: "Invalid model alias",
			opts: &RunOptions{
				ModelAlias: "invalid-model",
			},
			expectError: true,
			errorType:   ErrorValidation,
		},
		{
			name: "Negative timeout",
			opts: &RunOptions{
				Timeout: -5 * time.Second,
			},
			expectError: true,
			errorType:   ErrorValidation,
		},
		{
			name:        "Nil options",
			opts:        nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PreprocessOptions(tt.opts)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				
				if claudeErr, ok := err.(*ClaudeError); ok {
					if claudeErr.Type != tt.errorType {
						t.Errorf("Expected error type %v, got %v", tt.errorType, claudeErr.Type)
					}
				} else {
					t.Errorf("Expected ClaudeError but got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestIsValidModelAlias(t *testing.T) {
	tests := []struct {
		alias string
		want  bool
	}{
		{"sonnet", true},
		{"opus", true},
		{"haiku", true},
		{"invalid", false},
		{"", false},
		{"SONNET", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			if got := isValidModelAlias(tt.alias); got != tt.want {
				t.Errorf("isValidModelAlias(%q) = %v, want %v", tt.alias, got, tt.want)
			}
		})
	}
}

func TestIsValidSessionID(t *testing.T) {
	tests := []struct {
		sessionID string
		want      bool
	}{
		{"valid-session-123", true},
		{"", false},
		{"   ", false},
		{"any-non-empty-string", true},
	}

	for _, tt := range tests {
		t.Run(tt.sessionID, func(t *testing.T) {
			if got := isValidSessionID(tt.sessionID); got != tt.want {
				t.Errorf("isValidSessionID(%q) = %v, want %v", tt.sessionID, got, tt.want)
			}
		})
	}
}

func TestRetryPolicy_calculateBackoff(t *testing.T) {
	policy := &RetryPolicy{
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
	}

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 0},
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{10, 5 * time.Second}, // Should cap at MaxDelay
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := policy.calculateBackoff(tt.attempt)
			if got != tt.want {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()
	
	if policy.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries = 3, got %d", policy.MaxRetries)
	}
	
	if policy.BaseDelay != 100*time.Millisecond {
		t.Errorf("Expected BaseDelay = 100ms, got %v", policy.BaseDelay)
	}
	
	if policy.MaxDelay != 5*time.Second {
		t.Errorf("Expected MaxDelay = 5s, got %v", policy.MaxDelay)
	}
	
	if policy.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor = 2.0, got %f", policy.BackoffFactor)
	}
}

func TestBuildArgs_EnhancedFeatures(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		opts   *RunOptions
		want   []string
	}{
		{
			name:   "Model alias takes precedence",
			prompt: "test",
			opts: &RunOptions{
				Model:      "claude-3-sonnet-20240229",
				ModelAlias: "sonnet",
			},
			want: []string{"-p", "test", "--model", "sonnet"},
		},
		{
			name:   "Enhanced tool permissions",
			prompt: "test",
			opts: &RunOptions{
				AllowedTools: []string{
					"Bash(git log:*)",
					"Write(src/**)",
				},
			},
			want: []string{"-p", "test", "--allowedTools", "Bash(git log:*),Write(src/**)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildArgs(tt.prompt, tt.opts)
			
			// Check that all expected args are present
			for _, expectedArg := range tt.want {
				found := false
				for _, gotArg := range got {
					if gotArg == expectedArg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected argument %q not found in %v", expectedArg, got)
				}
			}
		})
	}
}

func TestValidateOptions(t *testing.T) {
	// Test that ValidateOptions is just an alias for PreprocessOptions
	opts := &RunOptions{
		AllowedTools: []string{"InvalidTool()"},
	}
	
	err := ValidateOptions(opts)
	if err == nil {
		t.Error("Expected validation error but got none")
	}
	
	if claudeErr, ok := err.(*ClaudeError); ok {
		if claudeErr.Type != ErrorValidation {
			t.Errorf("Expected ErrorValidation, got %v", claudeErr.Type)
		}
	} else {
		t.Errorf("Expected ClaudeError but got %T", err)
	}
}

// Integration test for RunPromptEnhanced (mock-based)
func TestRunPromptEnhanced_Integration(t *testing.T) {
	// Save original execCommand
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Mock successful command execution
	jsonOutput := `{"type":"result","result":"Enhanced test passed","cost_usd":0.001,"duration_ms":500,"num_turns":1,"session_id":"test-123"}`
	
	// Use the existing mock pattern - expect specific args for enhanced features
	expectedArgs := []string{"-p", "Test enhanced features", "--output-format", "json", "--allowedTools", "Bash(git log:*),Read", "--model", "sonnet"}
	execCommand = mockExecCommandContext(t, expectedArgs, jsonOutput, 0)

	client := NewClient("claude")
	opts := &RunOptions{
		Format: JSONOutput,
		AllowedTools: []string{
			"Bash(git log:*)",
			"Read",
		},
		ModelAlias: "sonnet",
		Timeout:    30 * time.Second,
	}

	result, err := client.RunPromptEnhanced("Test enhanced features", opts)
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
		return
	}

	if result.Result != "Enhanced test passed" {
		t.Errorf("Expected result 'Enhanced test passed', got %q", result.Result)
	}

	if result.CostUSD != 0.001 {
		t.Errorf("Expected cost 0.001, got %f", result.CostUSD)
	}
}
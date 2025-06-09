package claude

import (
	"testing"
)

func TestErrorType_String(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		want      string
	}{
		{ErrorAuthentication, "authentication"},
		{ErrorRateLimit, "rate_limit"},
		{ErrorPermission, "permission"},
		{ErrorCommand, "command"},
		{ErrorNetwork, "network"},
		{ErrorMCP, "mcp"},
		{ErrorValidation, "validation"},
		{ErrorTimeout, "timeout"},
		{ErrorSession, "session"},
		{ErrorUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.errorType.String(); got != tt.want {
				t.Errorf("ErrorType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorType_IsRetryable(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		want      bool
	}{
		{ErrorRateLimit, true},
		{ErrorNetwork, true},
		{ErrorTimeout, true},
		{ErrorMCP, true}, // Generally retryable, but depends on specific error
		{ErrorAuthentication, false},
		{ErrorPermission, false},
		{ErrorValidation, false},
		{ErrorCommand, false},
		{ErrorSession, false},
		{ErrorUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.errorType.String(), func(t *testing.T) {
			if got := tt.errorType.IsRetryable(); got != tt.want {
				t.Errorf("ErrorType.IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClaudeError_Error(t *testing.T) {
	tests := []struct {
		name  string
		err   *ClaudeError
		want  string
	}{
		{
			name: "Error with code",
			err: &ClaudeError{
				Type:    ErrorAuthentication,
				Message: "Invalid API key",
				Code:    401,
			},
			want: "claude error (authentication, code=401): Invalid API key",
		},
		{
			name: "Error without code",
			err: &ClaudeError{
				Type:    ErrorValidation,
				Message: "Invalid input",
			},
			want: "claude error (validation): Invalid input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ClaudeError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClaudeError_IsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  *ClaudeError
		want bool
	}{
		{
			name: "Rate limit error",
			err: &ClaudeError{
				Type:    ErrorRateLimit,
				Message: "Rate limit exceeded",
			},
			want: true,
		},
		{
			name: "MCP connection error",
			err: &ClaudeError{
				Type:    ErrorMCP,
				Message: "Connection refused to MCP server",
			},
			want: true,
		},
		{
			name: "MCP configuration error",
			err: &ClaudeError{
				Type:    ErrorMCP,
				Message: "Invalid MCP configuration file",
			},
			want: false,
		},
		{
			name: "Authentication error",
			err: &ClaudeError{
				Type:    ErrorAuthentication,
				Message: "Invalid API key",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.IsRetryable(); got != tt.want {
				t.Errorf("ClaudeError.IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClaudeError_RetryDelay(t *testing.T) {
	tests := []struct {
		name string
		err  *ClaudeError
		want int
	}{
		{
			name: "Rate limit with retry-after",
			err: &ClaudeError{
				Type:    ErrorRateLimit,
				Message: "Rate limit exceeded",
				Details: map[string]interface{}{
					"retry_after": 120,
				},
			},
			want: 120,
		},
		{
			name: "Rate limit without retry-after",
			err: &ClaudeError{
				Type:    ErrorRateLimit,
				Message: "Rate limit exceeded",
			},
			want: 60,
		},
		{
			name: "Network error",
			err: &ClaudeError{
				Type:    ErrorNetwork,
				Message: "Connection failed",
			},
			want: 5,
		},
		{
			name: "MCP connection error",
			err: &ClaudeError{
				Type:    ErrorMCP,
				Message: "Connection refused",
			},
			want: 3,
		},
		{
			name: "MCP config error",
			err: &ClaudeError{
				Type:    ErrorMCP,
				Message: "Invalid configuration",
			},
			want: 0,
		},
		{
			name: "Authentication error",
			err: &ClaudeError{
				Type:    ErrorAuthentication,
				Message: "Invalid API key",
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.RetryDelay(); got != tt.want {
				t.Errorf("ClaudeError.RetryDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	tests := []struct {
		name     string
		stderr   string
		exitCode int
		wantType ErrorType
		wantMsg  string
	}{
		{
			name:     "Authentication error",
			stderr:   "Error: Invalid API key provided",
			exitCode: 401,
			wantType: ErrorAuthentication,
			wantMsg:  "Authentication failed - check ANTHROPIC_API_KEY environment variable",
		},
		{
			name:     "Rate limit error",
			stderr:   "Error: Rate limit exceeded. Please try again later.",
			exitCode: 429,
			wantType: ErrorRateLimit,
			wantMsg:  "Rate limit exceeded - please wait before retrying",
		},
		{
			name:     "Permission error",
			stderr:   "Error: Tool 'Bash' not allowed by current permissions",
			exitCode: 1,
			wantType: ErrorPermission,
			wantMsg:  "Tool usage not permitted - check allowed/disallowed tools configuration",
		},
		{
			name:     "Network error",
			stderr:   "Error: Connection timeout while connecting to API",
			exitCode: 1,
			wantType: ErrorNetwork,
			wantMsg:  "Network connectivity issue",
		},
		{
			name:     "MCP connection error",
			stderr:   "Error: Failed to connect to MCP server: connection refused",
			exitCode: 1,
			wantType: ErrorMCP,
			wantMsg:  "MCP server error",
		},
		{
			name:     "MCP config error",
			stderr:   "Error: Invalid MCP configuration file format",
			exitCode: 1,
			wantType: ErrorMCP,
			wantMsg:  "MCP server error",
		},
		{
			name:     "Timeout error",
			stderr:   "Error: Operation timed out after 30 seconds",
			exitCode: 1,
			wantType: ErrorTimeout,
			wantMsg:  "Operation timed out",
		},
		{
			name:     "Session error",
			stderr:   "Error: Session not found: invalid session ID",
			exitCode: 1,
			wantType: ErrorSession,
			wantMsg:  "Session management error",
		},
		{
			name:     "Validation error",
			stderr:   "Error: Invalid argument '--invalid-flag'",
			exitCode: 1,
			wantType: ErrorValidation,
			wantMsg:  "Input validation failed",
		},
		{
			name:     "Generic command error",
			stderr:   "Error: Unknown command failed",
			exitCode: 1,
			wantType: ErrorCommand,
			wantMsg:  "Error: Unknown command failed",
		},
		{
			name:     "Empty stderr",
			stderr:   "",
			exitCode: 1,
			wantType: ErrorCommand,
			wantMsg:  "Command execution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseError(tt.stderr, tt.exitCode)
			
			if err.Type != tt.wantType {
				t.Errorf("ParseError() Type = %v, want %v", err.Type, tt.wantType)
			}
			
			if err.Message != tt.wantMsg {
				t.Errorf("ParseError() Message = %v, want %v", err.Message, tt.wantMsg)
			}
			
			if err.Code != tt.exitCode {
				t.Errorf("ParseError() Code = %v, want %v", err.Code, tt.exitCode)
			}
			
			// Check that stderr is preserved in details
			if stderr, exists := err.Details["stderr"]; exists {
				if stderr != tt.stderr {
					t.Errorf("ParseError() preserved stderr = %v, want %v", stderr, tt.stderr)
				}
			}
		})
	}
}

func TestExtractRetryAfter(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   int
	}{
		{
			name:   "Retry after seconds",
			stderr: "Rate limit exceeded. Please retry after 60 seconds.",
			want:   60,
		},
		{
			name:   "Retry-after header format",
			stderr: "HTTP 429: retry-after: 120",
			want:   120,
		},
		{
			name:   "Wait format",
			stderr: "Please wait 30 seconds before trying again",
			want:   30,
		},
		{
			name:   "Try again in format",
			stderr: "Error: Try again in 45 seconds",
			want:   45,
		},
		{
			name:   "No retry information",
			stderr: "Rate limit exceeded",
			want:   0,
		},
		{
			name:   "Case insensitive",
			stderr: "RETRY AFTER 90 SECONDS",
			want:   90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractRetryAfter(tt.stderr); got != tt.want {
				t.Errorf("extractRetryAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewClaudeError(t *testing.T) {
	err := NewClaudeError(ErrorValidation, "Test error")
	
	if err.Type != ErrorValidation {
		t.Errorf("NewClaudeError() Type = %v, want %v", err.Type, ErrorValidation)
	}
	
	if err.Message != "Test error" {
		t.Errorf("NewClaudeError() Message = %v, want %v", err.Message, "Test error")
	}
	
	if err.Details == nil {
		t.Error("NewClaudeError() Details should be initialized")
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("Invalid field", "test_field", "invalid_value")
	
	if err.Type != ErrorValidation {
		t.Errorf("NewValidationError() Type = %v, want %v", err.Type, ErrorValidation)
	}
	
	if err.Message != "Invalid field" {
		t.Errorf("NewValidationError() Message = %v, want %v", err.Message, "Invalid field")
	}
	
	if field, exists := err.Details["field"]; !exists || field != "test_field" {
		t.Errorf("NewValidationError() field detail = %v, want %v", field, "test_field")
	}
	
	if value, exists := err.Details["value"]; !exists || value != "invalid_value" {
		t.Errorf("NewValidationError() value detail = %v, want %v", value, "invalid_value")
	}
}

func TestClaudeError_isMCPConnectionError(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{
			name:    "Connection refused",
			message: "Failed to connect: connection refused",
			want:    true,
		},
		{
			name:    "Network timeout",
			message: "MCP server timeout error",
			want:    true,
		},
		{
			name:    "Socket error",
			message: "Socket connection failed",
			want:    true,
		},
		{
			name:    "Configuration error",
			message: "Invalid MCP configuration file",
			want:    false,
		},
		{
			name:    "Permission error",
			message: "MCP server permission denied",
			want:    false,
		},
		{
			name:    "Unknown MCP error",
			message: "Some unknown MCP error",
			want:    true, // Default to retryable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ClaudeError{
				Type:    ErrorMCP,
				Message: tt.message,
			}
			
			if got := err.isMCPConnectionError(); got != tt.want {
				t.Errorf("isMCPConnectionError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name     string
		haystack string
		needles  []string
		want     bool
	}{
		{
			name:     "Found match",
			haystack: "this is a test string",
			needles:  []string{"test", "example"},
			want:     true,
		},
		{
			name:     "No match",
			haystack: "this is a sample string",
			needles:  []string{"test", "example"},
			want:     false,
		},
		{
			name:     "Multiple matches",
			haystack: "this is a test example",
			needles:  []string{"test", "example"},
			want:     true,
		},
		{
			name:     "Empty needles",
			haystack: "this is a test string",
			needles:  []string{},
			want:     false,
		},
		{
			name:     "Empty haystack",
			haystack: "",
			needles:  []string{"test"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsAny(tt.haystack, tt.needles); got != tt.want {
				t.Errorf("containsAny() = %v, want %v", got, tt.want)
			}
		})
	}
}
package claude

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ErrorType represents the category of error that occurred
type ErrorType int

const (
	// ErrorUnknown represents an unclassified error
	ErrorUnknown ErrorType = iota
	// ErrorAuthentication represents API key or authentication issues
	ErrorAuthentication
	// ErrorRateLimit represents rate limiting from the API
	ErrorRateLimit
	// ErrorPermission represents tool permission denied errors
	ErrorPermission
	// ErrorCommand represents general CLI command errors
	ErrorCommand
	// ErrorNetwork represents network connectivity issues
	ErrorNetwork
	// ErrorMCP represents Model Context Protocol server errors
	ErrorMCP
	// ErrorValidation represents input validation errors
	ErrorValidation
	// ErrorTimeout represents operation timeout errors
	ErrorTimeout
	// ErrorSession represents session management errors
	ErrorSession
)

// String returns the string representation of the error type
func (e ErrorType) String() string {
	switch e {
	case ErrorAuthentication:
		return "authentication"
	case ErrorRateLimit:
		return "rate_limit"
	case ErrorPermission:
		return "permission"
	case ErrorCommand:
		return "command"
	case ErrorNetwork:
		return "network"
	case ErrorMCP:
		return "mcp"
	case ErrorValidation:
		return "validation"
	case ErrorTimeout:
		return "timeout"
	case ErrorSession:
		return "session"
	default:
		return "unknown"
	}
}

// IsRetryable returns true if this error type is generally retryable
func (e ErrorType) IsRetryable() bool {
	switch e {
	case ErrorRateLimit, ErrorNetwork, ErrorTimeout:
		return true
	case ErrorMCP:
		// MCP errors are sometimes retryable (connection issues) but not always (config issues)
		return true
	default:
		return false
	}
}

// ClaudeError represents a structured error from Claude Code operations
type ClaudeError struct {
	Type     ErrorType              `json:"type"`
	Message  string                 `json:"message"`
	Code     int                    `json:"code,omitempty"`
	Details  map[string]interface{} `json:"details,omitempty"`
	Original error                  `json:"-"`
}

// Error implements the error interface
func (e *ClaudeError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("claude error (%s, code=%d): %s", e.Type.String(), e.Code, e.Message)
	}
	return fmt.Sprintf("claude error (%s): %s", e.Type.String(), e.Message)
}

// Unwrap returns the original error for error unwrapping
func (e *ClaudeError) Unwrap() error {
	return e.Original
}

// IsRetryable returns true if this specific error is retryable
func (e *ClaudeError) IsRetryable() bool {
	// Check type-level retryability
	if !e.Type.IsRetryable() {
		return false
	}
	
	// For MCP errors, check if it's a connection vs configuration issue
	if e.Type == ErrorMCP {
		return e.isMCPConnectionError()
	}
	
	return true
}

// isMCPConnectionError determines if an MCP error is due to connection issues (retryable)
// vs configuration issues (not retryable)
func (e *ClaudeError) isMCPConnectionError() bool {
	lowerMsg := strings.ToLower(e.Message)
	
	// Connection-related issues (retryable)
	connectionKeywords := []string{
		"connection", "connect", "timeout", "refused", "unreachable",
		"network", "socket", "pipe", "broken pipe",
	}
	
	for _, keyword := range connectionKeywords {
		if strings.Contains(lowerMsg, keyword) {
			return true
		}
	}
	
	// Configuration-related issues (not retryable)
	configKeywords := []string{
		"configuration", "config", "invalid", "not found", "permission",
		"authentication", "unauthorized", "forbidden",
	}
	
	for _, keyword := range configKeywords {
		if strings.Contains(lowerMsg, keyword) {
			return false
		}
	}
	
	// Default to retryable for unknown MCP errors
	return true
}

// RetryDelay returns the recommended delay before retrying this error
func (e *ClaudeError) RetryDelay() int {
	switch e.Type {
	case ErrorRateLimit:
		// Extract retry-after header value if available
		if retryAfter, exists := e.Details["retry_after"]; exists {
			if seconds, ok := retryAfter.(int); ok {
				return seconds
			}
		}
		return 60 // Default 1 minute for rate limits
	case ErrorNetwork, ErrorTimeout:
		return 5 // 5 seconds for network issues
	case ErrorMCP:
		if e.isMCPConnectionError() {
			return 3 // 3 seconds for MCP connection issues
		}
		return 0 // Don't retry config issues
	default:
		return 0
	}
}

// ParseError analyzes stderr output and exit code to create a structured ClaudeError
// This is exported for use by the dangerous package
func ParseError(stderr string, exitCode int) *ClaudeError {
	stderr = strings.TrimSpace(stderr)
	lowerStderr := strings.ToLower(stderr)
	
	// Authentication errors
	if containsAny(lowerStderr, []string{
		"authentication", "api key", "unauthorized", "401", "forbidden", "403",
		"invalid api key", "missing api key", "anthropic_api_key",
	}) {
		return &ClaudeError{
			Type:    ErrorAuthentication,
			Message: "Authentication failed - check ANTHROPIC_API_KEY environment variable",
			Code:    exitCode,
			Details: map[string]interface{}{
				"suggestion": "Verify your API key is valid and has necessary permissions",
				"stderr":     stderr,
			},
		}
	}
	
	// Rate limit errors
	if containsAny(lowerStderr, []string{
		"rate limit", "too many requests", "429", "quota exceeded",
		"request limit", "usage limit",
	}) {
		retryAfter := extractRetryAfter(stderr)
		details := map[string]interface{}{
			"suggestion": "Wait before retrying or reduce request frequency",
			"stderr":     stderr,
		}
		if retryAfter > 0 {
			details["retry_after"] = retryAfter
		}
		
		return &ClaudeError{
			Type:    ErrorRateLimit,
			Message: "Rate limit exceeded - please wait before retrying",
			Code:    exitCode,
			Details: details,
		}
	}
	
	// Permission errors
	if containsAny(lowerStderr, []string{
		"permission denied", "not allowed", "tool not permitted",
		"access denied", "insufficient permissions", "unauthorized tool",
	}) {
		return &ClaudeError{
			Type:    ErrorPermission,
			Message: "Tool usage not permitted - check allowed/disallowed tools configuration",
			Code:    exitCode,
			Details: map[string]interface{}{
				"suggestion": "Update --allowedTools or permissions settings",
				"stderr":     stderr,
			},
		}
	}
	
	// MCP errors (check before network errors since MCP can have connection issues too)
	if containsAny(lowerStderr, []string{
		"mcp", "model context protocol", "mcp server", "mcp tool",
		"mcp config", "server error", "protocol error",
	}) {
		errorType := ErrorMCP
		suggestion := "Check MCP server configuration and ensure servers are running"
		
		// Determine if it's a connection or configuration issue
		if containsAny(lowerStderr, []string{
			"connection", "connect", "unreachable", "timeout", "refused",
		}) {
			suggestion = "MCP server connection failed - ensure server is running and accessible"
		} else if containsAny(lowerStderr, []string{
			"config", "configuration", "invalid", "not found", "parse",
		}) {
			suggestion = "MCP configuration error - check your MCP config file"
		}
		
		return &ClaudeError{
			Type:    errorType,
			Message: "MCP server error",
			Code:    exitCode,
			Details: map[string]interface{}{
				"suggestion": suggestion,
				"stderr":     stderr,
			},
		}
	}
	
	// Network errors (after MCP check)
	if containsAny(lowerStderr, []string{
		"network", "connection", "timeout", "dns", "unreachable",
		"connection refused", "connection reset", "socket", "no internet",
	}) {
		return &ClaudeError{
			Type:    ErrorNetwork,
			Message: "Network connectivity issue",
			Code:    exitCode,
			Details: map[string]interface{}{
				"suggestion": "Check internet connection and try again",
				"stderr":     stderr,
			},
		}
	}
	
	// Timeout errors
	if containsAny(lowerStderr, []string{
		"timeout", "timed out", "deadline exceeded", "context deadline",
	}) {
		return &ClaudeError{
			Type:    ErrorTimeout,
			Message: "Operation timed out",
			Code:    exitCode,
			Details: map[string]interface{}{
				"suggestion": "Increase timeout or try a simpler operation",
				"stderr":     stderr,
			},
		}
	}
	
	// Session errors
	if containsAny(lowerStderr, []string{
		"session", "session not found", "invalid session", "session expired",
		"resume", "conversation not found",
	}) {
		return &ClaudeError{
			Type:    ErrorSession,
			Message: "Session management error",
			Code:    exitCode,
			Details: map[string]interface{}{
				"suggestion": "Check session ID or start a new conversation",
				"stderr":     stderr,
			},
		}
	}
	
	// Validation errors (client-side)
	if containsAny(lowerStderr, []string{
		"invalid", "validation", "malformed", "bad request", "400",
		"invalid argument", "invalid option", "invalid flag",
	}) {
		return &ClaudeError{
			Type:    ErrorValidation,
			Message: "Input validation failed",
			Code:    exitCode,
			Details: map[string]interface{}{
				"suggestion": "Check command arguments and options",
				"stderr":     stderr,
			},
		}
	}
	
	// Generic command error
	message := "Command execution failed"
	if stderr != "" {
		// Use first line of stderr as the primary message
		lines := strings.Split(stderr, "\n")
		if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
			message = strings.TrimSpace(lines[0])
		}
	}
	
	return &ClaudeError{
		Type:    ErrorCommand,
		Message: message,
		Code:    exitCode,
		Details: map[string]interface{}{
			"stderr": stderr,
		},
	}
}

// containsAny returns true if the haystack contains any of the needles
func containsAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}

// extractRetryAfter attempts to extract retry-after value from error message
func extractRetryAfter(stderr string) int {
	// Look for patterns like "retry after 60 seconds" or "retry-after: 60"
	patterns := []string{
		`retry after (\d+)`,
		`retry-after:?\s*(\d+)`,
		`wait (\d+) seconds`,
		`try again in (\d+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		matches := re.FindStringSubmatch(stderr)
		if len(matches) > 1 {
			if seconds, err := strconv.Atoi(matches[1]); err == nil {
				return seconds
			}
		}
	}
	
	return 0
}

// NewClaudeError creates a new ClaudeError with the specified type and message
// This is exported for use by the dangerous package
func NewClaudeError(errorType ErrorType, message string) *ClaudeError {
	return &ClaudeError{
		Type:    errorType,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// NewValidationError creates a validation error with helpful context
func NewValidationError(message string, field string, value interface{}) *ClaudeError {
	return &ClaudeError{
		Type:    ErrorValidation,
		Message: message,
		Details: map[string]interface{}{
			"field": field,
			"value": value,
		},
	}
}
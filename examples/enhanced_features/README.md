# Enhanced Features Example

This example demonstrates the new enhanced features of the Claude Code Go SDK:

## Features Demonstrated

1. **Enhanced Tool Permissions** - Granular control with `Tool(command:pattern)` syntax
2. **Structured Error Handling** - Intelligent error classification and retry logic  
3. **Retry Policies** - Exponential backoff with rate limit handling
4. **Input Validation** - Pre-execution validation with helpful error messages
5. **Production-Safe Configurations** - Secure permission sets

## Usage

```bash
# Install dependencies
go mod download

# Run the example
go run main.go
```

## Key Features Shown

### Enhanced Tool Permission Syntax
```go
AllowedTools: []string{
    "Bash(git log:*)",      // Allow git log with any arguments
    "Bash(git status)",     // Allow only git status
    "Read",                 // Allow all file reading
    "Write(src/**)",        // Allow writing only to src directory
},
```

### Structured Error Handling
```go
if claudeErr, ok := err.(*claude.ClaudeError); ok {
    fmt.Printf("Error Type: %s\n", claudeErr.Type)
    fmt.Printf("Retryable: %t\n", claudeErr.IsRetryable())
    if suggestion, exists := claudeErr.Details["suggestion"]; exists {
        fmt.Printf("Suggestion: %s\n", suggestion)
    }
}
```

### Intelligent Retry Logic
```go
retryPolicy := &claude.RetryPolicy{
    MaxRetries:    3,
    BaseDelay:     500 * time.Millisecond,
    MaxDelay:      10 * time.Second,
    BackoffFactor: 2.0,
}

result, err := client.RunPromptWithRetryCtx(ctx, prompt, opts, retryPolicy)
```

## Error Types Handled

- `ErrorAuthentication` - API key issues
- `ErrorRateLimit` - Rate limiting with retry-after support
- `ErrorPermission` - Tool permission denials
- `ErrorNetwork` - Connectivity issues
- `ErrorMCP` - Model Context Protocol errors
- `ErrorValidation` - Input validation failures
- `ErrorTimeout` - Operation timeouts
- `ErrorSession` - Session management issues
- `ErrorCommand` - General command failures

This demonstrates production-ready error handling with intelligent retry strategies.
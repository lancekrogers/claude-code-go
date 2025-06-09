package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os/exec"
	"strings"
	"time"
)

// execCommand is a variable to allow mocking of exec.CommandContext for testing
var execCommand = exec.CommandContext

// OutputFormat defines the output format for Claude Code responses
type OutputFormat string

const (
	// TextOutput returns plain text responses
	TextOutput OutputFormat = "text"
	// JSONOutput returns structured JSON responses
	JSONOutput OutputFormat = "json"
	// StreamJSONOutput streams JSON responses as they arrive
	StreamJSONOutput OutputFormat = "stream-json"
)

// ClaudeClient is the main client for interacting with Claude Code
type ClaudeClient struct {
	// BinPath is the path to the Claude Code binary
	BinPath string
	// DefaultOptions are the default options to use for all requests
	DefaultOptions *RunOptions
}

// RunOptions configures how Claude Code is executed
type RunOptions struct {
	// Format specifies the output format (text, json, stream-json)
	Format OutputFormat
	// SystemPrompt overrides the default system prompt
	SystemPrompt string
	// AppendPrompt appends to the default system prompt
	AppendPrompt string
	// MCPConfigPath is the path to the MCP configuration file
	MCPConfigPath string
	// AllowedTools is a list of tools that Claude is allowed to use
	// Supports both legacy format ("Bash") and enhanced format ("Bash(git log:*)")
	AllowedTools []string
	// DisallowedTools is a list of tools that Claude is not allowed to use
	// Supports both legacy format ("Bash") and enhanced format ("Bash(git log:*)")
	DisallowedTools []string
	// PermissionTool is the MCP tool for handling permission prompts
	PermissionTool string
	// ResumeID is the session ID to resume
	ResumeID string
	// Continue indicates whether to continue the most recent conversation
	Continue bool
	// MaxTurns limits the number of agentic turns in non-interactive mode
	MaxTurns int
	// Verbose enables verbose logging
	Verbose bool
	// Model specifies the model to use (full model name)
	Model string
	
	// Enhanced options for 100% CLI support
	// ModelAlias specifies model using alias ("sonnet", "opus", "haiku")
	ModelAlias string
	// Timeout specifies the maximum duration for command execution
	Timeout time.Duration
	// ConfigFile specifies path to Claude configuration file
	ConfigFile string
	// Help shows help information
	Help bool
	// Version shows version information
	Version bool
	// DisableAutoUpdate disables automatic updates
	DisableAutoUpdate bool
	// Theme specifies the UI theme
	Theme string
	
	// Parsed tool permissions (computed from AllowedTools/DisallowedTools)
	// This field is populated automatically and should not be set directly
	ParsedAllowedTools    []ToolPermission `json:"-"`
	ParsedDisallowedTools []ToolPermission `json:"-"`
}

// ClaudeResult represents the structured result from Claude Code
type ClaudeResult struct {
	Type          string  `json:"type"`
	Subtype       string  `json:"subtype,omitempty"`
	Result        string  `json:"result,omitempty"`
	CostUSD       float64 `json:"cost_usd"`
	DurationMS    int64   `json:"duration_ms"`
	DurationAPIMS int64   `json:"duration_api_ms"`
	IsError       bool    `json:"is_error"`
	NumTurns      int     `json:"num_turns"`
	SessionID     string  `json:"session_id"`
}

// Message represents a message from Claude Code in streaming mode
type Message struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype,omitempty"`
	Message   json.RawMessage `json:"message,omitempty"`
	SessionID string          `json:"session_id"`
	// Additional fields for system/result messages
	CostUSD       float64  `json:"cost_usd,omitempty"`
	DurationMS    int64    `json:"duration_ms,omitempty"`
	DurationAPIMS int64    `json:"duration_api_ms,omitempty"`
	IsError       bool     `json:"is_error,omitempty"`
	NumTurns      int      `json:"num_turns,omitempty"`
	Result        string   `json:"result,omitempty"`
	Tools         []string `json:"tools,omitempty"`
	MCPServers    []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"mcp_servers,omitempty"`
}

// validateMCPToolName validates that MCP tool names follow the correct pattern: mcp__<serverName>__<toolName>
func validateMCPToolName(tool string) bool {
	return strings.HasPrefix(tool, "mcp__") && strings.Count(tool, "__") >= 2
}

// validateMCPTools validates all MCP tools in the slice
func validateMCPTools(tools []string) error {
	for _, tool := range tools {
		if strings.HasPrefix(tool, "mcp__") && !validateMCPToolName(tool) {
			return fmt.Errorf("invalid MCP tool name: %s (must follow pattern: mcp__<serverName>__<toolName>)", tool)
		}
	}
	return nil
}

// PreprocessOptions validates and preprocesses RunOptions before execution
func PreprocessOptions(opts *RunOptions) error {
	if opts == nil {
		return nil
	}
	
	// Validate and parse allowed tools
	if len(opts.AllowedTools) > 0 {
		parsed, err := ParseToolPermissions(opts.AllowedTools)
		if err != nil {
			return NewValidationError("Invalid allowed tool permissions", "AllowedTools", opts.AllowedTools)
		}
		opts.ParsedAllowedTools = parsed
		
		// Validate MCP tools in allowed tools
		if err := validateMCPTools(opts.AllowedTools); err != nil {
			return NewValidationError(err.Error(), "AllowedTools", opts.AllowedTools)
		}
	}
	
	// Validate and parse disallowed tools
	if len(opts.DisallowedTools) > 0 {
		parsed, err := ParseToolPermissions(opts.DisallowedTools)
		if err != nil {
			return NewValidationError("Invalid disallowed tool permissions", "DisallowedTools", opts.DisallowedTools)
		}
		opts.ParsedDisallowedTools = parsed
		
		// Validate MCP tools in disallowed tools
		if err := validateMCPTools(opts.DisallowedTools); err != nil {
			return NewValidationError(err.Error(), "DisallowedTools", opts.DisallowedTools)
		}
	}
	
	// Validate model alias
	if opts.ModelAlias != "" {
		if !isValidModelAlias(opts.ModelAlias) {
			return NewValidationError("Invalid model alias", "ModelAlias", opts.ModelAlias)
		}
	}
	
	// Validate timeout
	if opts.Timeout < 0 {
		return NewValidationError("Timeout cannot be negative", "Timeout", opts.Timeout)
	}
	
	// Validate session ID format if provided
	if opts.ResumeID != "" {
		if !isValidSessionID(opts.ResumeID) {
			return NewValidationError("Invalid session ID format", "ResumeID", opts.ResumeID)
		}
	}
	
	return nil
}

// isValidModelAlias checks if the model alias is supported
func isValidModelAlias(alias string) bool {
	validAliases := []string{"sonnet", "opus", "haiku"}
	for _, valid := range validAliases {
		if alias == valid {
			return true
		}
	}
	return false
}

// isValidSessionID validates session ID format (should be UUID-like)
func isValidSessionID(sessionID string) bool {
	// Be more lenient with session ID validation to avoid breaking existing usage
	// Just check for basic non-empty string for now
	// TODO: Implement stricter UUID validation when backward compatibility isn't a concern
	return strings.TrimSpace(sessionID) != ""
}

// NewClient creates a new Claude client with the specified binary path
func NewClient(binPath string) *ClaudeClient {
	return &ClaudeClient{
		BinPath: binPath,
		DefaultOptions: &RunOptions{
			Format: TextOutput,
		},
	}
}

// RunPrompt executes a prompt with Claude Code and returns the result
func (c *ClaudeClient) RunPrompt(prompt string, opts *RunOptions) (*ClaudeResult, error) {
	return c.RunPromptCtx(context.Background(), prompt, opts)
}

// RunPromptCtx executes a prompt with Claude Code and returns the result with context support
func (c *ClaudeClient) RunPromptCtx(ctx context.Context, prompt string, opts *RunOptions) (*ClaudeResult, error) {
	if opts == nil {
		opts = c.DefaultOptions
	}

	// Preprocess and validate options
	if err := PreprocessOptions(opts); err != nil {
		return nil, err
	}

	// Add timeout support if specified
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	args := BuildArgs(prompt, opts)

	cmd := execCommand(ctx, c.BinPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Enhanced error parsing
		var exitCode int
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
		
		claudeErr := ParseError(stderr.String(), exitCode)
		claudeErr.Original = err
		return nil, claudeErr
	}

	if opts.Format == JSONOutput {
		var res ClaudeResult
		if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
			return nil, NewClaudeError(ErrorValidation, fmt.Sprintf("failed to parse JSON response: %v", err))
		}
		return &res, nil
	}

	// For text output, just return the raw text
	return &ClaudeResult{
		Result:  stdout.String(),
		IsError: false,
	}, nil
}

// StreamPrompt executes a prompt with Claude Code and streams the results through a channel
func (c *ClaudeClient) StreamPrompt(ctx context.Context, prompt string, opts *RunOptions) (<-chan Message, <-chan error) {
	messageCh := make(chan Message)
	errCh := make(chan error, 1)

	if opts == nil {
		opts = c.DefaultOptions
	}

	// Force stream-json format for streaming
	streamOpts := *opts
	streamOpts.Format = StreamJSONOutput

	// Claude CLI requires --verbose when using --output-format=stream-json with --print
	streamOpts.Verbose = true

	args := BuildArgs(prompt, &streamOpts)

	go func() {
		defer close(messageCh)
		defer close(errCh)

		// Create a custom command that supports context
		cmd := execCommand(ctx, c.BinPath, args...)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errCh <- fmt.Errorf("failed to get stdout pipe: %w", err)
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			errCh <- fmt.Errorf("failed to get stderr pipe: %w", err)
			return
		}

		// Start capturing stderr in a goroutine
		stderrBuf := new(bytes.Buffer)
		go func() {
			_, _ = io.Copy(stderrBuf, stderr)
		}()

		if err := cmd.Start(); err != nil {
			errCh <- fmt.Errorf("failed to start command: %w", err)
			return
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			// Skip empty lines
			if strings.TrimSpace(line) == "" {
				continue
			}

			var msg Message
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				errCh <- fmt.Errorf("failed to parse JSON message: %w", err)
				return
			}

			select {
			case messageCh <- msg:
				// Message sent successfully
			case <-ctx.Done():
				// Context was canceled
				errCh <- ctx.Err()
				return
			}
		}

		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("scanner error: %w", err)
			return
		}

		if err := cmd.Wait(); err != nil {
			// Enhanced error parsing for streaming
			var exitCode int
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			} else {
				exitCode = 1
			}
			
			claudeErr := ParseError(stderrBuf.String(), exitCode)
			claudeErr.Original = err
			errCh <- claudeErr
			return
		}
	}()

	return messageCh, errCh
}

// RunFromStdin runs Claude Code with input from stdin
func (c *ClaudeClient) RunFromStdin(stdin io.Reader, prompt string, opts *RunOptions) (*ClaudeResult, error) {
	return c.RunFromStdinCtx(context.Background(), stdin, prompt, opts)
}

// RunFromStdinCtx runs Claude Code with input from stdin with context support
func (c *ClaudeClient) RunFromStdinCtx(ctx context.Context, stdin io.Reader, prompt string, opts *RunOptions) (*ClaudeResult, error) {
	if opts == nil {
		opts = c.DefaultOptions
	}

	// Preprocess and validate options
	if err := PreprocessOptions(opts); err != nil {
		return nil, err
	}

	// Add timeout support if specified
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	args := BuildArgs(prompt, opts)

	cmd := execCommand(ctx, c.BinPath, args...)
	cmd.Stdin = stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Enhanced error parsing
		var exitCode int
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
		
		claudeErr := ParseError(stderr.String(), exitCode)
		claudeErr.Original = err
		return nil, claudeErr
	}

	if opts.Format == JSONOutput {
		var res ClaudeResult
		if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
			return nil, NewClaudeError(ErrorValidation, fmt.Sprintf("failed to parse JSON response: %v", err))
		}
		return &res, nil
	}

	// For text output, just return the raw text
	return &ClaudeResult{
		Result:  stdout.String(),
		IsError: false,
	}, nil
}

// BuildArgs constructs the command-line arguments for Claude Code
// This is exported for use by the dangerous package
func BuildArgs(prompt string, opts *RunOptions) []string {
	args := []string{"-p"}

	// If prompt is empty, don't add it to args (useful when reading from stdin)
	if prompt != "" {
		args = append(args, prompt)
	}

	if opts.Format != "" {
		args = append(args, "--output-format", string(opts.Format))
	}

	if opts.SystemPrompt != "" {
		args = append(args, "--system-prompt", opts.SystemPrompt)
	}

	if opts.AppendPrompt != "" {
		args = append(args, "--append-system-prompt", opts.AppendPrompt)
	}

	if opts.MCPConfigPath != "" {
		args = append(args, "--mcp-config", opts.MCPConfigPath)
	}

	if len(opts.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(opts.AllowedTools, ","))
	}

	if len(opts.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools", strings.Join(opts.DisallowedTools, ","))
	}

	if opts.PermissionTool != "" {
		args = append(args, "--permission-prompt-tool", opts.PermissionTool)
	}

	if opts.ResumeID != "" {
		args = append(args, "--resume", opts.ResumeID)
	} else if opts.Continue {
		args = append(args, "--continue")
	}

	if opts.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", opts.MaxTurns))
	}

	if opts.Verbose {
		args = append(args, "--verbose")
	}

	// Model selection - prefer ModelAlias over Model for better UX
	if opts.ModelAlias != "" {
		args = append(args, "--model", opts.ModelAlias)
	} else if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}

	// Configuration file
	if opts.ConfigFile != "" {
		args = append(args, "--config", opts.ConfigFile)
	}

	// Help flag
	if opts.Help {
		args = append(args, "--help")
	}

	// Version flag
	if opts.Version {
		args = append(args, "--version")
	}

	// Disable autoupdate
	if opts.DisableAutoUpdate {
		args = append(args, "--disable-autoupdate")
	}

	// Theme
	if opts.Theme != "" {
		args = append(args, "--theme", opts.Theme)
	}

	return args
}

// RunWithMCP is a convenience method for running Claude with MCP configuration
func (c *ClaudeClient) RunWithMCP(prompt string, mcpConfigPath string, allowedTools []string) (*ClaudeResult, error) {
	return c.RunWithMCPCtx(context.Background(), prompt, mcpConfigPath, allowedTools)
}

// RunWithMCPCtx is a convenience method for running Claude with MCP configuration with context support
func (c *ClaudeClient) RunWithMCPCtx(ctx context.Context, prompt string, mcpConfigPath string, allowedTools []string) (*ClaudeResult, error) {
	return c.RunPromptCtx(ctx, prompt, &RunOptions{
		Format:        JSONOutput,
		MCPConfigPath: mcpConfigPath,
		AllowedTools:  allowedTools,
	})
}

// RunWithSystemPrompt is a convenience method for running Claude with a custom system prompt
func (c *ClaudeClient) RunWithSystemPrompt(prompt string, systemPrompt string, opts *RunOptions) (*ClaudeResult, error) {
	return c.RunWithSystemPromptCtx(context.Background(), prompt, systemPrompt, opts)
}

// RunWithSystemPromptCtx is a convenience method for running Claude with a custom system prompt with context support
func (c *ClaudeClient) RunWithSystemPromptCtx(ctx context.Context, prompt string, systemPrompt string, opts *RunOptions) (*ClaudeResult, error) {
	if opts == nil {
		opts = &RunOptions{}
	}

	// Create a copy to avoid modifying the original
	runOpts := *opts
	runOpts.SystemPrompt = systemPrompt

	return c.RunPromptCtx(ctx, prompt, &runOpts)
}

// ContinueConversation is a convenience method for continuing the most recent conversation
func (c *ClaudeClient) ContinueConversation(prompt string) (*ClaudeResult, error) {
	return c.ContinueConversationCtx(context.Background(), prompt)
}

// ContinueConversationCtx is a convenience method for continuing the most recent conversation with context support
func (c *ClaudeClient) ContinueConversationCtx(ctx context.Context, prompt string) (*ClaudeResult, error) {
	return c.RunPromptCtx(ctx, prompt, &RunOptions{
		Format:   JSONOutput,
		Continue: true,
	})
}

// ResumeConversation is a convenience method for resuming a specific conversation
func (c *ClaudeClient) ResumeConversation(prompt string, sessionID string) (*ClaudeResult, error) {
	return c.ResumeConversationCtx(context.Background(), prompt, sessionID)
}

// ResumeConversationCtx is a convenience method for resuming a specific conversation with context support
func (c *ClaudeClient) ResumeConversationCtx(ctx context.Context, prompt string, sessionID string) (*ClaudeResult, error) {
	return c.RunPromptCtx(ctx, prompt, &RunOptions{
		Format:   JSONOutput,
		ResumeID: sessionID,
	})
}

// RetryPolicy defines the retry behavior for failed requests
type RetryPolicy struct {
	MaxRetries    int           // Maximum number of retry attempts
	BaseDelay     time.Duration // Base delay between retries
	MaxDelay      time.Duration // Maximum delay between retries  
	BackoffFactor float64       // Exponential backoff factor
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
	}
}

// calculateBackoff calculates the delay for a given retry attempt
func (rp *RetryPolicy) calculateBackoff(attempt int) time.Duration {
	if attempt == 0 {
		return 0
	}
	
	delay := float64(rp.BaseDelay) * math.Pow(rp.BackoffFactor, float64(attempt-1))
	
	result := time.Duration(delay)
	if result > rp.MaxDelay {
		result = rp.MaxDelay
	}
	
	return result
}

// RunPromptWithRetry executes a prompt with intelligent retry logic for recoverable errors
func (c *ClaudeClient) RunPromptWithRetry(prompt string, opts *RunOptions, retryPolicy *RetryPolicy) (*ClaudeResult, error) {
	return c.RunPromptWithRetryCtx(context.Background(), prompt, opts, retryPolicy)
}

// RunPromptWithRetryCtx executes a prompt with context support and intelligent retry logic
func (c *ClaudeClient) RunPromptWithRetryCtx(ctx context.Context, prompt string, opts *RunOptions, retryPolicy *RetryPolicy) (*ClaudeResult, error) {
	if retryPolicy == nil {
		retryPolicy = DefaultRetryPolicy()
	}
	
	var lastErr error
	
	for attempt := 0; attempt <= retryPolicy.MaxRetries; attempt++ {
		// Calculate delay for this attempt (0 for first attempt)
		if attempt > 0 {
			delay := retryPolicy.calculateBackoff(attempt)
			
			select {
			case <-time.After(delay):
				// Continue with retry
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		
		result, err := c.RunPromptCtx(ctx, prompt, opts)
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if claudeErr, ok := err.(*ClaudeError); ok {
			if !claudeErr.IsRetryable() {
				return nil, err // Don't retry non-retryable errors
			}
			
			// For rate limit errors, respect the retry-after delay
			if claudeErr.Type == ErrorRateLimit {
				if retryAfter := claudeErr.RetryDelay(); retryAfter > 0 {
					select {
					case <-time.After(time.Duration(retryAfter) * time.Second):
						continue
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}
			}
		} else {
			// Non-ClaudeError types are not retryable
			return nil, err
		}
	}
	
	// All retries exhausted
	return nil, fmt.Errorf("max retries (%d) exceeded, last error: %w", retryPolicy.MaxRetries, lastErr)
}

// RunPromptEnhanced executes a prompt with all enhanced features: validation, timeout, and retry logic
func (c *ClaudeClient) RunPromptEnhanced(prompt string, opts *RunOptions) (*ClaudeResult, error) {
	return c.RunPromptEnhancedCtx(context.Background(), prompt, opts)
}

// RunPromptEnhancedCtx executes a prompt with context support and all enhanced features
func (c *ClaudeClient) RunPromptEnhancedCtx(ctx context.Context, prompt string, opts *RunOptions) (*ClaudeResult, error) {
	// Use default retry policy for enhanced mode
	return c.RunPromptWithRetryCtx(ctx, prompt, opts, DefaultRetryPolicy())
}

// ValidateOptions validates RunOptions without executing a command
func ValidateOptions(opts *RunOptions) error {
	return PreprocessOptions(opts)
}

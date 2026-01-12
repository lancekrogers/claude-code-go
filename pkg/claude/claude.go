package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// execCommand is a variable to allow mocking of exec.CommandContext for testing
var execCommand = exec.CommandContext

// ClaudeClient is the main client for interacting with Claude Code
type ClaudeClient struct {
	// BinPath is the path to the Claude Code binary
	BinPath string
	// DefaultOptions are the default options to use for all requests
	DefaultOptions *RunOptions
}

// ClaudeResult represents the structured result from Claude Code
type ClaudeResult struct {
	Type          string  `json:"type"`
	Subtype       string  `json:"subtype,omitempty"`
	Result        string  `json:"result,omitempty"`
	CostUSD       float64 `json:"total_cost_usd"`
	DurationMS    int64   `json:"duration_ms"`
	DurationAPIMS int64   `json:"duration_api_ms"`
	IsError       bool    `json:"is_error"`
	NumTurns      int     `json:"num_turns"`
	SessionID     string  `json:"session_id"`
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
		result, err := parseJSONResponse(stdout.Bytes())
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	// For text output, just return the raw text
	return &ClaudeResult{
		Result:  stdout.String(),
		IsError: false,
	}, nil
}

// parseJSONResponse handles both array and single-object JSON formats from Claude CLI
func parseJSONResponse(data []byte) (*ClaudeResult, error) {
	// Claude CLI now returns a JSON array of messages
	// We need to find the "result" type message
	var messages []Message
	if err := json.Unmarshal(data, &messages); err != nil {
		// Try single object for backwards compatibility
		var res ClaudeResult
		if err2 := json.Unmarshal(data, &res); err2 != nil {
			return nil, NewClaudeError(ErrorValidation, fmt.Sprintf("failed to parse JSON response: %v", err))
		}
		return &res, nil
	}

	// Find the result message in the array
	for _, msg := range messages {
		if msg.Type == "result" {
			return &ClaudeResult{
				Type:          msg.Type,
				Subtype:       msg.Subtype,
				Result:        msg.Result,
				CostUSD:       msg.CostUSD,
				DurationMS:    msg.DurationMS,
				DurationAPIMS: msg.DurationAPIMS,
				IsError:       msg.IsError,
				NumTurns:      msg.NumTurns,
				SessionID:     msg.SessionID,
			}, nil
		}
	}

	return nil, NewClaudeError(ErrorValidation, "no result message found in JSON response")
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
		result, err := parseJSONResponse(stdout.Bytes())
		if err != nil {
			return nil, err
		}
		return result, nil
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

	// Permission mode
	if opts.PermissionMode != "" {
		args = append(args, "--permission-mode", string(opts.PermissionMode))
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

	// Session control flags
	if opts.SessionID != "" {
		args = append(args, "--session-id", opts.SessionID)
	}

	if opts.ForkSession {
		args = append(args, "--fork-session")
	}

	if opts.NoSessionPersistence {
		args = append(args, "--no-session-persistence")
	}

	// MCP configuration (enhanced - multiple configs)
	if len(opts.MCPConfigs) > 0 {
		args = append(args, "--mcp-config")
		args = append(args, opts.MCPConfigs...)
	}

	if opts.StrictMCPConfig {
		args = append(args, "--strict-mcp-config")
	}

	// Additional directories
	for _, dir := range opts.AddDirectories {
		args = append(args, "--add-dir", dir)
	}

	// Print mode
	if opts.PrintMode {
		args = append(args, "--print")
	}

	// JSON Schema for structured output
	if opts.JSONSchema != "" {
		args = append(args, "--json-schema", opts.JSONSchema)
	}

	// Fallback model for overloaded primary
	if opts.FallbackModel != "" {
		args = append(args, "--fallback-model", opts.FallbackModel)
	}

	// Debug mode with filter
	if opts.Debug != "" {
		args = append(args, "--debug", opts.Debug)
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

// RunWithMCPConfigs executes with multiple MCP server configurations
func (c *ClaudeClient) RunWithMCPConfigs(prompt string, configs []string, opts *RunOptions) (*ClaudeResult, error) {
	return c.RunWithMCPConfigsCtx(context.Background(), prompt, configs, opts)
}

// RunWithMCPConfigsCtx executes with multiple MCP server configurations with context support
func (c *ClaudeClient) RunWithMCPConfigsCtx(ctx context.Context, prompt string, configs []string, opts *RunOptions) (*ClaudeResult, error) {
	if opts == nil {
		opts = &RunOptions{}
	}
	runOpts := *opts
	runOpts.MCPConfigs = configs
	if runOpts.Format == "" {
		runOpts.Format = JSONOutput
	}
	return c.RunPromptCtx(ctx, prompt, &runOpts)
}

// RunWithStrictMCP uses only specified MCP servers, ignoring all other configurations
func (c *ClaudeClient) RunWithStrictMCP(prompt string, configs []string, opts *RunOptions) (*ClaudeResult, error) {
	return c.RunWithStrictMCPCtx(context.Background(), prompt, configs, opts)
}

// RunWithStrictMCPCtx uses only specified MCP servers with context support
func (c *ClaudeClient) RunWithStrictMCPCtx(ctx context.Context, prompt string, configs []string, opts *RunOptions) (*ClaudeResult, error) {
	if opts == nil {
		opts = &RunOptions{}
	}
	runOpts := *opts
	runOpts.MCPConfigs = configs
	runOpts.StrictMCPConfig = true
	if runOpts.Format == "" {
		runOpts.Format = JSONOutput
	}
	return c.RunPromptCtx(ctx, prompt, &runOpts)
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
			delay := retryPolicy.calculateBackoffWithJitter(attempt)

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

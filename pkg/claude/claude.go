package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
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
	AllowedTools []string
	// DisallowedTools is a list of tools that Claude is not allowed to use
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
	// Model specifies the model to use
	Model string
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

	// Validate MCP tools
	if err := validateMCPTools(opts.AllowedTools); err != nil {
		return nil, err
	}
	if err := validateMCPTools(opts.DisallowedTools); err != nil {
		return nil, err
	}

	args := buildArgs(prompt, opts)

	cmd := execCommand(ctx, c.BinPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("claude command failed: %w: %s", err, stderr.String())
	}

	if opts.Format == JSONOutput {
		var res ClaudeResult
		if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
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

	args := buildArgs(prompt, &streamOpts)

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
			if stderrBuf.Len() > 0 {
				errCh <- fmt.Errorf("command failed: %w: %s", err, stderrBuf.String())
			} else {
				errCh <- fmt.Errorf("command failed: %w", err)
			}
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

	args := buildArgs(prompt, opts)

	cmd := execCommand(ctx, c.BinPath, args...)
	cmd.Stdin = stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("claude command failed: %w: %s", err, stderr.String())
	}

	if opts.Format == JSONOutput {
		var res ClaudeResult
		if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
		}
		return &res, nil
	}

	// For text output, just return the raw text
	return &ClaudeResult{
		Result:  stdout.String(),
		IsError: false,
	}, nil
}

// buildArgs constructs the command-line arguments for Claude Code
func buildArgs(prompt string, opts *RunOptions) []string {
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

	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
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

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

// Message represents a message from Claude Code in streaming mode
type Message struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype,omitempty"`
	Message   json.RawMessage `json:"message,omitempty"`
	SessionID string          `json:"session_id"`
	// Additional fields for system/result messages
	CostUSD       float64  `json:"total_cost_usd,omitempty"`
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

	if err := PreprocessOptions(&streamOpts); err != nil {
		errCh <- err
		close(messageCh)
		close(errCh)
		return messageCh, errCh
	}

	args := BuildArgs(prompt, &streamOpts)

	go func() {
		defer close(messageCh)
		defer close(errCh)

		runCtx := ctx
		var cancel context.CancelFunc
		if streamOpts.Timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, streamOpts.Timeout)
		} else {
			runCtx, cancel = context.WithCancel(ctx)
		}
		defer cancel()

		cleanupPlugins, err := preparePluginManager(runCtx, streamOpts.PluginManager)
		if err != nil {
			errCh <- err
			return
		}
		defer cleanupPlugins()

		// Create a custom command that supports context
		cmd := execCommand(runCtx, c.BinPath, args...)

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

			if err := applyMessageHooks(runCtx, &streamOpts, msg); err != nil {
				cancel()
				errCh <- err
				return
			}

			if msg.Type == "result" {
				if err := applyCompletionHooks(runCtx, &streamOpts, messageToResult(msg)); err != nil {
					cancel()
					errCh <- err
					return
				}
			}

			select {
			case messageCh <- msg:
				// Message sent successfully
			case <-runCtx.Done():
				// Context was canceled
				errCh <- runCtx.Err()
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

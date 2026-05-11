package claude

import (
	"context"
	"encoding/json"
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

	streamOpts, err := buildStreamJSONRunOptions(opts)
	if err != nil {
		errCh <- err
		close(messageCh)
		close(errCh)
		return messageCh, errCh
	}

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

		var assistantText strings.Builder
		seenResult := false
		if err := c.executeStreamJSON(runCtx, prompt, nil, streamOpts, func(msg Message) error {
			if text, ok := msg.AssistantText(); ok {
				assistantText.WriteString(text)
			}
			if msg.Type == "result" {
				seenResult = true
				result, err := normalizeResult(messageToResult(msg), assistantText.String())
				if err != nil {
					return err
				}
				msg.Result = result.Result
			}

			if err := applyMessageHooks(runCtx, streamOpts, msg); err != nil {
				return err
			}

			select {
			case messageCh <- msg:
			case <-runCtx.Done():
				return runCtx.Err()
			}

			if msg.Type == "result" {
				return applyCompletionHooks(runCtx, streamOpts, messageToResult(msg))
			}

			return nil
		}); err != nil {
			errCh <- err
			return
		}

		if !seenResult {
			errCh <- NewClaudeError(ErrorValidation, "no result message found in JSON response")
		}
	}()

	return messageCh, errCh
}

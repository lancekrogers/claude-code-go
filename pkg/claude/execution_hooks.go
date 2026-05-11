package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func parseJSONTranscript(data []byte) ([]Message, *ClaudeResult, error) {
	var messages []Message
	if err := json.Unmarshal(data, &messages); err == nil {
		result, resultErr := extractResultFromMessages(messages)
		if resultErr != nil {
			return nil, nil, resultErr
		}
		return messages, result, nil
	}

	var result ClaudeResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, nil, NewClaudeError(ErrorValidation, fmt.Sprintf("failed to parse JSON response: %v", err))
	}

	normalized, err := normalizeResult(&result, "")
	if err != nil {
		return nil, nil, err
	}
	return nil, normalized, nil
}

func extractResultFromMessages(messages []Message) (*ClaudeResult, error) {
	var assistantText strings.Builder
	for _, msg := range messages {
		if text, ok := msg.AssistantText(); ok {
			assistantText.WriteString(text)
		}
		if msg.Type != "result" {
			continue
		}

		return normalizeResult(messageToResult(msg), assistantText.String())
	}

	return nil, NewClaudeError(ErrorValidation, "no result message found in JSON response")
}

func normalizeResult(result *ClaudeResult, fallback string) (*ClaudeResult, error) {
	if result == nil {
		return nil, NewClaudeError(ErrorValidation, "nil result message")
	}
	if result.Result != "" || result.IsError {
		return result, nil
	}
	if fallback != "" {
		result.Result = fallback
		return result, nil
	}
	return nil, NewClaudeError(ErrorValidation, "empty result message in JSON response")
}

func messageToResult(msg Message) *ClaudeResult {
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
	}
}

func applyExecutionHooks(ctx context.Context, opts *RunOptions, messages []Message, result *ClaudeResult) error {
	if opts == nil {
		return nil
	}

	for _, msg := range messages {
		if err := applyMessageHooks(ctx, opts, msg); err != nil {
			return err
		}
	}

	return applyCompletionHooks(ctx, opts, result)
}

func applyMessageHooks(ctx context.Context, opts *RunOptions, msg Message) error {
	if opts == nil || opts.PluginManager == nil {
		return nil
	}

	if err := opts.PluginManager.OnMessage(ctx, msg); err != nil {
		return err
	}

	return emitToolUseHooks(ctx, opts.PluginManager, msg)
}

func applyCompletionHooks(ctx context.Context, opts *RunOptions, result *ClaudeResult) error {
	if opts == nil || result == nil {
		return nil
	}

	if tracker := opts.BudgetTracker; tracker != nil && result.CostUSD > 0 {
		if err := tracker.AddSpend(result.SessionID, result.CostUSD); err != nil {
			return err
		}
	}

	if opts.PluginManager != nil {
		if err := opts.PluginManager.OnComplete(ctx, result); err != nil {
			return err
		}
	}

	return nil
}

func ensurePluginManagerInitialized(ctx context.Context, pm *PluginManager) error {
	if pm == nil {
		return nil
	}

	return pm.Initialize(ctx)
}

func emitToolUseHooks(ctx context.Context, pm *PluginManager, msg Message) error {
	if pm == nil {
		return nil
	}

	toolCalls, ok := msg.ToolUses()
	if !ok {
		return nil
	}

	for _, call := range toolCalls {
		if err := pm.OnToolCall(ctx, call.Name, call.Input); err != nil {
			return err
		}
	}

	return nil
}

func decodeToolInput(raw map[string]interface{}) ToolInput {
	if raw == nil {
		return ToolInput{}
	}

	input := ToolInput{Raw: raw}
	input.Command = firstString(raw, "command")
	input.FilePath = firstString(raw, "file_path", "filePath", "path")
	input.Pattern = firstString(raw, "pattern")
	input.Content = firstString(raw, "content")
	input.OldString = firstString(raw, "old_string", "oldString")
	input.NewString = firstString(raw, "new_string", "newString")

	return input
}

func firstString(raw map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		if s, ok := value.(string); ok {
			return s
		}
	}

	return ""
}

package claude

import (
	"encoding/json"
	"strings"
)

// ToolUse represents a tool_use content block from a Claude assistant message.
type ToolUse struct {
	ID    string
	Name  string
	Input ToolInput
}

type assistantMessageEnvelope struct {
	Content []assistantContentBlock `json:"content"`
}

type assistantContentBlock struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text,omitempty"`
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
}

// AssistantText extracts assistant text from Claude Code stream messages.
//
// Current Claude Code stream-json emits assistant text as:
// {"type":"assistant","message":{"content":[{"type":"text","text":"..."}]}}
// Older/simple SDK consumers may also see text-like messages whose Message
// field is a JSON string; those are handled for compatibility.
func (m Message) AssistantText() (string, bool) {
	switch m.Type {
	case "assistant":
		var envelope assistantMessageEnvelope
		if err := json.Unmarshal(m.Message, &envelope); err != nil {
			return "", false
		}

		var b strings.Builder
		for _, item := range envelope.Content {
			if item.Type != "text" || item.Text == "" {
				continue
			}
			b.WriteString(item.Text)
		}
		if b.Len() == 0 {
			return "", false
		}
		return b.String(), true
	case "text", "thinking":
		var content string
		if err := json.Unmarshal(m.Message, &content); err != nil || content == "" {
			return "", false
		}
		return content, true
	default:
		return "", false
	}
}

// ToolUses extracts tool_use content blocks from Claude Code assistant messages.
func (m Message) ToolUses() ([]ToolUse, bool) {
	if m.Type != "assistant" || len(m.Message) == 0 {
		return nil, false
	}

	var envelope assistantMessageEnvelope
	if err := json.Unmarshal(m.Message, &envelope); err != nil {
		return nil, false
	}

	toolUses := make([]ToolUse, 0)
	for _, item := range envelope.Content {
		if item.Type != "tool_use" {
			continue
		}
		toolUses = append(toolUses, ToolUse{
			ID:    item.ID,
			Name:  item.Name,
			Input: decodeToolInput(item.Input),
		})
	}

	return toolUses, len(toolUses) > 0
}

package claude

import (
	"encoding/json"
	"testing"
)

func TestMessageAssistantTextExtractsCurrentStreamContent(t *testing.T) {
	raw := json.RawMessage(`{"role":"assistant","content":[{"type":"text","text":"hello"},{"type":"text","text":" world"}]}`)
	msg := Message{Type: "assistant", Message: raw}

	got, ok := msg.AssistantText()
	if !ok {
		t.Fatal("AssistantText() ok = false, want true")
	}
	if got != "hello world" {
		t.Fatalf("AssistantText() = %q, want %q", got, "hello world")
	}
}

func TestMessagePartialTextExtractsTextDelta(t *testing.T) {
	raw := json.RawMessage(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hel"}}`)
	msg := Message{Type: "stream_event", Event: raw}

	got, ok := msg.PartialText()
	if !ok {
		t.Fatal("PartialText() ok = false, want true")
	}
	if got != "Hel" {
		t.Fatalf("PartialText() = %q, want %q", got, "Hel")
	}
}

func TestMessagePartialTextIgnoresNonTextEvents(t *testing.T) {
	cases := map[string]Message{
		"non stream_event type": {Type: "assistant", Event: json.RawMessage(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"x"}}`)},
		"empty event":           {Type: "stream_event"},
		"message_start":         {Type: "stream_event", Event: json.RawMessage(`{"type":"message_start","message":{"id":"m"}}`)},
		"input_json_delta":      {Type: "stream_event", Event: json.RawMessage(`{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"{"}}`)},
		"content_block_stop":    {Type: "stream_event", Event: json.RawMessage(`{"type":"content_block_stop","index":0}`)},
		"empty text_delta":      {Type: "stream_event", Event: json.RawMessage(`{"type":"content_block_delta","delta":{"type":"text_delta","text":""}}`)},
	}
	for name, msg := range cases {
		if got, ok := msg.PartialText(); ok || got != "" {
			t.Errorf("%s: PartialText() = (%q, %v), want (\"\", false)", name, got, ok)
		}
	}
}

func TestMessageToolUsesExtractsCurrentStreamContent(t *testing.T) {
	raw := json.RawMessage(`{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Read","input":{"file_path":"README.md"}}]}`)
	msg := Message{Type: "assistant", Message: raw}

	got, ok := msg.ToolUses()
	if !ok {
		t.Fatal("ToolUses() ok = false, want true")
	}
	if len(got) != 1 {
		t.Fatalf("ToolUses() length = %d, want 1", len(got))
	}
	if got[0].ID != "toolu_1" {
		t.Fatalf("ToolUses()[0].ID = %q, want %q", got[0].ID, "toolu_1")
	}
	if got[0].Name != "Read" {
		t.Fatalf("ToolUses()[0].Name = %q, want %q", got[0].Name, "Read")
	}
	if got[0].Input.FilePath != "README.md" {
		t.Fatalf("ToolUses()[0].Input.FilePath = %q, want %q", got[0].Input.FilePath, "README.md")
	}
}

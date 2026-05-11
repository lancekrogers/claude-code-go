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

package claude

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// pngHeader base64-encodes to "iVBORw==".
var pngHeader = []byte{0x89, 'P', 'N', 'G'}

const imageEventLine = `{"type":"user","message":{"role":"user","content":[` +
	`{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBORw=="}},` +
	`{"type":"text","text":"What color is \"this\"?"}` +
	`]},"parent_tool_use_id":null}` + "\n"

type capturedRun struct {
	args  []string
	calls int
	stdin string
}

func captureStdinCommand(t *testing.T) *capturedRun {
	t.Helper()
	originalExecCommand := execCommand
	t.Cleanup(func() { execCommand = originalExecCommand })

	run := &capturedRun{stdin: filepath.Join(t.TempDir(), "stdin.jsonl")}
	execCommand = func(_ context.Context, name string, arg ...string) *exec.Cmd {
		run.calls++
		run.args = append([]string(nil), arg...)
		cs := append([]string{"-test.run=TestHelperProcess", "--", name}, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{
			"GO_WANT_HELPER_PROCESS=1",
			"GO_HELPER_CAPTURE_STDIN=" + run.stdin,
		}
		return cmd
	}
	return run
}

func (r *capturedRun) stdinLine(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(r.stdin)
	if err != nil {
		t.Fatalf("read captured stdin: %v", err)
	}
	return string(data)
}

func assertArgPair(t *testing.T, args []string, flag, value string) {
	t.Helper()
	i := slices.Index(args, flag)
	if i < 0 || i+1 >= len(args) || args[i+1] != value {
		t.Errorf("args = %v, want %s %s", args, flag, value)
	}
}

func imageAndTextBlocks() []ContentBlock {
	return []ContentBlock{
		ImageBlock("image/png", pngHeader),
		TextBlock(`What color is "this"?`),
	}
}

func TestContentStdinEncodesUserEvent(t *testing.T) {
	stdin, err := contentStdin(imageAndTextBlocks())
	if err != nil {
		t.Fatalf("contentStdin() error = %v", err)
	}
	got := make([]byte, len(imageEventLine)+16)
	n, _ := stdin.Read(got)
	if string(got[:n]) != imageEventLine {
		t.Fatalf("stdin = %q, want %q", got[:n], imageEventLine)
	}
}

func TestStreamContentWritesBlocksToStdin(t *testing.T) {
	run := captureStdinCommand(t)
	client := NewClient("claude")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msgCh, errCh := client.StreamContent(ctx, imageAndTextBlocks(), &RunOptions{
		ResumeID:       "sess-1",
		PermissionMode: PermissionModeAcceptEdits,
		AppendPrompt:   "be brief",
		Effort:         EffortHigh,
	})

	var results []Message
	for msg := range msgCh {
		if msg.Type == "result" {
			results = append(results, msg)
		}
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("stream error: %v", err)
		}
	}

	if len(results) != 1 || results[0].Result != "captured" {
		t.Fatalf("results = %+v, want one captured result", results)
	}
	if got := run.stdinLine(t); got != imageEventLine {
		t.Errorf("stdin = %q, want %q", got, imageEventLine)
	}
	if run.args[0] != "-p" || run.args[1] != "--output-format" {
		t.Errorf("args = %v, want -p followed by flags and no inline prompt", run.args)
	}
	assertArgPair(t, run.args, "--output-format", "stream-json")
	assertArgPair(t, run.args, "--input-format", "stream-json")
	assertArgPair(t, run.args, "--resume", "sess-1")
	assertArgPair(t, run.args, "--permission-mode", "acceptEdits")
	assertArgPair(t, run.args, "--append-system-prompt", "be brief")
	assertArgPair(t, run.args, "--effort", "high")
	if !slices.Contains(run.args, "--verbose") {
		t.Errorf("args = %v, want --verbose", run.args)
	}
}

func TestStreamContentLeavesDefaultOptionsUnchanged(t *testing.T) {
	captureStdinCommand(t)
	client := NewClient("claude")

	msgCh, errCh := client.StreamContent(context.Background(), imageAndTextBlocks(), nil)
	for range msgCh {
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("stream error: %v", err)
		}
	}

	if client.DefaultOptions.InputFormat != "" || client.DefaultOptions.Format != TextOutput {
		t.Fatalf("DefaultOptions = %+v, want the text defaults untouched", client.DefaultOptions)
	}
}

func TestRunContentCtxWritesBlocksToStdin(t *testing.T) {
	run := captureStdinCommand(t)
	client := NewClient("claude")

	result, err := client.RunContentCtx(context.Background(), imageAndTextBlocks(), &RunOptions{
		Format:   JSONOutput,
		ResumeID: "sess-2",
	})
	if err != nil {
		t.Fatalf("RunContentCtx() error = %v", err)
	}
	if result.Result != "captured" || result.SessionID != "test-session" {
		t.Fatalf("result = %+v, want captured result for test-session", result)
	}
	if got := run.stdinLine(t); got != imageEventLine {
		t.Errorf("stdin = %q, want %q", got, imageEventLine)
	}
	assertArgPair(t, run.args, "--input-format", "stream-json")
	assertArgPair(t, run.args, "--output-format", "stream-json")
	assertArgPair(t, run.args, "--resume", "sess-2")
}

func TestContentBlocksValidation(t *testing.T) {
	tests := []struct {
		name   string
		blocks []ContentBlock
		field  string
	}{
		{name: "no blocks", blocks: nil, field: "Blocks"},
		{name: "empty text", blocks: []ContentBlock{TextBlock("  ")}, field: "Blocks[0].Text"},
		{name: "image without source", blocks: []ContentBlock{{Type: ContentBlockImage}}, field: "Blocks[0].Source"},
		{name: "image without media type", blocks: []ContentBlock{ImageBlock("", pngHeader)}, field: "Blocks[0].Source.MediaType"},
		{name: "image without data", blocks: []ContentBlock{TextBlock("hi"), ImageBlock("image/png", nil)}, field: "Blocks[1].Source.Data"},
		{
			name:   "non-base64 source",
			blocks: []ContentBlock{{Type: ContentBlockImage, Source: &ImageSource{Type: "url", MediaType: "image/png", Data: "x"}}},
			field:  "Blocks[0].Source.Type",
		},
		{name: "unknown block type", blocks: []ContentBlock{{Type: "document", Text: "x"}}, field: "Blocks[0].Type"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			run := captureStdinCommand(t)
			client := NewClient("claude")

			msgCh, errCh := client.StreamContent(context.Background(), tc.blocks, nil)
			for range msgCh {
			}
			assertValidationField(t, <-errCh, tc.field)

			_, err := client.RunContentCtx(context.Background(), tc.blocks, nil)
			assertValidationField(t, err, tc.field)

			if run.calls != 0 {
				t.Fatalf("CLI started %d times, want 0 for invalid blocks", run.calls)
			}
		})
	}
}

func assertValidationField(t *testing.T, err error, field string) {
	t.Helper()
	var claudeErr *ClaudeError
	if !errors.As(err, &claudeErr) || claudeErr.Type != ErrorValidation {
		t.Fatalf("error = %v, want validation ClaudeError", err)
	}
	if claudeErr.Details["field"] != field {
		t.Fatalf("field = %v, want %q", claudeErr.Details["field"], field)
	}
}

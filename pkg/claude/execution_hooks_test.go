package claude

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type lifecyclePlugin struct {
	mu             sync.Mutex
	initCount      int
	shutdownCount  int
	messageCount   int
	completeCount  int
	toolCalls      []string
	lastToolInput  ToolInput
	lastResultCost float64
}

func (p *lifecyclePlugin) Name() string    { return "lifecycle" }
func (p *lifecyclePlugin) Version() string { return "test" }

func (p *lifecyclePlugin) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.initCount++
	return nil
}

func (p *lifecyclePlugin) OnToolCall(ctx context.Context, toolName string, input ToolInput) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.toolCalls = append(p.toolCalls, toolName)
	p.lastToolInput = input
	return nil
}

func (p *lifecyclePlugin) OnMessage(ctx context.Context, msg Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messageCount++
	return nil
}

func (p *lifecyclePlugin) OnComplete(ctx context.Context, result *ClaudeResult) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completeCount++
	p.lastResultCost = result.CostUSD
	return nil
}

func (p *lifecyclePlugin) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.shutdownCount++
	return nil
}

func TestRunPromptCtx_AppliesLifecycleHooks(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"pwd"}}]},"session_id":"hook-session"}` + "\n" +
		`{"type":"result","subtype":"success","total_cost_usd":0.25,"duration_ms":12,"duration_api_ms":8,"is_error":false,"num_turns":1,"result":"done","session_id":"hook-session"}` + "\n"
	execCommand = mockExecCommandContext(t, []string{"-p", "Hook test", "--output-format", "stream-json", "--verbose"}, jsonOutput, 0)

	pm := NewPluginManager()
	plugin := &lifecyclePlugin{}
	if err := pm.Register(plugin, nil); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	tracker := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 1.0})
	client := NewClient("claude")

	result, err := client.RunPromptCtx(context.Background(), "Hook test", &RunOptions{
		Format:        JSONOutput,
		PluginManager: pm,
		BudgetTracker: tracker,
	})
	if err != nil {
		t.Fatalf("RunPromptCtx() error = %v", err)
	}

	if result.Result != "done" {
		t.Fatalf("Result = %q, want %q", result.Result, "done")
	}
	if tracker.TotalSpent() != 0.25 {
		t.Fatalf("TotalSpent = %f, want %f", tracker.TotalSpent(), 0.25)
	}
	if tracker.SessionSpent("hook-session") != 0.25 {
		t.Fatalf("SessionSpent = %f, want %f", tracker.SessionSpent("hook-session"), 0.25)
	}

	plugin.mu.Lock()
	defer plugin.mu.Unlock()

	if plugin.initCount != 1 {
		t.Fatalf("Initialize count = %d, want 1", plugin.initCount)
	}
	if plugin.shutdownCount != 0 {
		t.Fatalf("Shutdown count = %d, want 0", plugin.shutdownCount)
	}
	if plugin.messageCount != 2 {
		t.Fatalf("Message count = %d, want 2", plugin.messageCount)
	}
	if plugin.completeCount != 1 {
		t.Fatalf("Complete count = %d, want 1", plugin.completeCount)
	}
	if len(plugin.toolCalls) != 1 || plugin.toolCalls[0] != "Bash" {
		t.Fatalf("Tool calls = %v, want [Bash]", plugin.toolCalls)
	}
	if plugin.lastToolInput.Command != "pwd" {
		t.Fatalf("Tool command = %q, want %q", plugin.lastToolInput.Command, "pwd")
	}
	if plugin.lastResultCost != 0.25 {
		t.Fatalf("Result cost = %f, want %f", plugin.lastResultCost, 0.25)
	}
}

func TestRunPromptCtx_DoesNotShutdownPreinitializedPluginManager(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"result","subtype":"success","total_cost_usd":0.01,"duration_ms":5,"duration_api_ms":5,"is_error":false,"num_turns":1,"result":"ok","session_id":"sticky-session"}` + "\n"
	execCommand = mockExecCommandContext(t, []string{"-p", "Sticky hooks", "--output-format", "stream-json", "--verbose"}, jsonOutput, 0)

	pm := NewPluginManager()
	plugin := &lifecyclePlugin{}
	if err := pm.Register(plugin, nil); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := pm.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	defer func() {
		_ = pm.Shutdown(context.Background())
	}()

	client := NewClient("claude")
	if _, err := client.RunPromptCtx(context.Background(), "Sticky hooks", &RunOptions{
		Format:        JSONOutput,
		PluginManager: pm,
	}); err != nil {
		t.Fatalf("RunPromptCtx() error = %v", err)
	}

	plugin.mu.Lock()
	defer plugin.mu.Unlock()

	if plugin.initCount != 1 {
		t.Fatalf("Initialize count = %d, want 1", plugin.initCount)
	}
	if plugin.shutdownCount != 0 {
		t.Fatalf("Shutdown count = %d, want 0", plugin.shutdownCount)
	}
}

func TestRunPromptCtx_ReusesLazyInitializedPluginManager(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"result","subtype":"success","total_cost_usd":0.01,"duration_ms":5,"duration_api_ms":5,"is_error":false,"num_turns":1,"result":"ok","session_id":"sticky-session"}` + "\n"
	execCommand = mockExecCommandContext(t, []string{"-p", "Sticky hooks", "--output-format", "stream-json", "--verbose"}, jsonOutput, 0)

	pm := NewPluginManager()
	plugin := &lifecyclePlugin{}
	if err := pm.Register(plugin, nil); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	client := NewClient("claude")
	for i := 0; i < 2; i++ {
		if _, err := client.RunPromptCtx(context.Background(), "Sticky hooks", &RunOptions{
			Format:        JSONOutput,
			PluginManager: pm,
		}); err != nil {
			t.Fatalf("RunPromptCtx() error = %v", err)
		}
	}

	plugin.mu.Lock()
	defer plugin.mu.Unlock()

	if plugin.initCount != 1 {
		t.Fatalf("Initialize count = %d, want 1", plugin.initCount)
	}
	if plugin.shutdownCount != 0 {
		t.Fatalf("Shutdown count = %d, want 0", plugin.shutdownCount)
	}
}

func TestRunFromStdinCtx_AppliesLifecycleHooks(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"README.md"}}]},"session_id":"stdin-hook-session"}` + "\n" +
		`{"type":"result","subtype":"success","total_cost_usd":0.05,"duration_ms":7,"duration_api_ms":5,"is_error":false,"num_turns":1,"result":"stdin-done","session_id":"stdin-hook-session"}` + "\n"
	execCommand = mockExecCommandContext(t, []string{"-p", "Hook stdin", "--output-format", "stream-json", "--verbose"}, jsonOutput, 0)

	pm := NewPluginManager()
	plugin := &lifecyclePlugin{}
	if err := pm.Register(plugin, nil); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	client := NewClient("claude")
	result, err := client.RunFromStdinCtx(context.Background(), strings.NewReader("stdin body"), "Hook stdin", &RunOptions{
		Format:        JSONOutput,
		PluginManager: pm,
	})
	if err != nil {
		t.Fatalf("RunFromStdinCtx() error = %v", err)
	}
	if result.Result != "stdin-done" {
		t.Fatalf("Result = %q, want %q", result.Result, "stdin-done")
	}

	plugin.mu.Lock()
	defer plugin.mu.Unlock()

	if plugin.initCount != 1 {
		t.Fatalf("Initialize count = %d, want 1", plugin.initCount)
	}
	if plugin.shutdownCount != 0 {
		t.Fatalf("Shutdown count = %d, want 0", plugin.shutdownCount)
	}
	if plugin.messageCount != 2 {
		t.Fatalf("Message count = %d, want 2", plugin.messageCount)
	}
	if plugin.completeCount != 1 {
		t.Fatalf("Complete count = %d, want 1", plugin.completeCount)
	}
	if len(plugin.toolCalls) != 1 || plugin.toolCalls[0] != "Read" {
		t.Fatalf("Tool calls = %v, want [Read]", plugin.toolCalls)
	}
	if plugin.lastToolInput.FilePath != "README.md" {
		t.Fatalf("Tool file path = %q, want %q", plugin.lastToolInput.FilePath, "README.md")
	}
}

func TestRunPromptCtx_BudgetExceededReturnsResultAndError(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"result","subtype":"success","total_cost_usd":0.25,"duration_ms":12,"duration_api_ms":8,"is_error":false,"num_turns":1,"result":"done","session_id":"budget-session"}`
	execCommand = mockExecCommandContext(t, []string{"-p", "Budget test", "--output-format", "json"}, jsonOutput, 0)

	tracker := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 0.10})
	client := NewClient("claude")

	result, err := client.RunPromptCtx(context.Background(), "Budget test", &RunOptions{
		Format:        JSONOutput,
		BudgetTracker: tracker,
	})
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("RunPromptCtx() error = %v, want ErrBudgetExceeded", err)
	}
	if result == nil {
		t.Fatal("RunPromptCtx() result = nil, want result")
	}
	if result.Result != "done" {
		t.Fatalf("Result = %q, want %q", result.Result, "done")
	}
	if tracker.TotalSpent() != 0.25 {
		t.Fatalf("TotalSpent = %f, want %f", tracker.TotalSpent(), 0.25)
	}
}

func TestRunPromptCtx_WithPluginManagerBackfillsEmptyResultFromAssistantText(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hook answer"}]},"session_id":"hook-session"}` + "\n" +
		`{"type":"result","subtype":"success","total_cost_usd":0.01,"duration_ms":1,"duration_api_ms":1,"is_error":false,"num_turns":1,"result":"","session_id":"hook-session"}` + "\n"
	execCommand = mockExecCommandContext(t, []string{"-p", "Hook answer", "--output-format", "stream-json", "--verbose"}, jsonOutput, 0)

	pm := NewPluginManager()
	if err := pm.Register(&BasePlugin{PluginName: "noop", PluginVersion: "test"}, nil); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	client := NewClient("claude")
	result, err := client.RunPromptCtx(context.Background(), "Hook answer", &RunOptions{
		Format:        JSONOutput,
		PluginManager: pm,
	})
	if err != nil {
		t.Fatalf("RunPromptCtx() error = %v", err)
	}
	if result.Result != "hook answer" {
		t.Fatalf("Result = %q, want %q", result.Result, "hook answer")
	}
}

func TestStreamPrompt_AppliesLifecycleHooks(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	mockBinary := buildStreamingMockBinary(t)
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.Command(mockBinary)
	}

	pm := NewPluginManager()
	plugin := &lifecyclePlugin{}
	if err := pm.Register(plugin, nil); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	tracker := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 1.0})
	client := NewClient("claude")
	messageCh, errCh := client.StreamPrompt(context.Background(), "Stream hooks", &RunOptions{
		PluginManager: pm,
		BudgetTracker: tracker,
	})

	var gotMessages []Message
	for msg := range messageCh {
		gotMessages = append(gotMessages, msg)
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("StreamPrompt() error = %v", err)
		}
	}

	if len(gotMessages) != 2 {
		t.Fatalf("Expected 2 streamed messages, got %d", len(gotMessages))
	}
	if tracker.TotalSpent() != 0.4 {
		t.Fatalf("TotalSpent = %f, want %f", tracker.TotalSpent(), 0.4)
	}

	plugin.mu.Lock()
	defer plugin.mu.Unlock()

	if plugin.initCount != 1 {
		t.Fatalf("Initialize count = %d, want 1", plugin.initCount)
	}
	if plugin.shutdownCount != 0 {
		t.Fatalf("Shutdown count = %d, want 0", plugin.shutdownCount)
	}
	if plugin.messageCount != 2 {
		t.Fatalf("Message count = %d, want 2", plugin.messageCount)
	}
	if plugin.completeCount != 1 {
		t.Fatalf("Complete count = %d, want 1", plugin.completeCount)
	}
	if len(plugin.toolCalls) != 1 || plugin.toolCalls[0] != "Read" {
		t.Fatalf("Tool calls = %v, want [Read]", plugin.toolCalls)
	}
	if plugin.lastToolInput.FilePath != "README.md" {
		t.Fatalf("Tool file path = %q, want %q", plugin.lastToolInput.FilePath, "README.md")
	}
}

func TestStreamPrompt_BudgetExceededReturnsResultBeforeError(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	mockBinary := buildStreamingMockBinary(t)
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.Command(mockBinary)
	}

	tracker := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 0.10})
	client := NewClient("claude")
	messageCh, errCh := client.StreamPrompt(context.Background(), "Stream hooks", &RunOptions{
		BudgetTracker: tracker,
	})

	var gotMessages []Message
	for msg := range messageCh {
		gotMessages = append(gotMessages, msg)
	}

	var gotErr error
	for err := range errCh {
		if err != nil {
			gotErr = err
		}
	}

	if !errors.Is(gotErr, ErrBudgetExceeded) {
		t.Fatalf("StreamPrompt() error = %v, want ErrBudgetExceeded", gotErr)
	}
	if len(gotMessages) != 2 {
		t.Fatalf("Expected 2 streamed messages, got %d", len(gotMessages))
	}
	if gotMessages[len(gotMessages)-1].Type != "result" {
		t.Fatalf("Last streamed message type = %q, want %q", gotMessages[len(gotMessages)-1].Type, "result")
	}
	if tracker.TotalSpent() != 0.4 {
		t.Fatalf("TotalSpent = %f, want %f", tracker.TotalSpent(), 0.4)
	}
}

func TestRunPromptCtx_HandlesLargeStreamJSONMessages(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	mockBinary := buildLargeResultMockBinary(t, 128*1024)
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.Command(mockBinary)
	}

	pm := NewPluginManager()
	if err := pm.Register(&BasePlugin{PluginName: "noop", PluginVersion: "test"}, nil); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	client := NewClient("claude")
	result, err := client.RunPromptCtx(context.Background(), "Large message", &RunOptions{
		Format:        JSONOutput,
		PluginManager: pm,
	})
	if err != nil {
		t.Fatalf("RunPromptCtx() error = %v", err)
	}
	if len(result.Result) != 128*1024 {
		t.Fatalf("Result length = %d, want %d", len(result.Result), 128*1024)
	}
}

func TestParseJSONTranscript_BackfillsEmptyResultFromAssistantText(t *testing.T) {
	data := []byte(`[` +
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"final answer"}]},"session_id":"stream-session"},` +
		`{"type":"result","subtype":"success","is_error":false,"result":"","session_id":"stream-session"}` +
		`]`)

	_, result, err := parseJSONTranscript(data)
	if err != nil {
		t.Fatalf("parseJSONTranscript() error = %v", err)
	}
	if result.Result != "final answer" {
		t.Fatalf("Result = %q, want %q", result.Result, "final answer")
	}
}

func TestParseJSONTranscript_RejectsEmptySuccessfulResult(t *testing.T) {
	data := []byte(`{"type":"result","subtype":"success","is_error":false,"result":"","session_id":"stream-session"}`)

	_, _, err := parseJSONTranscript(data)
	if err == nil {
		t.Fatal("parseJSONTranscript() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "empty result message") {
		t.Fatalf("parseJSONTranscript() error = %v, want empty result message", err)
	}
}

func TestStreamPrompt_BackfillsEmptyResultFromAssistantMessage(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"stream answer"}]},"session_id":"stream-session"}` + "\n" +
		`{"type":"result","subtype":"success","total_cost_usd":0.01,"duration_ms":1,"duration_api_ms":1,"is_error":false,"num_turns":1,"result":"","session_id":"stream-session"}` + "\n"
	execCommand = mockExecCommandContext(t, []string{"-p", "Stream answer", "--output-format", "stream-json", "--verbose"}, jsonOutput, 0)

	client := NewClient("claude")
	messageCh, errCh := client.StreamPrompt(context.Background(), "Stream answer", &RunOptions{})

	var gotResult Message
	for msg := range messageCh {
		if msg.Type == "result" {
			gotResult = msg
		}
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("StreamPrompt() error = %v", err)
		}
	}
	if gotResult.Result != "stream answer" {
		t.Fatalf("result message Result = %q, want %q", gotResult.Result, "stream answer")
	}
}

func TestStreamPrompt_NoResultReturnsValidationError(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"orphan answer"}]},"session_id":"stream-session"}` + "\n"
	execCommand = mockExecCommandContext(t, []string{"-p", "Stream answer", "--output-format", "stream-json", "--verbose"}, jsonOutput, 0)

	client := NewClient("claude")
	messageCh, errCh := client.StreamPrompt(context.Background(), "Stream answer", &RunOptions{})
	for range messageCh {
	}

	var gotErr error
	for err := range errCh {
		if err != nil {
			gotErr = err
		}
	}
	if gotErr == nil {
		t.Fatal("StreamPrompt() error = nil, want error")
	}
	if !strings.Contains(gotErr.Error(), "no result message") {
		t.Fatalf("StreamPrompt() error = %v, want no result message", gotErr)
	}
}

func buildStreamingMockBinary(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	mockSource := filepath.Join(tempDir, "mock_stream.go")
	mockBinary := filepath.Join(tempDir, "mock_stream")

	source := `package main
import "fmt"
func main() {
	fmt.Println(` + "`" + `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"README.md"}}]},"session_id":"stream-session"}` + "`" + `)
	fmt.Println(` + "`" + `{"type":"result","subtype":"success","total_cost_usd":0.4,"duration_ms":20,"duration_api_ms":15,"is_error":false,"num_turns":1,"result":"stream-done","session_id":"stream-session"}` + "`" + `)
}
`

	if err := os.WriteFile(mockSource, []byte(source), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	build := exec.Command("go", "build", "-o", mockBinary, mockSource)
	if err := build.Run(); err != nil {
		t.Fatalf("go build error = %v", err)
	}

	return mockBinary
}

func buildLargeResultMockBinary(t *testing.T, size int) string {
	t.Helper()

	tempDir := t.TempDir()
	mockSource := filepath.Join(tempDir, "mock_large_stream.go")
	mockBinary := filepath.Join(tempDir, "mock_large_stream")

	source := `package main
import (
	"encoding/json"
	"os"
	"strings"
)
func main() {
	payload := strings.Repeat("a", ` + fmt.Sprintf("%d", size) + `)
	msg := map[string]any{
		"type": "result",
		"subtype": "success",
		"total_cost_usd": 0.01,
		"duration_ms": 1,
		"duration_api_ms": 1,
		"is_error": false,
		"num_turns": 1,
		"result": payload,
		"session_id": "large-session",
	}
	_ = json.NewEncoder(os.Stdout).Encode(msg)
}
`

	if err := os.WriteFile(mockSource, []byte(source), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	build := exec.Command("go", "build", "-o", mockBinary, mockSource)
	if err := build.Run(); err != nil {
		t.Fatalf("go build error = %v", err)
	}

	return mockBinary
}

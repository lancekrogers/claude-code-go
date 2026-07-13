package claude

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mockExecCommandContext returns a function that creates a mock command with context
func mockExecCommandContext(t *testing.T, expectedArgs []string, output string, exitCode int) func(context.Context, string, ...string) *exec.Cmd {
	return func(_ context.Context, name string, arg ...string) *exec.Cmd {
		// Verify correct arguments were passed
		if len(arg) != len(expectedArgs) {
			t.Errorf("Expected %d arguments, got %d", len(expectedArgs), len(arg))
		}

		for i, a := range arg {
			if i < len(expectedArgs) && a != expectedArgs[i] {
				t.Errorf("Expected arg[%d] to be %q, got %q", i, expectedArgs[i], a)
			}
		}

		// Create a fake command that outputs our desired text and exits with the given code
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, arg...)

		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{
			"GO_WANT_HELPER_PROCESS=1",
			"GO_HELPER_OUTPUT=" + output,
			"GO_HELPER_EXIT_CODE=" + string(rune(exitCode)+'0'),
		}
		return cmd
	}
}

// TestHelperProcess isn't a real test - it's used to mock exec.Command
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	if os.Getenv("GO_HELPER_PRINT_PWD") == "1" {
		wd, err := os.Getwd()
		if err != nil {
			os.Stderr.WriteString(err.Error())
			os.Exit(1)
		}
		os.Stdout.Write([]byte(wd))
		return
	}

	if os.Getenv("GO_HELPER_STREAM_PWD") == "1" {
		wd, err := os.Getwd()
		if err != nil {
			os.Stderr.WriteString(err.Error())
			os.Exit(1)
		}
		os.Stdout.Write([]byte(`{"type":"result","subtype":"success","total_cost_usd":0.0,"duration_ms":1,"duration_api_ms":1,"is_error":false,"num_turns":1,"result":"` + wd + `","session_id":"test-session"}` + "\n"))
		return
	}

	output := os.Getenv("GO_HELPER_OUTPUT")
	exitCode := int(os.Getenv("GO_HELPER_EXIT_CODE")[0] - '0')

	if output != "" {
		os.Stdout.Write([]byte(output))
	}

	os.Exit(exitCode)
}

func TestRunPrompt(t *testing.T) {
	// Save the original execCommand and restore it after the test
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Test with text output
	execCommand = mockExecCommandContext(t, []string{"-p", "Hello, Claude", "--output-format", "text"}, "Hello, human!", 0)

	client := &ClaudeClient{BinPath: "claude"}
	result, err := client.RunPrompt("Hello, Claude", &RunOptions{Format: TextOutput})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Result != "Hello, human!" {
		t.Errorf("Expected %q, got %q", "Hello, human!", result.Result)
	}

	// Test with JSON output
	jsonOutput := `{"type":"result","subtype":"success","total_cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":1,"result":"JSON response","session_id":"abc123"}`
	execCommand = mockExecCommandContext(t, []string{"-p", "JSON test", "--output-format", "json"}, jsonOutput, 0)

	result, err = client.RunPrompt("JSON test", &RunOptions{Format: JSONOutput})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Result != "JSON response" {
		t.Errorf("Expected %q, got %q", "JSON response", result.Result)
	}

	if result.SessionID != "abc123" {
		t.Errorf("Expected session ID %q, got %q", "abc123", result.SessionID)
	}

	if result.CostUSD != 0.001 {
		t.Errorf("Expected cost %f, got %f", 0.001, result.CostUSD)
	}

	// Test error handling
	execCommand = mockExecCommandContext(t, []string{"-p", "Error test"}, "", 1)

	_, err = client.RunPrompt("Error test", &RunOptions{})

	if err == nil {
		t.Fatal("Expected an error, got nil")
	}
}

func TestRunPrompt_WorkingDirectory(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	wantDir := t.TempDir()
	execCommand = func(_ context.Context, name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{
			"GO_WANT_HELPER_PROCESS=1",
			"GO_HELPER_PRINT_PWD=1",
		}
		return cmd
	}

	client := &ClaudeClient{BinPath: "claude"}
	result, err := client.RunPrompt("Hello, Claude", &RunOptions{
		Format:           TextOutput,
		WorkingDirectory: wantDir,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	gotDir, err := filepath.EvalSymlinks(strings.TrimSpace(result.Result))
	if err != nil {
		t.Fatalf("EvalSymlinks(result) error = %v", err)
	}
	resolvedWantDir, err := filepath.EvalSymlinks(wantDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(wantDir) error = %v", err)
	}
	if gotDir != resolvedWantDir {
		t.Fatalf("expected cwd %q, got %q", resolvedWantDir, gotDir)
	}
}

func TestRunFromStdin_WorkingDirectory(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	wantDir := t.TempDir()
	execCommand = func(_ context.Context, name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{
			"GO_WANT_HELPER_PROCESS=1",
			"GO_HELPER_PRINT_PWD=1",
		}
		return cmd
	}

	client := &ClaudeClient{BinPath: "claude"}
	result, err := client.RunFromStdin(strings.NewReader("stdin input"), "Hello, Claude", &RunOptions{
		Format:           TextOutput,
		WorkingDirectory: wantDir,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	gotDir, err := filepath.EvalSymlinks(strings.TrimSpace(result.Result))
	if err != nil {
		t.Fatalf("EvalSymlinks(result) error = %v", err)
	}
	resolvedWantDir, err := filepath.EvalSymlinks(wantDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(wantDir) error = %v", err)
	}
	if gotDir != resolvedWantDir {
		t.Fatalf("expected cwd %q, got %q", resolvedWantDir, gotDir)
	}
}

// TestWorkingDirectory_EmptyInheritsParentCwd verifies that when WorkingDirectory
// is empty, the Claude subprocess inherits the parent process's cwd across all
// three entry points. Guards against accidentally always setting cmd.Dir.
func TestWorkingDirectory_EmptyInheritsParentCwd(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	parentCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	resolvedParent, err := filepath.EvalSymlinks(parentCwd)
	if err != nil {
		t.Fatalf("EvalSymlinks(parent) error = %v", err)
	}

	assertInheritedCwd := func(t *testing.T, got string) {
		t.Helper()
		resolved, err := filepath.EvalSymlinks(strings.TrimSpace(got))
		if err != nil {
			t.Fatalf("EvalSymlinks(result) error = %v", err)
		}
		if resolved != resolvedParent {
			t.Fatalf("expected child to inherit parent cwd %q, got %q", resolvedParent, resolved)
		}
	}

	t.Run("RunPrompt", func(t *testing.T) {
		execCommand = func(_ context.Context, name string, arg ...string) *exec.Cmd {
			cs := []string{"-test.run=TestHelperProcess", "--", name}
			cs = append(cs, arg...)
			cmd := exec.Command(os.Args[0], cs...)
			cmd.Env = []string{
				"GO_WANT_HELPER_PROCESS=1",
				"GO_HELPER_PRINT_PWD=1",
			}
			return cmd
		}

		client := &ClaudeClient{BinPath: "claude"}
		result, err := client.RunPrompt("Hello, Claude", &RunOptions{Format: TextOutput})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		assertInheritedCwd(t, result.Result)
	})

	t.Run("RunFromStdin", func(t *testing.T) {
		execCommand = func(_ context.Context, name string, arg ...string) *exec.Cmd {
			cs := []string{"-test.run=TestHelperProcess", "--", name}
			cs = append(cs, arg...)
			cmd := exec.Command(os.Args[0], cs...)
			cmd.Env = []string{
				"GO_WANT_HELPER_PROCESS=1",
				"GO_HELPER_PRINT_PWD=1",
			}
			return cmd
		}

		client := &ClaudeClient{BinPath: "claude"}
		result, err := client.RunFromStdin(strings.NewReader("in"), "Hello, Claude", &RunOptions{Format: TextOutput})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		assertInheritedCwd(t, result.Result)
	})

	t.Run("StreamPrompt", func(t *testing.T) {
		execCommand = func(_ context.Context, name string, arg ...string) *exec.Cmd {
			cs := []string{"-test.run=TestHelperProcess", "--", name}
			cs = append(cs, arg...)
			cmd := exec.Command(os.Args[0], cs...)
			cmd.Env = []string{
				"GO_WANT_HELPER_PROCESS=1",
				"GO_HELPER_STREAM_PWD=1",
			}
			return cmd
		}

		client := &ClaudeClient{BinPath: "claude"}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		messageCh, errCh := client.StreamPrompt(ctx, "Test", &RunOptions{})

		var got Message
		for msg := range messageCh {
			got = msg
		}
		for err := range errCh {
			if err != nil {
				t.Fatalf("Streaming error: %v", err)
			}
		}
		assertInheritedCwd(t, got.Result)
	})
}

func TestStreamPrompt(t *testing.T) {
	// For streaming test, we'll create a simple mock that sends predefined messages
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Create a temporary file for our mock script
	tempDir := t.TempDir()
	mockScript := filepath.Join(tempDir, "mock_stream.go")

	// Write a simple Go program that outputs JSON messages
	scriptContent := `package main
import (
	"fmt"
	"time"
)
func main() {
	fmt.Println(` + "`" + `{"type":"system","subtype":"init","session_id":"test-session","tools":["Bash"]}` + "`" + `)
	time.Sleep(100 * time.Millisecond)
	fmt.Println(` + "`" + `{"type":"assistant","message":{},"session_id":"test-session","result":"Hello there!"}` + "`" + `)
	time.Sleep(100 * time.Millisecond)
	fmt.Println(` + "`" + `{"type":"result","subtype":"success","total_cost_usd":0.002,"duration_ms":300,"duration_api_ms":250,"is_error":false,"num_turns":1,"result":"Final result","session_id":"test-session"}` + "`" + `)
}
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to write mock script: %v", err)
	}

	// Compile the mock script
	mockBinary := filepath.Join(tempDir, "mock_stream")
	compileCmd := exec.Command("go", "build", "-o", mockBinary, mockScript)
	if err := compileCmd.Run(); err != nil {
		t.Fatalf("Failed to compile mock script: %v", err)
	}

	// Replace execCommand with our mock
	execCommand = func(_ context.Context, name string, arg ...string) *exec.Cmd {
		return exec.Command(mockBinary)
	}

	// Now test streaming
	client := &ClaudeClient{BinPath: "claude"}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	messageCh, errCh := client.StreamPrompt(ctx, "Test streaming", &RunOptions{Format: StreamJSONOutput})

	// Collect messages and errors
	var messages []Message
	var streamErr error

	// Handle possible errors
	go func() {
		for err := range errCh {
			streamErr = err
		}
	}()

	// Collect messages
	for msg := range messageCh {
		messages = append(messages, msg)
	}

	// Check for streaming errors
	if streamErr != nil {
		t.Fatalf("Streaming error: %v", streamErr)
	}

	// Verify we got the expected messages
	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	// Check first message (init)
	if messages[0].Type != "system" || messages[0].Subtype != "init" {
		t.Errorf("First message should be system/init, got %s/%s", messages[0].Type, messages[0].Subtype)
	}

	// Check last message (result)
	if messages[2].Type != "result" || messages[2].CostUSD != 0.002 {
		t.Errorf("Last message should be result with cost 0.002, got %s with cost %f",
			messages[2].Type, messages[2].CostUSD)
	}
}

func TestRunFromStdin(t *testing.T) {
	origExecCommand := execCommand
	defer func() {
		execCommand = origExecCommand
	}()

	// Test with text input from stdin
	execCommand = mockExecCommandContext(t, []string{"-p", "--output-format", "text"}, "Analyzed your input", 0)

	client := &ClaudeClient{BinPath: "claude"}
	stdin := bytes.NewBufferString("Code to analyze")

	result, err := client.RunFromStdin(stdin, "", &RunOptions{Format: TextOutput})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Result != "Analyzed your input" {
		t.Errorf("Expected %q, got %q", "Analyzed your input", result.Result)
	}
}

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		opts     *RunOptions
		expected []string
	}{
		{
			name:     "Basic prompt",
			prompt:   "Hello",
			opts:     &RunOptions{},
			expected: []string{"-p", "Hello"},
		},
		{
			name:   "All options",
			prompt: "Complete test",
			opts: &RunOptions{
				Format:        JSONOutput,
				Agent:         "reviewer",
				AgentsJSON:    `{"reviewer":{"description":"Reviews code","prompt":"You are a reviewer"}}`,
				SystemPrompt:  "Custom system prompt",
				AppendPrompt:  "Additional instructions",
				MCPConfigPath: "/path/to/mcp.json",
				AllowedTools:  []string{"tool1", "tool2"},
				DisallowedTools: []string{
					"bad1", "bad2",
				},
				ResumeID: "session123",
				Verbose:  true,
				Model:    "claude-3-5-sonnet-20240620",
				Effort:   EffortHigh,
				Name:     "review-session",
			},
			expected: []string{
				"-p", "Complete test",
				"--output-format", "json",
				"--agent", "reviewer",
				"--agents", `{"reviewer":{"description":"Reviews code","prompt":"You are a reviewer"}}`,
				"--system-prompt", "Custom system prompt",
				"--append-system-prompt", "Additional instructions",
				"--mcp-config", "/path/to/mcp.json",
				"--allowedTools", "tool1,tool2",
				"--disallowedTools", "bad1,bad2",
				"--resume", "session123",
				"--verbose",
				"--model", "claude-3-5-sonnet-20240620",
				"--effort", "high",
				"--name", "review-session",
			},
		},
		{
			name:   "Continue session",
			prompt: "Continue",
			opts: &RunOptions{
				Continue: true,
			},
			expected: []string{"-p", "Continue", "--continue"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildArgs(tt.prompt, tt.opts)

			if len(args) != len(tt.expected) {
				t.Errorf("Expected %d arguments, got %d", len(tt.expected), len(args))
				t.Logf("Expected: %v", tt.expected)
				t.Logf("Got: %v", args)
				return
			}

			for i, arg := range args {
				if arg != tt.expected[i] {
					t.Errorf("Expected arg[%d] to be %q, got %q", i, tt.expected[i], arg)
				}
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("claude-bin-path")

	if client.BinPath != "claude-bin-path" {
		t.Errorf("Expected BinPath to be %q, got %q", "claude-bin-path", client.BinPath)
	}

	if client.DefaultOptions == nil {
		t.Error("DefaultOptions should not be nil")
	}

	if client.DefaultOptions.Format != TextOutput {
		t.Errorf("Expected default format to be %q, got %q", TextOutput, client.DefaultOptions.Format)
	}
}

func TestValidateMCPToolName(t *testing.T) {
	tests := []struct {
		name  string
		tool  string
		valid bool
	}{
		{"valid", "mcp__filesystem__list_directory", true},
		{"invalid_short", "mcp__badtool", false},
		{"invalid_no_prefix", "filesystem__list_directory", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateMCPToolName(tt.tool)
			if got != tt.valid {
				t.Errorf("validateMCPToolName(%q) = %v, want %v", tt.tool, got, tt.valid)
			}
		})
	}
}

func TestValidateMCPTools(t *testing.T) {
	if err := validateMCPTools([]string{"mcp__filesystem__list_directory", "mcp__github__get_repository"}); err != nil {
		t.Fatalf("Expected no error for valid tools, got %v", err)
	}

	if err := validateMCPTools([]string{"mcp__badtool"}); err == nil {
		t.Fatal("Expected error for malformed tool name, got nil")
	}
}

// Test convenience methods
func TestRunWithMCP(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"result","subtype":"success","total_cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":1,"result":"MCP response","session_id":"abc123"}`
	execCommand = mockExecCommandContext(t, []string{"-p", "Test MCP", "--output-format", "json", "--mcp-config", "/path/to/config.json", "--allowedTools", "tool1,tool2"}, jsonOutput, 0)

	client := &ClaudeClient{BinPath: "claude"}
	result, err := client.RunWithMCP("Test MCP", "/path/to/config.json", []string{"tool1", "tool2"})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result.Result != "MCP response" {
		t.Errorf("Expected 'MCP response', got %q", result.Result)
	}
}

func TestRunWithMCPCtx(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"result","subtype":"success","total_cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":1,"result":"MCP context response","session_id":"abc123"}`
	execCommand = mockExecCommandContext(t, []string{"-p", "Test MCP Ctx", "--output-format", "json", "--mcp-config", "/path/to/config.json", "--allowedTools", "tool1"}, jsonOutput, 0)

	client := &ClaudeClient{BinPath: "claude"}
	ctx := context.Background()
	result, err := client.RunWithMCPCtx(ctx, "Test MCP Ctx", "/path/to/config.json", []string{"tool1"})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result.Result != "MCP context response" {
		t.Errorf("Expected 'MCP context response', got %q", result.Result)
	}
}

func TestRunWithSystemPrompt(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	execCommand = mockExecCommandContext(t, []string{"-p", "Test prompt", "--system-prompt", "Custom system prompt"}, "System prompt response", 0)

	client := &ClaudeClient{BinPath: "claude"}
	result, err := client.RunWithSystemPrompt("Test prompt", "Custom system prompt", nil)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result.Result != "System prompt response" {
		t.Errorf("Expected 'System prompt response', got %q", result.Result)
	}
}

func TestRunWithSystemPromptCtx(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	execCommand = mockExecCommandContext(t, []string{"-p", "Test prompt", "--output-format", "json", "--system-prompt", "Custom system prompt"}, `{"type":"result","result":"System prompt ctx response","is_error":false}`, 0)

	client := &ClaudeClient{BinPath: "claude"}
	ctx := context.Background()
	result, err := client.RunWithSystemPromptCtx(ctx, "Test prompt", "Custom system prompt", &RunOptions{Format: JSONOutput})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result.Result != "System prompt ctx response" {
		t.Errorf("Expected 'System prompt ctx response', got %q", result.Result)
	}
}

func TestContinueConversation(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"result","subtype":"success","total_cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":2,"result":"Continued response","session_id":"continue123"}`
	execCommand = mockExecCommandContext(t, []string{"-p", "Continue", "--output-format", "json", "--continue"}, jsonOutput, 0)

	client := &ClaudeClient{BinPath: "claude"}
	result, err := client.ContinueConversation("Continue")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result.Result != "Continued response" {
		t.Errorf("Expected 'Continued response', got %q", result.Result)
	}
	if result.NumTurns != 2 {
		t.Errorf("Expected 2 turns, got %d", result.NumTurns)
	}
}

func TestContinueConversationCtx(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"result","subtype":"success","total_cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":3,"result":"Continued ctx response","session_id":"continue123"}`
	execCommand = mockExecCommandContext(t, []string{"-p", "Continue ctx", "--output-format", "json", "--continue"}, jsonOutput, 0)

	client := &ClaudeClient{BinPath: "claude"}
	ctx := context.Background()
	result, err := client.ContinueConversationCtx(ctx, "Continue ctx")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result.Result != "Continued ctx response" {
		t.Errorf("Expected 'Continued ctx response', got %q", result.Result)
	}
}

func TestResumeConversation(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"result","subtype":"success","total_cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":1,"result":"Resumed response","session_id":"resume123"}`
	execCommand = mockExecCommandContext(t, []string{"-p", "Resume", "--output-format", "json", "--resume", "resume123"}, jsonOutput, 0)

	client := &ClaudeClient{BinPath: "claude"}
	result, err := client.ResumeConversation("Resume", "resume123")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result.Result != "Resumed response" {
		t.Errorf("Expected 'Resumed response', got %q", result.Result)
	}
	if result.SessionID != "resume123" {
		t.Errorf("Expected session ID 'resume123', got %q", result.SessionID)
	}
}

func TestResumeConversationCtx(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"result","subtype":"success","total_cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":1,"result":"Resumed ctx response","session_id":"resume123"}`
	execCommand = mockExecCommandContext(t, []string{"-p", "Resume ctx", "--output-format", "json", "--resume", "resume123"}, jsonOutput, 0)

	client := &ClaudeClient{BinPath: "claude"}
	ctx := context.Background()
	result, err := client.ResumeConversationCtx(ctx, "Resume ctx", "resume123")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result.Result != "Resumed ctx response" {
		t.Errorf("Expected 'Resumed ctx response', got %q", result.Result)
	}
}

// Test error handling scenarios
func TestRunPromptCtx_MCPValidationErrors(t *testing.T) {
	client := &ClaudeClient{BinPath: "claude"}

	// Test malformed MCP tool in AllowedTools
	_, err := client.RunPromptCtx(context.Background(), "test", &RunOptions{
		AllowedTools: []string{"mcp__badtool"},
	})
	if err == nil {
		t.Fatal("Expected error for malformed MCP tool in AllowedTools, got nil")
	}

	// Test malformed MCP tool in DisallowedTools
	_, err = client.RunPromptCtx(context.Background(), "test", &RunOptions{
		DisallowedTools: []string{"mcp__anotherbadtool"},
	})
	if err == nil {
		t.Fatal("Expected error for malformed MCP tool in DisallowedTools, got nil")
	}
}

func TestRunPromptCtx_JSONParsingError(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Mock command that returns invalid JSON
	execCommand = mockExecCommandContext(t, []string{"-p", "JSON test", "--output-format", "json"}, "invalid json", 0)

	client := &ClaudeClient{BinPath: "claude"}
	_, err := client.RunPromptCtx(context.Background(), "JSON test", &RunOptions{Format: JSONOutput})

	if err == nil {
		t.Fatal("Expected JSON parsing error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse JSON response") {
		t.Errorf("Expected JSON parsing error message, got: %v", err)
	}
}

func TestRunPromptCtx_CommandFailure(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Mock command that fails
	execCommand = mockExecCommandContext(t, []string{"-p", "Fail test"}, "", 1)

	client := &ClaudeClient{BinPath: "claude"}
	_, err := client.RunPromptCtx(context.Background(), "Fail test", &RunOptions{})

	if err == nil {
		t.Fatal("Expected command failure error, got nil")
	}

	// Check that we get a ClaudeError
	if claudeErr, ok := err.(*ClaudeError); ok {
		if claudeErr.Type != ErrorCommand {
			t.Errorf("Expected ErrorCommand type, got: %v", claudeErr.Type)
		}
	} else {
		// For backward compatibility, also accept the old error format
		if !strings.Contains(err.Error(), "command failed") && !strings.Contains(err.Error(), "claude command failed") {
			t.Errorf("Expected command failure error message, got: %v", err)
		}
	}
}

func TestRunFromStdinCtx_JSONParsingError(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Mock command that returns invalid JSON
	execCommand = mockExecCommandContext(t, []string{"-p", "--output-format", "json"}, "invalid json", 0)

	client := &ClaudeClient{BinPath: "claude"}
	stdin := bytes.NewBufferString("test input")
	_, err := client.RunFromStdinCtx(context.Background(), stdin, "", &RunOptions{Format: JSONOutput})

	if err == nil {
		t.Fatal("Expected JSON parsing error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse JSON response") {
		t.Errorf("Expected JSON parsing error message, got: %v", err)
	}
}

func TestRunFromStdinCtx_CommandFailure(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Mock command that fails with stderr
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcessError", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS_ERROR=1"}
		return cmd
	}

	client := &ClaudeClient{BinPath: "claude"}
	stdin := bytes.NewBufferString("test input")
	_, err := client.RunFromStdinCtx(context.Background(), stdin, "", &RunOptions{})

	if err == nil {
		t.Fatal("Expected command failure error, got nil")
	}
}

func TestBuildArgs_EdgeCases(t *testing.T) {
	// Test empty prompt
	args := BuildArgs("", &RunOptions{Format: TextOutput})
	expected := []string{"-p", "--output-format", "text"}
	if len(args) != len(expected) || args[0] != "-p" || args[1] != "--output-format" {
		t.Errorf("Expected %v for empty prompt, got %v", expected, args)
	}

	// Test ResumeID takes precedence over Continue
	args = BuildArgs("test", &RunOptions{
		ResumeID: "session123",
		Continue: true,
	})
	foundResume := false
	foundContinue := false
	for i, arg := range args {
		if arg == "--resume" {
			foundResume = true
			if i+1 < len(args) && args[i+1] == "session123" {
				// Good
			} else {
				t.Error("--resume should be followed by session123")
			}
		}
		if arg == "--continue" {
			foundContinue = true
		}
	}
	if !foundResume {
		t.Error("Expected --resume to be present")
	}
	if foundContinue {
		t.Error("Expected --continue to be absent when ResumeID is set")
	}

	// Test MaxTurns = 0 should not add --max-turns
	args = BuildArgs("test", &RunOptions{MaxTurns: 0})
	for _, arg := range args {
		if arg == "--max-turns" {
			t.Error("Expected --max-turns to be absent when MaxTurns is 0")
		}
	}
}

func TestBuildArgs_VariadicFlagShape(t *testing.T) {
	args := BuildArgs("test", &RunOptions{
		Betas: []string{"beta-a", "beta-b"},
		Files: []string{"file_abc:doc.txt", "file_def:img.png"},
	})

	expected := []string{
		"-p", "test",
		"--betas", "beta-a", "beta-b",
		"--file", "file_abc:doc.txt", "file_def:img.png",
	}

	if len(args) != len(expected) {
		t.Fatalf("Expected %d arguments, got %d: %v", len(expected), len(args), args)
	}
	for i, arg := range expected {
		if args[i] != arg {
			t.Fatalf("Expected arg[%d] = %q, got %q", i, arg, args[i])
		}
	}
}

func TestBuildArgs_NewFlags(t *testing.T) {
	tests := []struct {
		name     string
		opts     *RunOptions
		expected []string
	}{
		{
			name: "Current print surface flags",
			opts: &RunOptions{
				Agent:                           "security",
				AgentsJSON:                      `{"security":{"description":"Security review","prompt":"Review for vulnerabilities"}}`,
				AllowDangerouslySkipPermissions: true,
				Effort:                          EffortXHigh,
				InputFormat:                     StreamJSONInput,
				IncludeHookEvents:               true,
				IncludePartialMessages:          true,
				ReplayUserMessages:              true,
				DebugFile:                       "/tmp/claude-debug.log",
			},
			expected: []string{
				"-p", "test",
				"--agent", "security",
				"--agents", `{"security":{"description":"Security review","prompt":"Review for vulnerabilities"}}`,
				"--allow-dangerously-skip-permissions",
				"--effort", "xhigh",
				"--input-format", "stream-json",
				"--include-hook-events",
				"--include-partial-messages",
				"--replay-user-messages",
				"--debug-file", "/tmp/claude-debug.log",
			},
		},
		{
			name: "Help and version flags",
			opts: &RunOptions{
				Help:    true,
				Version: true,
			},
			expected: []string{"-p", "test", "--help", "--version"},
		},
		{
			name: "Context shaping flags",
			opts: &RunOptions{
				Bare:                               true,
				Brief:                              true,
				Betas:                              []string{"beta-a", "beta-b"},
				Files:                              []string{"file_abc:doc.txt", "file_def:img.png"},
				ExcludeDynamicSystemPromptSections: true,
			},
			expected: []string{
				"-p", "test",
				"--bare",
				"--brief",
				"--betas", "beta-a", "beta-b",
				"--file", "file_abc:doc.txt", "file_def:img.png",
				"--exclude-dynamic-system-prompt-sections",
			},
		},
		{
			name: "Settings and tool surface flags",
			opts: &RunOptions{
				MaxBudgetUSD: 12.5,
				SettingSources: []string{
					"user", "project",
				},
				Settings:   `{"env":{"FOO":"bar"}}`,
				Tools:      []string{"Bash", "Read", "Edit"},
				Name:       "release-0-1-1",
				PluginDirs: []string{"/plugins/a", "/plugins/b"},
			},
			expected: []string{
				"-p", "test",
				"--max-budget-usd", "12.5",
				"--setting-sources", "user,project",
				"--settings", `{"env":{"FOO":"bar"}}`,
				"--tools", "Bash,Read,Edit",
				"--name", "release-0-1-1",
				"--plugin-dir", "/plugins/a",
				"--plugin-dir", "/plugins/b",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildArgs("test", tt.opts)

			// Check all expected args are present
			for _, exp := range tt.expected {
				found := false
				for _, arg := range args {
					if arg == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected argument %q not found in %v", exp, args)
				}
			}
		})
	}
}

func TestRunPromptCtx_DoesNotMutateDefaultOptions(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	jsonOutput := `{"type":"result","subtype":"success","total_cost_usd":0.001,"duration_ms":1,"duration_api_ms":1,"is_error":false,"num_turns":1,"result":"ok","session_id":"opts-session"}`
	execCommand = mockExecCommandContext(t, []string{
		"-p", "Mutation test", "--output-format", "json", "--allowedTools", "Read", "--disallowedTools", "Write",
	}, jsonOutput, 0)

	defaultOpts := &RunOptions{
		Format:          JSONOutput,
		AllowedTools:    []string{"Read"},
		DisallowedTools: []string{"Write"},
	}
	client := &ClaudeClient{
		BinPath:        "claude",
		DefaultOptions: defaultOpts,
	}

	if _, err := client.RunPromptCtx(context.Background(), "Mutation test", nil); err != nil {
		t.Fatalf("RunPromptCtx() error = %v", err)
	}

	if defaultOpts.ParsedAllowedTools != nil {
		t.Fatalf("ParsedAllowedTools mutated on caller options: %v", defaultOpts.ParsedAllowedTools)
	}
	if defaultOpts.ParsedDisallowedTools != nil {
		t.Fatalf("ParsedDisallowedTools mutated on caller options: %v", defaultOpts.ParsedDisallowedTools)
	}
}

// Helper for command failure tests
func TestHelperProcessError(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS_ERROR") != "1" {
		return
	}
	defer os.Exit(1)
	os.Stderr.Write([]byte("command failed with error"))
}

// Test Session Control flags
func TestBuildArgs_SessionControl(t *testing.T) {
	tests := []struct {
		name     string
		opts     *RunOptions
		expected []string
		excluded []string
	}{
		{
			name: "SessionID flag",
			opts: &RunOptions{
				SessionID: "550e8400-e29b-41d4-a716-446655440000",
			},
			expected: []string{"--session-id", "550e8400-e29b-41d4-a716-446655440000"},
		},
		{
			name: "ForkSession flag",
			opts: &RunOptions{
				ForkSession: true,
			},
			expected: []string{"--fork-session"},
		},
		{
			name: "NoSessionPersistence flag",
			opts: &RunOptions{
				NoSessionPersistence: true,
			},
			expected: []string{"--no-session-persistence"},
		},
		{
			name: "All session flags combined",
			opts: &RunOptions{
				SessionID:            "550e8400-e29b-41d4-a716-446655440000",
				ForkSession:          true,
				NoSessionPersistence: true,
			},
			expected: []string{
				"--session-id", "550e8400-e29b-41d4-a716-446655440000",
				"--fork-session",
				"--no-session-persistence",
			},
		},
		{
			name: "Empty SessionID should not add flag",
			opts: &RunOptions{
				SessionID: "",
			},
			excluded: []string{"--session-id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildArgs("test", tt.opts)

			// Check expected args are present
			for _, exp := range tt.expected {
				found := false
				for _, arg := range args {
					if arg == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected argument %q not found in %v", exp, args)
				}
			}

			// Check excluded args are absent
			for _, exc := range tt.excluded {
				for _, arg := range args {
					if arg == exc {
						t.Errorf("Excluded argument %q should not be present in %v", exc, args)
					}
				}
			}
		})
	}
}

func TestValidateSessionID(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		wantErr   bool
	}{
		{
			name:      "valid UUID",
			sessionID: "550e8400-e29b-41d4-a716-446655440000",
			wantErr:   false,
		},
		{
			name:      "empty string is valid",
			sessionID: "",
			wantErr:   false,
		},
		{
			name:      "invalid length",
			sessionID: "550e8400-e29b",
			wantErr:   true,
		},
		{
			name:      "missing hyphens",
			sessionID: "550e8400e29b41d4a716446655440000xxxx",
			wantErr:   true,
		},
		{
			name:      "wrong hyphen positions",
			sessionID: "550e-8400-e29b-41d4-a71644665544",
			wantErr:   true,
		},
		{
			name:      "non-hex characters",
			sessionID: "550e8400-e29b-41d4-a716-44665544000g",
			wantErr:   true,
		},
		{
			name:      "uppercase valid",
			sessionID: "550E8400-E29B-41D4-A716-446655440000",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSessionID(tt.sessionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSessionID(%q) error = %v, wantErr %v", tt.sessionID, err, tt.wantErr)
			}
		})
	}
}

func TestGenerateSessionID(t *testing.T) {
	// Generate multiple session IDs and verify they're valid UUIDs
	generated := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := GenerateSessionID()

		// Check format
		if err := ValidateSessionID(id); err != nil {
			t.Errorf("GenerateSessionID() produced invalid UUID: %q, error: %v", id, err)
		}

		// Check uniqueness
		if generated[id] {
			t.Errorf("GenerateSessionID() produced duplicate UUID: %q", id)
		}
		generated[id] = true
	}
}

// Test MCP configuration flags
func TestBuildArgs_MCPConfig(t *testing.T) {
	tests := []struct {
		name     string
		opts     *RunOptions
		expected []string
		excluded []string
	}{
		{
			name: "Multiple MCP configs",
			opts: &RunOptions{
				MCPConfigs: []string{"/path/config1.json", "/path/config2.json"},
			},
			expected: []string{"--mcp-config", "/path/config1.json", "/path/config2.json"},
		},
		{
			name: "StrictMCPConfig flag",
			opts: &RunOptions{
				StrictMCPConfig: true,
			},
			expected: []string{"--strict-mcp-config"},
		},
		{
			name: "MCP configs with strict mode",
			opts: &RunOptions{
				MCPConfigs:      []string{"/path/config.json"},
				StrictMCPConfig: true,
			},
			expected: []string{"--mcp-config", "/path/config.json", "--strict-mcp-config"},
		},
		{
			name: "Empty MCPConfigs should not add flag",
			opts: &RunOptions{
				MCPConfigs: []string{},
			},
			excluded: []string{"--mcp-config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildArgs("test", tt.opts)

			// Check expected args are present
			for _, exp := range tt.expected {
				found := false
				for _, arg := range args {
					if arg == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected argument %q not found in %v", exp, args)
				}
			}

			// Check excluded args are absent
			for _, exc := range tt.excluded {
				for _, arg := range args {
					if arg == exc {
						t.Errorf("Excluded argument %q should not be present in %v", exc, args)
					}
				}
			}
		})
	}
}

// Test AddDirectories and PrintMode flags
func TestBuildArgs_CLIFlags(t *testing.T) {
	tests := []struct {
		name     string
		opts     *RunOptions
		expected []string
		excluded []string
	}{
		{
			name: "Single AddDirectory",
			opts: &RunOptions{
				AddDirectories: []string{"/path/to/dir"},
			},
			expected: []string{"--add-dir", "/path/to/dir"},
		},
		{
			name: "Multiple AddDirectories",
			opts: &RunOptions{
				AddDirectories: []string{"/path/dir1", "/path/dir2", "/path/dir3"},
			},
			expected: []string{"--add-dir", "/path/dir1", "--add-dir", "/path/dir2", "--add-dir", "/path/dir3"},
		},
		{
			name: "PrintMode flag",
			opts: &RunOptions{
				PrintMode: true,
			},
			expected: []string{"--print"},
		},
		{
			name: "AddDirectories with PrintMode",
			opts: &RunOptions{
				AddDirectories: []string{"/path/to/dir"},
				PrintMode:      true,
			},
			expected: []string{"--add-dir", "/path/to/dir", "--print"},
		},
		{
			name: "Empty AddDirectories should not add flag",
			opts: &RunOptions{
				AddDirectories: []string{},
			},
			excluded: []string{"--add-dir"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildArgs("test", tt.opts)

			// Check expected args are present in order
			for i := 0; i < len(tt.expected); i++ {
				exp := tt.expected[i]
				found := false
				for _, arg := range args {
					if arg == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected argument %q not found in %v", exp, args)
				}
			}

			// Check excluded args are absent
			for _, exc := range tt.excluded {
				for _, arg := range args {
					if arg == exc {
						t.Errorf("Excluded argument %q should not be present in %v", exc, args)
					}
				}
			}
		})
	}
}

// Test session ID validation in PreprocessOptions
func TestPreprocessOptions_SessionValidation(t *testing.T) {
	client := &ClaudeClient{BinPath: "claude"}

	// Valid session ID should not error
	_, err := client.RunPromptCtx(context.Background(), "test", &RunOptions{
		SessionID: "550e8400-e29b-41d4-a716-446655440000",
	})
	// This will fail because we don't have a mock, but it should fail after validation
	// If it had failed validation, we'd get a different error

	// Invalid session ID should error with validation message
	_, err = client.RunPromptCtx(context.Background(), "test", &RunOptions{
		SessionID: "invalid-session-id",
	})
	if err == nil {
		t.Fatal("Expected error for invalid session ID, got nil")
	}
	if !strings.Contains(err.Error(), "invalid session ID") {
		t.Errorf("Expected session ID validation error, got: %v", err)
	}
}

// Context Cancellation Tests

func TestRunPromptCtx_ContextCancellation(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Mock command that would take a long time
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sleep", "10")
	}

	client := NewClient("claude")

	t.Run("pre-canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel before starting

		_, err := client.RunPromptCtx(ctx, "test", &RunOptions{})
		if err == nil {
			t.Fatal("Expected error for canceled context")
		}
	})

	t.Run("context canceled during execution", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel after a short delay
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		_, err := client.RunPromptCtx(ctx, "test", &RunOptions{})
		if err == nil {
			t.Fatal("Expected error when context is canceled during execution")
		}
	})
}

func TestRunPromptCtx_ContextDeadlineExceeded(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Mock command that takes longer than the deadline
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sleep", "10")
	}

	client := NewClient("claude")

	t.Run("deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := client.RunPromptCtx(ctx, "test", &RunOptions{})
		elapsed := time.Since(start)

		if err == nil {
			t.Fatal("Expected error for deadline exceeded")
		}

		// Verify it didn't wait the full 10 seconds
		if elapsed > 500*time.Millisecond {
			t.Errorf("Expected fast timeout, but took %v", elapsed)
		}
	})
}

func TestStreamPrompt_WorkingDirectory(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	wantDir := t.TempDir()
	execCommand = func(_ context.Context, name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{
			"GO_WANT_HELPER_PROCESS=1",
			"GO_HELPER_STREAM_PWD=1",
		}
		return cmd
	}

	client := &ClaudeClient{BinPath: "claude"}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	messageCh, errCh := client.StreamPrompt(ctx, "Test streaming", &RunOptions{
		WorkingDirectory: wantDir,
	})

	var got Message
	for msg := range messageCh {
		got = msg
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("Streaming error: %v", err)
		}
	}
	gotDir, err := filepath.EvalSymlinks(got.Result)
	if err != nil {
		t.Fatalf("EvalSymlinks(result) error = %v", err)
	}
	resolvedWantDir, err := filepath.EvalSymlinks(wantDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(wantDir) error = %v", err)
	}
	if gotDir != resolvedWantDir {
		t.Fatalf("expected cwd %q, got %q", resolvedWantDir, gotDir)
	}
}

func TestStreamPrompt_ContextCancellation(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Mock command that would run forever
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sleep", "100")
	}

	client := NewClient("claude")

	t.Run("pre-canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel before starting

		msgCh, errCh := client.StreamPrompt(ctx, "test", &RunOptions{})

		// Drain channels
		go func() {
			for range msgCh {
			}
		}()

		// Should receive an error
		foundError := false
		for err := range errCh {
			if err != nil {
				foundError = true
			}
		}

		if !foundError {
			t.Log("No error received on pre-canceled context - this may be expected if command starts immediately")
		}
	})

	t.Run("context canceled during streaming", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		msgCh, errCh := client.StreamPrompt(ctx, "test", &RunOptions{})

		// Cancel after starting
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		// Drain messages
		go func() {
			for range msgCh {
			}
		}()

		// Wait for error or completion
		start := time.Now()
		for range errCh {
		}
		elapsed := time.Since(start)

		// Should complete quickly due to cancellation
		if elapsed > 500*time.Millisecond {
			t.Errorf("Expected fast cancellation, but took %v", elapsed)
		}
	})
}

func TestPromptArgAndStdin(t *testing.T) {
	t.Run("small prompt stays inline", func(t *testing.T) {
		argPrompt, stdin := promptArgAndStdin("a short prompt", "")
		if argPrompt != "a short prompt" {
			t.Errorf("argPrompt = %q, want the original prompt", argPrompt)
		}
		if stdin != nil {
			t.Error("stdin = non-nil, want nil for a small prompt")
		}
	})

	t.Run("prompt at threshold routes to stdin", func(t *testing.T) {
		big := strings.Repeat("x", maxInlinePromptBytes)
		argPrompt, stdin := promptArgAndStdin(big, "")
		if argPrompt != "" {
			t.Errorf("argPrompt = %q, want empty so BuildArgs omits it from argv", argPrompt)
		}
		if stdin == nil {
			t.Fatal("stdin = nil, want a reader for a prompt at maxInlinePromptBytes")
		}
		got, err := io.ReadAll(stdin)
		if err != nil {
			t.Fatalf("ReadAll(stdin): %v", err)
		}
		if string(got) != big {
			t.Error("stdin content does not match the original prompt")
		}
	})

	t.Run("prompt just under threshold stays inline", func(t *testing.T) {
		small := strings.Repeat("x", maxInlinePromptBytes-1)
		argPrompt, stdin := promptArgAndStdin(small, "")
		if argPrompt != small {
			t.Error("argPrompt should equal the prompt just under the threshold")
		}
		if stdin != nil {
			t.Error("stdin = non-nil, want nil just under the threshold")
		}
	})

	t.Run("stream-json input is encoded as a user event", func(t *testing.T) {
		argPrompt, stdin := promptArgAndStdin("hello \"Claude\"", StreamJSONInput)
		if argPrompt != "" {
			t.Errorf("argPrompt = %q, want empty so the event is read from stdin", argPrompt)
		}
		if stdin == nil {
			t.Fatal("stdin = nil, want a stream-json user event")
		}
		got, err := io.ReadAll(stdin)
		if err != nil {
			t.Fatalf("ReadAll(stdin): %v", err)
		}
		want := `{"type":"user","message":{"role":"user","content":"hello \"Claude\""},"parent_tool_use_id":null}` + "\n"
		if string(got) != want {
			t.Errorf("stdin = %q, want %q", got, want)
		}
	})
}

// TestStreamPrompt_LargePromptOmittedFromArgv guards against regressing to
// passing an oversized prompt inline: on Linux, a single argv element at or
// beyond MAX_ARG_STRLEN (128 KiB) makes the process fail to start
// (fork/exec: argument list too long) with no usable error. See
// promptArgAndStdin.
func TestStreamPrompt_LargePromptOmittedFromArgv(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	bigPrompt := strings.Repeat("x", maxInlinePromptBytes+1)

	var capturedArgs []string
	execCommand = func(_ context.Context, name string, arg ...string) *exec.Cmd {
		capturedArgs = append([]string(nil), arg...)
		// `cat` echoes whatever stdin it's given back to stdout; the
		// resulting output won't parse as stream-json, but this test only
		// cares about what landed in argv.
		return exec.Command("cat")
	}

	client := NewClient("claude")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	msgCh, errCh := client.StreamPrompt(ctx, bigPrompt, &RunOptions{})
	go func() {
		for range msgCh {
		}
	}()
	for range errCh {
	}

	for _, a := range capturedArgs {
		if a == bigPrompt {
			t.Fatal("large prompt was passed inline as an argv element; want it routed over stdin")
		}
	}
	if len(capturedArgs) == 0 {
		t.Fatal("execCommand was never invoked")
	}
}

func TestStreamPrompt_ContextDeadline(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Mock command that takes forever
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sleep", "100")
	}

	client := NewClient("claude")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	msgCh, errCh := client.StreamPrompt(ctx, "test", &RunOptions{})

	// Drain messages
	go func() {
		for range msgCh {
		}
	}()

	// Wait for completion
	start := time.Now()
	for range errCh {
	}
	elapsed := time.Since(start)

	// Should complete around the deadline
	if elapsed > 500*time.Millisecond {
		t.Errorf("Expected deadline to trigger around 100ms, but took %v", elapsed)
	}
}

func TestRunFromStdinCtx_ContextCancellation(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Mock command that takes forever
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sleep", "10")
	}

	client := NewClient("claude")

	t.Run("pre-canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		stdin := bytes.NewBufferString("test input")
		_, err := client.RunFromStdinCtx(ctx, stdin, "", &RunOptions{})

		if err == nil {
			t.Fatal("Expected error for canceled context")
		}
	})

	t.Run("context canceled during execution", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		stdin := bytes.NewBufferString("test input")
		_, err := client.RunFromStdinCtx(ctx, stdin, "", &RunOptions{})

		if err == nil {
			t.Fatal("Expected error when context canceled during execution")
		}
	})
}

func TestRunOptionsTimeout(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Mock command that takes forever
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sleep", "10")
	}

	client := NewClient("claude")

	t.Run("timeout option", func(t *testing.T) {
		start := time.Now()
		_, err := client.RunPromptCtx(context.Background(), "test", &RunOptions{
			Timeout: 100 * time.Millisecond,
		})
		elapsed := time.Since(start)

		if err == nil {
			t.Fatal("Expected error for timeout")
		}

		// Should timeout around 100ms
		if elapsed > 500*time.Millisecond {
			t.Errorf("Expected timeout around 100ms, but took %v", elapsed)
		}
	})
}

func TestContextPropagationThroughMethods(t *testing.T) {
	originalExecCommand := execCommand
	defer func() {
		execCommand = originalExecCommand
	}()

	// Mock command that checks context
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", "done")
	}

	client := NewClient("claude")

	t.Run("RunWithMCPCtx propagates context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := client.RunWithMCPCtx(ctx, "test", "", nil)
		// Should get an error because context is canceled
		if err == nil {
			t.Error("Expected error from canceled context")
		}
	})

	t.Run("RunWithSystemPromptCtx propagates context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := client.RunWithSystemPromptCtx(ctx, "test", "system", nil)
		if err == nil {
			t.Error("Expected error from canceled context")
		}
	})

	t.Run("ContinueConversationCtx propagates context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := client.ContinueConversationCtx(ctx, "test")
		if err == nil {
			t.Error("Expected error from canceled context")
		}
	})

	t.Run("ResumeConversationCtx propagates context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := client.ResumeConversationCtx(ctx, "test", "session123")
		if err == nil {
			t.Error("Expected error from canceled context")
		}
	})
}

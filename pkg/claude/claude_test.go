package claude

import (
	"bytes"
	"context"
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
	jsonOutput := `{"type":"result","subtype":"success","cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":1,"result":"JSON response","session_id":"abc123"}`
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
	fmt.Println(` + "`" + `{"type":"result","subtype":"success","cost_usd":0.002,"duration_ms":300,"duration_api_ms":250,"is_error":false,"num_turns":1,"result":"Final result","session_id":"test-session"}` + "`" + `)
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
				Format:          JSONOutput,
				SystemPrompt:    "Custom system prompt",
				AppendPrompt:    "Additional instructions",
				MCPConfigPath:   "/path/to/mcp.json",
				AllowedTools:    []string{"tool1", "tool2"},
				DisallowedTools: []string{"bad1", "bad2"},
				PermissionTool:  "permit_tool",
				ResumeID:        "session123",
				MaxTurns:        5,
				Verbose:         true,
				Model:           "claude-3-5-sonnet-20240620",
			},
			expected: []string{
				"-p", "Complete test",
				"--output-format", "json",
				"--system-prompt", "Custom system prompt",
				"--append-system-prompt", "Additional instructions",
				"--mcp-config", "/path/to/mcp.json",
				"--allowedTools", "tool1,tool2",
				"--disallowedTools", "bad1,bad2",
				"--permission-prompt-tool", "permit_tool",
				"--resume", "session123",
				"--max-turns", "5",
				"--verbose",
				"--model", "claude-3-5-sonnet-20240620",
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

	jsonOutput := `{"type":"result","subtype":"success","cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":1,"result":"MCP response","session_id":"abc123"}`
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

	jsonOutput := `{"type":"result","subtype":"success","cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":1,"result":"MCP context response","session_id":"abc123"}`
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

	jsonOutput := `{"type":"result","subtype":"success","cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":2,"result":"Continued response","session_id":"continue123"}`
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

	jsonOutput := `{"type":"result","subtype":"success","cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":3,"result":"Continued ctx response","session_id":"continue123"}`
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

	jsonOutput := `{"type":"result","subtype":"success","cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":1,"result":"Resumed response","session_id":"resume123"}`
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

	jsonOutput := `{"type":"result","subtype":"success","cost_usd":0.001,"duration_ms":1234,"duration_api_ms":1000,"is_error":false,"num_turns":1,"result":"Resumed ctx response","session_id":"resume123"}`
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

func TestBuildArgs_NewFlags(t *testing.T) {
	tests := []struct {
		name     string
		opts     *RunOptions
		expected []string
	}{
		{
			name: "Config file flag",
			opts: &RunOptions{
				ConfigFile: "/path/to/config.json",
			},
			expected: []string{"-p", "test", "--config", "/path/to/config.json"},
		},
		{
			name: "Help flag",
			opts: &RunOptions{
				Help: true,
			},
			expected: []string{"-p", "test", "--help"},
		},
		{
			name: "Version flag",
			opts: &RunOptions{
				Version: true,
			},
			expected: []string{"-p", "test", "--version"},
		},
		{
			name: "Disable autoupdate flag",
			opts: &RunOptions{
				DisableAutoUpdate: true,
			},
			expected: []string{"-p", "test", "--disable-autoupdate"},
		},
		{
			name: "Theme flag",
			opts: &RunOptions{
				Theme: "dark",
			},
			expected: []string{"-p", "test", "--theme", "dark"},
		},
		{
			name: "All new flags combined",
			opts: &RunOptions{
				ConfigFile:        "/config.json",
				Help:              true,
				Version:           true,
				DisableAutoUpdate: true,
				Theme:             "light",
			},
			expected: []string{"-p", "test", "--config", "/config.json", "--help", "--version", "--disable-autoupdate", "--theme", "light"},
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

// Helper for command failure tests
func TestHelperProcessError(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS_ERROR") != "1" {
		return
	}
	defer os.Exit(1)
	os.Stderr.Write([]byte("command failed with error"))
}

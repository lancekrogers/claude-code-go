package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	twoModelsPayload = `{"models":[` +
		`{"value":"default","resolvedModel":"claude-opus-5-5","displayName":"Default (recommended)","description":"Opus 5.5","supportsEffort":true,"supportedEffortLevels":["low","medium","high","xhigh","max"],"supportsAdaptiveThinking":true,"supportsFastMode":true,"supportsAutoMode":true},` +
		`{"value":"fable","resolvedModel":"claude-fable-5-2","displayName":"Fable 5.2","description":"For your toughest challenges","supportsEffort":true,"supportedEffortLevels":["low","high"]},` +
		`{"value":"claude-haiku-4-5-20251001","displayName":"Haiku 4.5","description":"Fastest for quick answers"}` +
		`]}`
	controlRequestIDPlaceholder = "{{ID}}"
)

func successResponse(payload string) string {
	return `{"type":"control_response","response":{"subtype":"success","request_id":"{{ID}}","response":` + payload + `}}` + "\n"
}

// TestControlHelperProcess isn't a real test. It stands in for the claude CLI
// in SupportedModels tests: it reads the initialize request from stdin, writes
// CONTROL_HELPER_PREAMBLE and then CONTROL_HELPER_RESPONSE (or the contents of
// CONTROL_HELPER_RESPONSE_FILE) with {{ID}} replaced by the request's ID, and
// records all of stdin in CONTROL_HELPER_CAPTURE once stdin closes.
func TestControlHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CONTROL_HELPER") != "1" {
		return
	}

	stdin := bufio.NewReader(os.Stdin)
	first, _ := stdin.ReadString('\n')
	var request struct {
		RequestID string `json:"request_id"`
	}
	_ = json.Unmarshal([]byte(first), &request)

	captureRest := func() {
		rest, _ := io.ReadAll(stdin)
		if path := os.Getenv("CONTROL_HELPER_CAPTURE"); path != "" {
			_ = os.WriteFile(path, []byte(first+string(rest)), 0600)
		}
	}

	switch os.Getenv("CONTROL_HELPER_MODE") {
	case "exit":
		os.Stderr.WriteString(os.Getenv("CONTROL_HELPER_STDERR"))
		code, _ := strconv.Atoi(os.Getenv("CONTROL_HELPER_EXIT"))
		os.Exit(code)
	case "hang":
		if path := os.Getenv("CONTROL_HELPER_READY"); path != "" {
			_ = os.WriteFile(path, nil, 0600)
		}
		time.Sleep(time.Minute)
		os.Exit(0)
	}

	response := os.Getenv("CONTROL_HELPER_RESPONSE")
	if path := os.Getenv("CONTROL_HELPER_RESPONSE_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			os.Exit(2)
		}
		response = string(data)
	}
	os.Stdout.WriteString(os.Getenv("CONTROL_HELPER_PREAMBLE"))
	os.Stdout.WriteString(strings.ReplaceAll(response, controlRequestIDPlaceholder, request.RequestID))

	if os.Getenv("CONTROL_HELPER_MODE") == "linger" {
		time.Sleep(time.Minute)
	}
	captureRest()
	os.Exit(0)
}

type controlFake struct {
	calls   int
	args    []string
	cmd     *exec.Cmd
	capture string
}

// useControlFake replaces execCommand with the helper process. It uses
// exec.Command, not exec.CommandContext, so tests prove SupportedModels kills
// and reaps the child itself rather than relying on the context.
func useControlFake(t *testing.T, env ...string) *controlFake {
	t.Helper()
	original := execCommand
	t.Cleanup(func() { execCommand = original })

	fake := &controlFake{capture: filepath.Join(t.TempDir(), "stdin.jsonl")}
	execCommand = func(_ context.Context, _ string, arg ...string) *exec.Cmd {
		fake.calls++
		fake.args = append([]string(nil), arg...)
		cmd := exec.Command(os.Args[0], "-test.run=^TestControlHelperProcess$")
		cmd.Env = append([]string{
			"GO_WANT_CONTROL_HELPER=1",
			"CONTROL_HELPER_CAPTURE=" + fake.capture,
		}, env...)
		fake.cmd = cmd
		return cmd
	}
	return fake
}

func (f *controlFake) assertReaped(t *testing.T) {
	t.Helper()
	if f.cmd == nil || f.cmd.ProcessState == nil {
		t.Fatal("child process was not waited on")
	}
}

func setControlExitGrace(t *testing.T, d time.Duration) {
	t.Helper()
	original := controlExitGrace
	controlExitGrace = d
	t.Cleanup(func() { controlExitGrace = original })
}

func TestSupportedModelsReturnsModels(t *testing.T) {
	fake := useControlFake(t, "CONTROL_HELPER_RESPONSE="+successResponse(twoModelsPayload))
	client := NewClient("claude")

	models, err := client.SupportedModels(context.Background(), nil)
	if err != nil {
		t.Fatalf("SupportedModels() error = %v", err)
	}

	values := make([]string, len(models))
	for i, m := range models {
		values[i] = m.Value
	}
	if want := []string{"default", "fable", "claude-haiku-4-5-20251001"}; !slices.Equal(values, want) {
		t.Fatalf("values = %v, want %v", values, want)
	}
	fable := models[1]
	if fable.DisplayName != "Fable 5.2" || fable.ResolvedModel != "claude-fable-5-2" || fable.Description != "For your toughest challenges" {
		t.Errorf("fable = %+v", fable)
	}
	if !fable.SupportsEffort || !slices.Equal(fable.SupportedEffortLevels, []EffortLevel{EffortLow, EffortHigh}) {
		t.Errorf("fable effort = %v %v", fable.SupportsEffort, fable.SupportedEffortLevels)
	}
	if !models[0].SupportsFastMode || !models[0].SupportsAutoMode || !models[0].SupportsAdaptiveThinking {
		t.Errorf("default capabilities = %+v", models[0])
	}
	if models[2].SupportsEffort || models[2].ResolvedModel != "" {
		t.Errorf("haiku 4.5 = %+v, want absent fields left zero", models[2])
	}
	fake.assertReaped(t)
}

func TestSupportedModelsSendsOnlyInitialize(t *testing.T) {
	fake := useControlFake(t, "CONTROL_HELPER_RESPONSE="+successResponse(twoModelsPayload))
	client := NewClient("claude")
	client.DefaultOptions.Env = map[string]string{"FROM_DEFAULT": "1", "SHARED": "default"}
	workDir := t.TempDir()

	_, err := client.SupportedModels(context.Background(), &RunOptions{
		Env:              map[string]string{"SHARED": "run"},
		WorkingDirectory: workDir,
		// None of these change the model list, so none may reach argv.
		ResumeID:   "sess-1",
		ModelAlias: "opus",
		MCPConfigs: []string{"mcp.json"},
		Format:     JSONOutput,
	})
	if err != nil {
		t.Fatalf("SupportedModels() error = %v", err)
	}

	wantArgs := []string{
		"-p", "--input-format", "stream-json", "--output-format", "stream-json",
		"--verbose", "--no-session-persistence", "--settings", `{"disableAllHooks":true}`,
	}
	if !slices.Equal(fake.args, wantArgs) {
		t.Errorf("args = %q, want %q", fake.args, wantArgs)
	}
	if fake.cmd.Dir != workDir {
		t.Errorf("Dir = %q, want %q", fake.cmd.Dir, workDir)
	}
	for _, kv := range []string{"FROM_DEFAULT=1", "SHARED=run"} {
		if !slices.Contains(fake.cmd.Env, kv) {
			t.Errorf("Env missing %q: %v", kv, fake.cmd.Env)
		}
	}

	data, err := os.ReadFile(fake.capture)
	if err != nil {
		t.Fatalf("read captured stdin: %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("stdin had %d lines, want only the initialize request: %q", len(lines), data)
	}
	var request struct {
		Type      string          `json:"type"`
		RequestID string          `json:"request_id"`
		Request   json.RawMessage `json:"request"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &request); err != nil {
		t.Fatalf("stdin line is not JSON: %v", err)
	}
	if request.Type != "control_request" || request.RequestID == "" || string(request.Request) != `{"subtype":"initialize"}` {
		t.Errorf("request = %s", lines[0])
	}
}

func TestSupportedModelsSettingsArgs(t *testing.T) {
	tests := []struct {
		name string
		opts *RunOptions
		want []string
	}{
		{
			name: "inline settings get disableAllHooks merged in",
			opts: &RunOptions{Settings: `{"availableModels":["sonnet"]}`},
			want: []string{"--settings", `{"availableModels":["sonnet"],"disableAllHooks":true}`},
		},
		{
			name: "caller's disableAllHooks wins",
			opts: &RunOptions{Settings: `{"disableAllHooks":false}`},
			want: []string{"--settings", `{"disableAllHooks":false}`},
		},
		{
			name: "settings file passes through",
			opts: &RunOptions{Settings: "/etc/claude/settings.json"},
			want: []string{"--settings", "/etc/claude/settings.json"},
		},
		{
			name: "setting sources and bare",
			opts: &RunOptions{SettingSources: []string{"user", "project"}, Bare: true},
			want: []string{"--settings", `{"disableAllHooks":true}`, "--setting-sources", "user,project", "--bare"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := useControlFake(t, "CONTROL_HELPER_RESPONSE="+successResponse(twoModelsPayload))
			if _, err := NewClient("claude").SupportedModels(context.Background(), tc.opts); err != nil {
				t.Fatalf("SupportedModels() error = %v", err)
			}
			got := fake.args[slices.Index(fake.args, "--settings"):]
			if !slices.Equal(got, tc.want) {
				t.Errorf("args tail = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSupportedModelsRejectsInvalidOptions(t *testing.T) {
	tests := []struct {
		name  string
		opts  *RunOptions
		field string
	}{
		{name: "negative timeout", opts: &RunOptions{Timeout: -time.Second}, field: "Timeout"},
		{name: "unknown setting source", opts: &RunOptions{SettingSources: []string{"global"}}, field: "SettingSources"},
		{name: "malformed inline settings", opts: &RunOptions{Settings: `{"availableModels":`}, field: "Settings"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := useControlFake(t)
			_, err := NewClient("claude").SupportedModels(context.Background(), tc.opts)
			assertValidationField(t, err, tc.field)
			if fake.calls != 0 {
				t.Fatalf("CLI started %d times, want 0", fake.calls)
			}
		})
	}
}

func TestSupportedModelsToleratesUnknownFields(t *testing.T) {
	payload := `{"commands":[{"name":"x"}],"account":{"subscriptionType":"max"},"futureKey":{"a":[1,2]},` +
		`"models":[{"value":"opus","displayName":"Opus","description":"d","supportsThinkingOff":true,"futureField":{"nested":[1]}}]}`
	useControlFake(t, "CONTROL_HELPER_RESPONSE="+
		`{"type":"control_response","pending_permission_requests":[],"response":{"subtype":"success","request_id":"{{ID}}","extra":1,"response":`+payload+`}}`+"\n")

	models, err := NewClient("claude").SupportedModels(context.Background(), nil)
	if err != nil {
		t.Fatalf("SupportedModels() error = %v", err)
	}
	if len(models) != 1 || models[0].Value != "opus" || models[0].DisplayName != "Opus" {
		t.Fatalf("models = %+v", models)
	}
}

func TestSupportedModelsSkipsNoiseBeforeResponse(t *testing.T) {
	preamble := strings.Join([]string{
		"not json at all",
		`{"type":"system","subtype":"hook_started","hook_event":"SessionStart"}`,
		"",
		`{broken`,
		`{"type":"control_request","request_id":"cli_1","request":{"subtype":"can_use_tool"}}`,
		`{"type":"control_response","response":{"subtype":"success","request_id":"req_someone_else","response":{"models":[{"value":"wrong","displayName":"Wrong","description":""}]}}}`,
		`{"type":"control_response","response":"not an object"}`,
	}, "\n") + "\n"
	useControlFake(t,
		"CONTROL_HELPER_PREAMBLE="+preamble,
		"CONTROL_HELPER_RESPONSE="+successResponse(twoModelsPayload),
	)

	models, err := NewClient("claude").SupportedModels(context.Background(), nil)
	if err != nil {
		t.Fatalf("SupportedModels() error = %v", err)
	}
	if len(models) != 3 || models[0].Value != "default" {
		t.Fatalf("models = %+v, want the response matching our request ID", models)
	}
}

func TestSupportedModelsHandlesLongResponseLine(t *testing.T) {
	long := strings.Repeat("x", 200*1024)
	payload := `{"commands":[{"name":"big","description":"` + long + `"}],"models":[{"value":"sonnet","displayName":"Sonnet","description":"d"}]}`
	// Linux caps one environment string at 128 KiB, so the response goes
	// through a file.
	responseFile := filepath.Join(t.TempDir(), "response.jsonl")
	if err := os.WriteFile(responseFile, []byte(successResponse(payload)), 0600); err != nil {
		t.Fatal(err)
	}
	useControlFake(t, "CONTROL_HELPER_RESPONSE_FILE="+responseFile)

	models, err := NewClient("claude").SupportedModels(context.Background(), nil)
	if err != nil {
		t.Fatalf("SupportedModels() error = %v", err)
	}
	if len(models) != 1 || models[0].Value != "sonnet" {
		t.Fatalf("models = %+v", models)
	}
}

func TestSupportedModelsEmptyList(t *testing.T) {
	useControlFake(t, "CONTROL_HELPER_RESPONSE="+successResponse(`{"models":[]}`))

	models, err := NewClient("claude").SupportedModels(context.Background(), nil)
	if err != nil {
		t.Fatalf("SupportedModels() error = %v", err)
	}
	if models == nil || len(models) != 0 {
		t.Fatalf("models = %#v, want an empty non-nil slice", models)
	}
}

func TestSupportedModelsMissingModelsField(t *testing.T) {
	useControlFake(t, "CONTROL_HELPER_RESPONSE="+successResponse(`{"commands":[]}`))

	_, err := NewClient("claude").SupportedModels(context.Background(), nil)
	var claudeErr *ClaudeError
	if !errors.As(err, &claudeErr) || claudeErr.Type != ErrorValidation || !strings.Contains(claudeErr.Message, "no models") {
		t.Fatalf("error = %v, want validation error about missing models", err)
	}
}

func TestSupportedModelsErrorResponse(t *testing.T) {
	fake := useControlFake(t, "CONTROL_HELPER_RESPONSE="+
		`{"type":"control_response","response":{"subtype":"error","request_id":"{{ID}}","error":"Already initialized"}}`+"\n")

	_, err := NewClient("claude").SupportedModels(context.Background(), nil)
	var claudeErr *ClaudeError
	if !errors.As(err, &claudeErr) || claudeErr.Type != ErrorCommand {
		t.Fatalf("error = %v, want command ClaudeError", err)
	}
	if !strings.Contains(claudeErr.Message, "Already initialized") || claudeErr.Details["control_error"] != "Already initialized" {
		t.Errorf("error = %+v, want the CLI's error text", claudeErr)
	}
	if errors.Is(err, ErrNoControlResponse) {
		t.Error("an error response must not be reported as no response")
	}
	fake.assertReaped(t)
}

func TestSupportedModelsCLIExitsWithoutResponse(t *testing.T) {
	tests := []struct {
		name     string
		stderr   string
		exitCode int
		message  string
	}{
		{name: "old CLI rejects a flag", stderr: "error: unknown option '--no-session-persistence'\n", exitCode: 1, message: "unknown option"},
		{name: "silent clean exit", exitCode: 0, message: "exited without answering"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := useControlFake(t,
				"CONTROL_HELPER_MODE=exit",
				"CONTROL_HELPER_STDERR="+tc.stderr,
				"CONTROL_HELPER_EXIT="+strconv.Itoa(tc.exitCode),
			)

			_, err := NewClient("claude").SupportedModels(context.Background(), nil)
			if !errors.Is(err, ErrNoControlResponse) {
				t.Fatalf("error = %v, want ErrNoControlResponse", err)
			}
			var claudeErr *ClaudeError
			if !errors.As(err, &claudeErr) || claudeErr.Code != tc.exitCode || !strings.Contains(claudeErr.Message, tc.message) {
				t.Fatalf("error = %+v, want code %d and message containing %q", claudeErr, tc.exitCode, tc.message)
			}
			// Contains, not equality: a coverage-instrumented helper appends a
			// GOCOVERDIR warning to stderr when it exits.
			if stderr, _ := claudeErr.Details["stderr"].(string); !strings.Contains(stderr, strings.TrimSpace(tc.stderr)) {
				t.Errorf("stderr detail = %q, want it to contain %q", stderr, tc.stderr)
			}
			fake.assertReaped(t)
		})
	}
}

func TestSupportedModelsCancelReapsProcess(t *testing.T) {
	ready := filepath.Join(t.TempDir(), "ready")
	fake := useControlFake(t, "CONTROL_HELPER_MODE=hang", "CONTROL_HELPER_READY="+ready)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, err := NewClient("claude").SupportedModels(ctx, nil)
		errCh <- err
	}()

	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("fake CLI never received the request")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("SupportedModels did not return after cancel")
	}
	fake.assertReaped(t)
	if fake.cmd.ProcessState.Success() {
		t.Error("hung CLI exited cleanly; want it killed")
	}
}

func TestSupportedModelsTimeout(t *testing.T) {
	fake := useControlFake(t, "CONTROL_HELPER_MODE=hang")

	start := time.Now()
	_, err := NewClient("claude").SupportedModels(context.Background(), &RunOptions{Timeout: 300 * time.Millisecond})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	var claudeErr *ClaudeError
	if !errors.As(err, &claudeErr) || claudeErr.Type != ErrorTimeout {
		t.Fatalf("error = %v, want timeout ClaudeError", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("took %v to time out", elapsed)
	}
	fake.assertReaped(t)
}

func TestSupportedModelsKillsCLIThatLingers(t *testing.T) {
	setControlExitGrace(t, 200*time.Millisecond)
	fake := useControlFake(t, "CONTROL_HELPER_MODE=linger", "CONTROL_HELPER_RESPONSE="+successResponse(twoModelsPayload))

	start := time.Now()
	models, err := NewClient("claude").SupportedModels(context.Background(), nil)
	if err != nil {
		t.Fatalf("SupportedModels() error = %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("models = %+v", models)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("took %v; want the lingering CLI killed after the grace period", elapsed)
	}
	fake.assertReaped(t)
}

func TestSupportedModelsMissingBinary(t *testing.T) {
	client := NewClient("claude-code-go-test-no-such-binary")

	_, err := client.SupportedModels(context.Background(), nil)
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("error = %v, want exec.ErrNotFound", err)
	}
	var claudeErr *ClaudeError
	if !errors.As(err, &claudeErr) || claudeErr.Type != ErrorCommand {
		t.Fatalf("error = %v, want command ClaudeError", err)
	}
}

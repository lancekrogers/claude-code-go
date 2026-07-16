package claude

import (
	"context"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

// envEntries returns the values of cmd.Env whose key equals key.
func envEntries(env []string, key string) []string {
	var values []string
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			values = append(values, strings.TrimPrefix(entry, prefix))
		}
	}
	return values
}

func TestApplyEnv_NilAndEmptyLeaveCmdEnvNil(t *testing.T) {
	for name, env := range map[string]map[string]string{
		"nil":   nil,
		"empty": {},
	} {
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command("claude")
			ApplyEnv(cmd, env)
			if cmd.Env != nil {
				t.Fatalf("cmd.Env = %v, want nil so the child inherits the parent environment", cmd.Env)
			}
		})
	}
}

func TestApplyEnv_InjectsAndPreservesInheritedAmbientVar(t *testing.T) {
	cmd := exec.Command("claude")
	ApplyEnv(cmd, map[string]string{"CCGO_INJECTED": "value"})

	if got := envEntries(cmd.Env, "CCGO_INJECTED"); len(got) != 1 || got[0] != "value" {
		t.Fatalf("CCGO_INJECTED entries = %v, want exactly one %q", got, "value")
	}

	// PATH is a stand-in for any ambient variable inherited from the parent
	// process; injecting an unrelated key must not drop it.
	wantPath := os.Getenv("PATH")
	if got := envEntries(cmd.Env, "PATH"); len(got) != 1 || got[0] != wantPath {
		t.Fatalf("PATH entries = %v, want exactly one inherited %q", got, wantPath)
	}
}

func TestApplyEnv_OverrideSupersedesInheritedValue(t *testing.T) {
	cmd := exec.Command("claude")
	// Simulate an inherited environment that already carries the key. This also
	// covers the mocked-exec path where a caller pre-populates cmd.Env.
	cmd.Env = []string{"CCGO_OVERRIDE=inherited", "CCGO_KEEP=keep"}

	ApplyEnv(cmd, map[string]string{"CCGO_OVERRIDE": "caller"})

	got := envEntries(cmd.Env, "CCGO_OVERRIDE")
	if len(got) != 1 {
		t.Fatalf("CCGO_OVERRIDE appears %d times %v, want exactly one entry", len(got), got)
	}
	if got[0] != "caller" {
		t.Fatalf("CCGO_OVERRIDE = %q, want caller's %q", got[0], "caller")
	}
	if keep := envEntries(cmd.Env, "CCGO_KEEP"); len(keep) != 1 || keep[0] != "keep" {
		t.Fatalf("CCGO_KEEP entries = %v, want the inherited value preserved", keep)
	}
}

func TestApplyEnv_AppendsOverridesInSortedOrder(t *testing.T) {
	cmd := exec.Command("claude")
	cmd.Env = []string{"CCGO_BASE=base"}

	ApplyEnv(cmd, map[string]string{
		"CCGO_C": "3",
		"CCGO_A": "1",
		"CCGO_B": "2",
	})

	// Base entries come first, overrides follow in sorted key order for
	// deterministic output.
	want := []string{"CCGO_BASE=base", "CCGO_A=1", "CCGO_B=2", "CCGO_C=3"}
	if !reflect.DeepEqual(cmd.Env, want) {
		t.Fatalf("cmd.Env = %v, want %v", cmd.Env, want)
	}
}

func TestMergeEnv(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]string
		override map[string]string
		want     map[string]string
	}{
		{
			name:     "both empty returns nil",
			base:     nil,
			override: nil,
			want:     nil,
		},
		{
			name:     "base only",
			base:     map[string]string{"A": "1"},
			override: nil,
			want:     map[string]string{"A": "1"},
		},
		{
			name:     "override only",
			base:     nil,
			override: map[string]string{"B": "2"},
			want:     map[string]string{"B": "2"},
		},
		{
			name:     "union with override winning",
			base:     map[string]string{"A": "1", "SHARED": "base"},
			override: map[string]string{"B": "2", "SHARED": "override"},
			want:     map[string]string{"A": "1", "B": "2", "SHARED": "override"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeEnv(tc.base, tc.override)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("mergeEnv() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMergeEnv_DoesNotMutateInputs(t *testing.T) {
	base := map[string]string{"A": "1"}
	override := map[string]string{"A": "2", "B": "3"}

	merged := mergeEnv(base, override)
	merged["A"] = "mutated"
	merged["C"] = "new"

	if base["A"] != "1" || len(base) != 1 {
		t.Fatalf("base mutated: %v", base)
	}
	if override["A"] != "2" || override["B"] != "3" || len(override) != 2 {
		t.Fatalf("override mutated: %v", override)
	}
}

func TestResolveRunOptions_MergesDefaultEnvWithPerRunEnv(t *testing.T) {
	defaultOpts := &RunOptions{
		Env: map[string]string{"FROM_DEFAULT": "default", "SHARED": "default"},
	}
	client := &ClaudeClient{BinPath: "claude", DefaultOptions: defaultOpts}

	perRun := &RunOptions{
		Env: map[string]string{"FROM_RUN": "run", "SHARED": "run"},
	}
	resolved := client.resolveRunOptions(perRun)

	want := map[string]string{
		"FROM_DEFAULT": "default",
		"FROM_RUN":     "run",
		"SHARED":       "run", // per-run wins on conflict
	}
	if !reflect.DeepEqual(resolved.Env, want) {
		t.Fatalf("resolved.Env = %v, want %v", resolved.Env, want)
	}

	// The union must not mutate the client defaults or the per-run map.
	if !reflect.DeepEqual(defaultOpts.Env, map[string]string{"FROM_DEFAULT": "default", "SHARED": "default"}) {
		t.Fatalf("DefaultOptions.Env mutated: %v", defaultOpts.Env)
	}
	if !reflect.DeepEqual(perRun.Env, map[string]string{"FROM_RUN": "run", "SHARED": "run"}) {
		t.Fatalf("per-run Env mutated: %v", perRun.Env)
	}
}

func TestResolveRunOptions_NilPerRunUsesDefaultEnv(t *testing.T) {
	defaultOpts := &RunOptions{
		Env: map[string]string{"FROM_DEFAULT": "default"},
	}
	client := &ClaudeClient{BinPath: "claude", DefaultOptions: defaultOpts}

	resolved := client.resolveRunOptions(nil)
	if !reflect.DeepEqual(resolved.Env, map[string]string{"FROM_DEFAULT": "default"}) {
		t.Fatalf("resolved.Env = %v, want the client defaults", resolved.Env)
	}
}

// envInjectingExec returns a mock execCommand that runs TestHelperProcess with
// the given helper markers as the base environment. ApplyEnv treats that base as
// the inherited environment and appends RunOptions.Env on top, so the helper
// process observes the injected variables.
func envInjectingExec(helperEnv ...string) func(context.Context, string, ...string) *exec.Cmd {
	return func(_ context.Context, name string, arg ...string) *exec.Cmd {
		cs := append([]string{"-test.run=TestHelperProcess", "--", name}, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append([]string{"GO_WANT_HELPER_PROCESS=1"}, helperEnv...)
		return cmd
	}
}

func TestRunPrompt_EnvVisibleToChild(t *testing.T) {
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()
	execCommand = envInjectingExec("GO_HELPER_PRINT_ENV=CCGO_ENV_TEST")

	client := &ClaudeClient{BinPath: "claude"}
	result, err := client.RunPrompt("hi", &RunOptions{
		Format: TextOutput,
		Env:    map[string]string{"CCGO_ENV_TEST": "injected-value"},
	})
	if err != nil {
		t.Fatalf("RunPrompt() error = %v", err)
	}
	if got := strings.TrimSpace(result.Result); got != "injected-value" {
		t.Fatalf("child saw CCGO_ENV_TEST=%q, want %q", got, "injected-value")
	}
}

func TestRunFromStdin_EnvVisibleToChild(t *testing.T) {
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()
	execCommand = envInjectingExec("GO_HELPER_PRINT_ENV=CCGO_ENV_TEST")

	client := &ClaudeClient{BinPath: "claude"}
	result, err := client.RunFromStdin(strings.NewReader("stdin"), "hi", &RunOptions{
		Format: TextOutput,
		Env:    map[string]string{"CCGO_ENV_TEST": "stdin-value"},
	})
	if err != nil {
		t.Fatalf("RunFromStdin() error = %v", err)
	}
	if got := strings.TrimSpace(result.Result); got != "stdin-value" {
		t.Fatalf("child saw CCGO_ENV_TEST=%q, want %q", got, "stdin-value")
	}
}

func TestStreamPrompt_EnvVisibleToChild(t *testing.T) {
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()
	execCommand = envInjectingExec("GO_HELPER_STREAM_ENV=CCGO_ENV_TEST")

	client := &ClaudeClient{BinPath: "claude"}
	messageCh, errCh := client.StreamPrompt(context.Background(), "hi", &RunOptions{
		Format: StreamJSONOutput,
		Env:    map[string]string{"CCGO_ENV_TEST": "stream-value"},
	})

	var result string
	for msg := range messageCh {
		if msg.Type == "result" {
			result = msg.Result
		}
	}
	if err := <-errCh; err != nil {
		t.Fatalf("StreamPrompt() error = %v", err)
	}
	if result != "stream-value" {
		t.Fatalf("child saw CCGO_ENV_TEST=%q, want %q", result, "stream-value")
	}
}

func TestRunPrompt_EmptyEnvLeavesCmdEnvUntouched(t *testing.T) {
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	factoryEnv := []string{"GO_WANT_HELPER_PROCESS=1", "GO_HELPER_OUTPUT=ok", "GO_HELPER_EXIT_CODE=0"}
	var captured *exec.Cmd
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cs := append([]string{"-test.run=TestHelperProcess", "--", name}, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append([]string(nil), factoryEnv...)
		captured = cmd
		return cmd
	}

	client := &ClaudeClient{BinPath: "claude"}
	if _, err := client.RunPrompt("hi", &RunOptions{Format: TextOutput}); err != nil {
		t.Fatalf("RunPrompt() error = %v", err)
	}

	// With no RunOptions.Env, ApplyEnv must leave cmd.Env exactly as the exec
	// factory set it: it neither rewrites the slice from os.Environ nor appends.
	if !reflect.DeepEqual(captured.Env, factoryEnv) {
		t.Fatalf("cmd.Env = %v, want it untouched %v", captured.Env, factoryEnv)
	}
}

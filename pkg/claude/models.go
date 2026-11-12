package claude

import (
	"bufio"
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"time"
)

// ModelInfo describes one model the installed Claude CLI offers, as reported
// by its initialize control response. Pass Value as RunOptions.ModelAlias or
// RunOptions.Model to select the model. Fields the CLI adds later are ignored.
type ModelInfo struct {
	// Value is the alias or model ID to pass to --model, such as "opus" or
	// "claude-opus-4-8". The CLI also lists a "default" entry for its own
	// default model; leave ModelAlias and Model empty to use that one.
	Value string `json:"value"`
	// ResolvedModel is the model ID that Value resolves to. It is empty on CLIs
	// older than 2.1.197.
	ResolvedModel string `json:"resolvedModel,omitempty"`
	// DisplayName is the human-readable name, such as "Opus 5.5".
	DisplayName string `json:"displayName"`
	// Description summarizes what the model is suited for.
	Description string `json:"description"`
	// SupportsEffort reports whether the model accepts RunOptions.Effort.
	SupportsEffort bool `json:"supportsEffort,omitempty"`
	// SupportedEffortLevels lists the effort levels the model accepts.
	SupportedEffortLevels []EffortLevel `json:"supportedEffortLevels,omitempty"`
	// SupportsAdaptiveThinking reports whether the model decides when and how
	// much to think.
	SupportsAdaptiveThinking bool `json:"supportsAdaptiveThinking,omitempty"`
	// SupportsFastMode reports whether the model supports fast mode.
	SupportsFastMode bool `json:"supportsFastMode,omitempty"`
	// SupportsAutoMode reports whether the model supports auto mode.
	SupportsAutoMode bool `json:"supportsAutoMode,omitempty"`
}

// ErrNoControlResponse reports that the Claude CLI exited without answering a
// control request. A CLI that predates the stream-json control protocol, or
// that rejects one of the flags SupportedModels passes, ends this way. The
// returned *ClaudeError still carries the exit code and stderr.
var ErrNoControlResponse = errors.New("claude CLI exited without a control response")

const (
	// defaultControlTimeout matches the Python Agent SDK's default initialize
	// timeout. It applies when RunOptions.Timeout is zero.
	defaultControlTimeout = 60 * time.Second
	maxControlLineBytes   = 16 * 1024 * 1024
	disableHooksSettings  = `{"disableAllHooks":true}`
)

// controlExitGrace bounds how long the CLI may keep running after it answers
// and its stdin is closed before it is killed.
var controlExitGrace = 5 * time.Second

// SupportedModels returns the models the installed Claude CLI can use, in the
// order the CLI lists them. The list reflects the CLI version, the account it
// is signed in to, and settings such as availableModels.
//
// It starts the CLI in stream-json mode, sends only an initialize control
// request, and closes stdin once the CLI answers. No user message is sent, so
// no model call is made, and the run passes --no-session-persistence.
// SessionStart hooks are disabled for the call; see Settings below.
//
// Only the RunOptions fields that change which models the CLI offers apply:
// Env (merged with DefaultOptions.Env as in RunPromptCtx), WorkingDirectory,
// Settings, SettingSources, Bare, and Timeout. A nil opts uses DefaultOptions.
// When Settings is empty the call passes --settings {"disableAllHooks":true}.
// When Settings is inline JSON without a disableAllHooks key, that key is
// added. A settings file path is passed through unchanged, so the hooks it
// enables run.
//
// The call waits up to Timeout, or 60 seconds when Timeout is zero, and also
// stops when ctx is done. The child process is always waited on before
// SupportedModels returns. Errors are *ClaudeError values:
//   - the binary cannot be started: wraps the exec error, so
//     errors.Is(err, exec.ErrNotFound) works for a missing binary on PATH;
//   - the CLI exits without answering: wraps ErrNoControlResponse;
//   - ctx is cancelled or the deadline passes: wraps the context error;
//   - the CLI answers with an error, or with no models field: the message
//     says which.
func (c *ClaudeClient) SupportedModels(ctx context.Context, opts *RunOptions) ([]ModelInfo, error) {
	payload, err := c.initializeControl(ctx, opts)
	if err != nil {
		return nil, err
	}

	var fields struct {
		Models json.RawMessage `json:"models"`
	}
	if err := json.Unmarshal(payload, &fields); err != nil {
		return nil, newWrappedClaudeError(ErrorValidation, "failed to parse initialize response", err)
	}
	if len(fields.Models) == 0 || string(fields.Models) == "null" {
		return nil, NewClaudeError(ErrorValidation, "initialize response has no models field; the Claude CLI may be too old to list models")
	}

	models := []ModelInfo{}
	if err := json.Unmarshal(fields.Models, &models); err != nil {
		return nil, newWrappedClaudeError(ErrorValidation, "failed to parse models in initialize response", err)
	}
	return models, nil
}

// initializeControl runs the CLI, sends one initialize control request, and
// returns the response payload.
func (c *ClaudeClient) initializeControl(ctx context.Context, opts *RunOptions) (json.RawMessage, error) {
	resolved := c.resolveRunOptions(opts)
	args, err := controlArgs(resolved)
	if err != nil {
		return nil, err
	}

	timeout := resolved.Timeout
	if timeout == 0 {
		timeout = defaultControlTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	requestID, request, err := newInitializeRequest()
	if err != nil {
		return nil, err
	}

	cmd := execCommand(runCtx, c.BinPath, args...)
	if resolved.WorkingDirectory != "" {
		cmd.Dir = resolved.WorkingDirectory
	}
	ApplyEnv(cmd, resolved.Env)
	cmd.WaitDelay = controlExitGrace
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, newWrappedClaudeError(ErrorCommand, "failed to get stdin pipe", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, newWrappedClaudeError(ErrorCommand, "failed to get stdout pipe", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, newWrappedClaudeError(ErrorCommand, "failed to start Claude CLI", err)
	}

	proc := &controlProcess{cmd: cmd, stdin: stdin, stdout: stdout, reader: startControlReader(stdout, requestID)}
	// A failed write means the CLI already exited; the reader then reaches EOF
	// and the exit status and stderr explain why.
	_, _ = stdin.Write(request)

	select {
	case out := <-proc.reader.found:
		_ = proc.shutdown(runCtx, controlExitGrace)
		return out.payload, out.err
	case <-proc.reader.done:
		select {
		case out := <-proc.reader.found:
			_ = proc.shutdown(runCtx, controlExitGrace)
			return out.payload, out.err
		default:
		}
		if proc.reader.err != nil {
			_ = proc.shutdown(runCtx, 0)
			return nil, newWrappedClaudeError(ErrorCommand, "failed to read Claude CLI output", proc.reader.err)
		}
		waitErr := proc.shutdown(runCtx, controlExitGrace)
		return nil, noControlResponseError(stderr.String(), waitErr)
	case <-runCtx.Done():
		_ = proc.shutdown(runCtx, 0)
		return nil, controlContextError(runCtx.Err())
	}
}

type controlProcess struct {
	cmd    *exec.Cmd
	stdin  io.Closer
	stdout io.Closer
	reader *controlReader
}

// shutdown closes stdin so the CLI exits, waits up to grace (or until ctx
// ends), then kills the CLI if it is still running. It always waits for the
// process, so no child outlives the call, and returns the Wait error.
func (p *controlProcess) shutdown(ctx context.Context, grace time.Duration) error {
	_ = p.stdin.Close()
	exited := make(chan error, 1)
	go func() {
		// Wait must not run until reads from the stdout pipe are done.
		<-p.reader.done
		exited <- p.cmd.Wait()
	}()

	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case err := <-exited:
		return err
	case <-timer.C:
	case <-ctx.Done():
	}

	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	// Closing the read end unblocks the reader even if a grandchild process
	// still holds the write end of the pipe.
	_ = p.stdout.Close()
	return <-exited
}

func controlArgs(opts *RunOptions) ([]string, error) {
	if opts.Timeout < 0 {
		return nil, NewValidationError("Timeout cannot be negative", "Timeout", opts.Timeout)
	}
	for _, source := range opts.SettingSources {
		if !isValidSettingSource(source) {
			return nil, NewValidationError("Invalid setting source", "SettingSources", opts.SettingSources)
		}
	}
	settings, err := controlSettings(opts.Settings)
	if err != nil {
		return nil, err
	}

	args := []string{
		"-p",
		"--input-format", string(StreamJSONInput),
		"--output-format", string(StreamJSONOutput),
		"--verbose",
		"--no-session-persistence",
		"--settings", settings,
	}
	if len(opts.SettingSources) > 0 {
		args = append(args, "--setting-sources", strings.Join(opts.SettingSources, ","))
	}
	if opts.Bare {
		args = append(args, "--bare")
	}
	return args, nil
}

// controlSettings returns the --settings value for a control-only run. The CLI
// honors only the last --settings flag, so disableAllHooks is merged into the
// caller's inline JSON rather than passed separately.
func controlSettings(settings string) (string, error) {
	trimmed := strings.TrimSpace(settings)
	if trimmed == "" {
		return disableHooksSettings, nil
	}
	if !strings.HasPrefix(trimmed, "{") {
		return settings, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &fields); err != nil {
		return "", NewValidationError("Settings is not a valid JSON object", "Settings", len(settings))
	}
	if _, set := fields["disableAllHooks"]; set {
		return settings, nil
	}
	fields["disableAllHooks"] = json.RawMessage("true")
	merged, err := json.Marshal(fields)
	if err != nil {
		return "", newWrappedClaudeError(ErrorValidation, "failed to encode settings", err)
	}
	return string(merged), nil
}

func newInitializeRequest() (string, []byte, error) {
	var nonce [8]byte
	if _, err := cryptorand.Read(nonce[:]); err != nil {
		return "", nil, newWrappedClaudeError(ErrorCommand, "failed to generate control request ID", err)
	}
	requestID := "req_1_" + hex.EncodeToString(nonce[:])

	request, err := json.Marshal(map[string]any{
		"type":       "control_request",
		"request_id": requestID,
		"request":    map[string]string{"subtype": "initialize"},
	})
	if err != nil {
		return "", nil, newWrappedClaudeError(ErrorCommand, "failed to encode control request", err)
	}
	return requestID, append(request, '\n'), nil
}

type controlOutcome struct {
	payload json.RawMessage
	err     error
}

// controlReader reads the CLI's stdout until EOF. It delivers the response
// matching requestID on found and keeps draining afterwards so the CLI never
// blocks on a full pipe. err is set before done closes when reading fails.
type controlReader struct {
	found chan controlOutcome
	done  chan struct{}
	err   error
}

func startControlReader(stdout io.Reader, requestID string) *controlReader {
	r := &controlReader{
		found: make(chan controlOutcome, 1),
		done:  make(chan struct{}),
	}
	go func() {
		defer close(r.done)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), maxControlLineBytes)
		sent := false
		for scanner.Scan() {
			if sent {
				continue
			}
			if out, ok := parseControlResponse(scanner.Bytes(), requestID); ok {
				r.found <- out
				sent = true
			}
		}
		if err := scanner.Err(); err != nil && !sent {
			r.err = err
		}
	}()
	return r
}

// parseControlResponse reports whether line is the control_response for
// requestID. Other message types, other request IDs, and lines that are not
// JSON are skipped.
func parseControlResponse(line []byte, requestID string) (controlOutcome, bool) {
	var envelope struct {
		Type     string `json:"type"`
		Response struct {
			Subtype   string          `json:"subtype"`
			RequestID string          `json:"request_id"`
			Response  json.RawMessage `json:"response"`
			Error     string          `json:"error"`
		} `json:"response"`
	}
	if err := json.Unmarshal(line, &envelope); err != nil {
		return controlOutcome{}, false
	}
	if envelope.Type != "control_response" || envelope.Response.RequestID != requestID {
		return controlOutcome{}, false
	}

	if envelope.Response.Subtype == "error" {
		message := envelope.Response.Error
		if message == "" {
			message = "unknown error"
		}
		claudeErr := NewClaudeError(ErrorCommand, "Claude CLI rejected the initialize control request: "+message)
		claudeErr.Details["control_error"] = envelope.Response.Error
		return controlOutcome{err: claudeErr}, true
	}
	return controlOutcome{payload: envelope.Response.Response}, true
}

func noControlResponseError(stderr string, waitErr error) *ClaudeError {
	exitCode := 0
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		exitCode = exitErr.ExitCode()
	}

	// ParseError's Type and Details are kept as a hint, but its keyword match
	// can mislabel flag errors (an unknown --no-session-persistence reads as a
	// session error), so the message always carries the CLI's own words.
	claudeErr := ParseError(stderr, exitCode)
	claudeErr.Message = "Claude CLI exited without answering the initialize control request"
	if line, _, _ := strings.Cut(strings.TrimSpace(stderr), "\n"); line != "" {
		claudeErr.Message += ": " + strings.TrimSpace(line)
	}
	claudeErr.Original = errors.Join(ErrNoControlResponse, waitErr)
	return claudeErr
}

func controlContextError(err error) *ClaudeError {
	if errors.Is(err, context.DeadlineExceeded) {
		return newWrappedClaudeError(ErrorTimeout, "timed out waiting for the Claude CLI to answer the initialize control request", err)
	}
	return newWrappedClaudeError(ErrorCommand, "canceled while waiting for the Claude CLI to answer the initialize control request", err)
}

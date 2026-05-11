package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"strings"
)

func buildStreamJSONRunOptions(opts *RunOptions) (*RunOptions, error) {
	streamOpts := cloneRunOptions(opts)
	streamOpts.Format = StreamJSONOutput

	// Claude CLI requires --verbose when using --output-format=stream-json with --print.
	streamOpts.Verbose = true

	if err := PreprocessOptions(streamOpts); err != nil {
		return nil, err
	}

	return streamOpts, nil
}

func (c *ClaudeClient) runPromptWithStructuredHooks(ctx context.Context, prompt string, stdin io.Reader, opts *RunOptions) (*ClaudeResult, error) {
	streamOpts, err := buildStreamJSONRunOptions(opts)
	if err != nil {
		return nil, err
	}

	if streamOpts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, streamOpts.Timeout)
		defer cancel()
	}

	var result *ClaudeResult
	var assistantText strings.Builder
	err = c.executeStreamJSON(ctx, prompt, stdin, streamOpts, func(msg Message) error {
		if text, ok := msg.AssistantText(); ok {
			assistantText.WriteString(text)
		}
		if msg.Type == "result" {
			normalized, err := normalizeResult(messageToResult(msg), assistantText.String())
			if err != nil {
				return err
			}
			msg.Result = normalized.Result
			result = normalized
		}

		if err := applyMessageHooks(ctx, streamOpts, msg); err != nil {
			return err
		}

		if msg.Type != "result" {
			return nil
		}

		return applyCompletionHooks(ctx, streamOpts, result)
	})
	if err != nil {
		if result != nil {
			return result, err
		}
		return nil, err
	}

	if result == nil {
		return nil, NewClaudeError(ErrorValidation, "no result message found in JSON response")
	}

	return result, nil
}

func (c *ClaudeClient) executeStreamJSON(ctx context.Context, prompt string, stdin io.Reader, opts *RunOptions, onMessage func(Message) error) error {
	if err := ensurePluginManagerInitialized(ctx, opts.PluginManager); err != nil {
		return err
	}

	args := BuildArgs(prompt, opts)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := execCommand(runCtx, c.BinPath, args...)
	if opts.WorkingDirectory != "" {
		cmd.Dir = opts.WorkingDirectory
	}
	if stdin != nil {
		cmd.Stdin = stdin
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return newWrappedClaudeError(ErrorCommand, "failed to get stdout pipe", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return newWrappedClaudeError(ErrorCommand, "failed to get stderr pipe", err)
	}

	stderrBuf := new(bytes.Buffer)
	if err := cmd.Start(); err != nil {
		return newWrappedClaudeError(ErrorCommand, "failed to start command", err)
	}

	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		_, _ = io.Copy(stderrBuf, stderr)
	}()

	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			cancel()
			_ = cmd.Wait()
			<-stderrDone
			return newWrappedClaudeError(ErrorValidation, "failed to parse JSON message", err)
		}

		if onMessage == nil {
			continue
		}

		if err := onMessage(msg); err != nil {
			cancel()
			_ = cmd.Wait()
			<-stderrDone
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		cancel()
		_ = cmd.Wait()
		<-stderrDone
		return newWrappedClaudeError(ErrorCommand, "failed to scan stream output", err)
	}

	if err := cmd.Wait(); err != nil {
		<-stderrDone
		var exitCode int
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}

		claudeErr := ParseError(stderrBuf.String(), exitCode)
		claudeErr.Original = err
		return claudeErr
	}

	<-stderrDone

	return nil
}

func newWrappedClaudeError(errorType ErrorType, message string, err error) *ClaudeError {
	claudeErr := NewClaudeError(errorType, message)
	claudeErr.Original = err
	return claudeErr
}

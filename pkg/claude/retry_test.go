package claude

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// These tests replace the package's execCommand seam and must not run in parallel.
func mockRetrySequence(t *testing.T, responses ...string) *int {
	t.Helper()
	original := execCommand
	t.Cleanup(func() { execCommand = original })
	calls := 0
	execCommand = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		index := calls
		calls++
		if index >= len(responses) {
			t.Fatalf("unexpected CLI invocation %d", calls)
		}
		stderr := responses[index]
		exitCode := 0
		if stderr != "" {
			exitCode = 1
		}
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHelperProcess$", "--")
		cmd.Env = []string{
			"GO_WANT_HELPER_PROCESS=1", "GO_HELPER_OUTPUT=success",
			"GO_HELPER_STDERR=" + stderr, "GO_HELPER_EXIT_CODE=" + strconv.Itoa(exitCode),
			// Avoid the race runtime's exit sleep in these short-lived CLI helpers.
			"GORACE=atexit_sleep_ms=0",
		}
		return cmd
	}
	return &calls
}

func TestRunPromptWithRetryCtxExhaustion(t *testing.T) {
	for _, tc := range []struct {
		name     string
		retries  int
		stderr   string
		deadline time.Duration
	}{
		{"zero_retries_with_hint", 0, "rate limit; retry-after: 10", time.Second},
		{"zero_retries_without_hint", 0, "rate limit", time.Second},
		{"last_attempt_with_hint", 1, "rate limit; retry-after: 1", 1750 * time.Millisecond},
	} {
		t.Run(tc.name, func(t *testing.T) {
			responses := make([]string, tc.retries+1)
			for i := range responses {
				responses[i] = tc.stderr
			}
			calls := mockRetrySequence(t, responses...)
			ctx, cancel := context.WithTimeout(context.Background(), tc.deadline)
			defer cancel()
			result, err := NewClient("claude").RunPromptWithRetryCtx(ctx, "test", nil, &RetryPolicy{MaxRetries: tc.retries})
			var ce *ClaudeError
			if result != nil || !errors.As(err, &ce) || ce.Type != ErrorRateLimit {
				t.Fatalf("expected wrapped rate-limit exhaustion, got result=%v err=%v", result, err)
			}
			if !strings.Contains(err.Error(), "max retries (") || ce.Original == nil {
				t.Fatalf("exhaustion lost its message or process error: %v", err)
			}
			if *calls != tc.retries+1 {
				t.Fatalf("CLI invocations = %d, want %d", *calls, tc.retries+1)
			}
		})
	}
}

func TestRunPromptWithRetryCtxSingleDelay(t *testing.T) {
	// A single wait is 2s; additive backoff would take at least 2.8s.
	calls := mockRetrySequence(t, "rate limit; retry-after: 2", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2700*time.Millisecond)
	defer cancel()
	result, err := NewClient("claude").RunPromptWithRetryCtx(ctx, "test", nil, &RetryPolicy{
		MaxRetries: 1, BaseDelay: time.Second, MaxDelay: time.Second, BackoffFactor: 2,
	})
	if err != nil || result == nil || result.Result != "success" || *calls != 2 {
		t.Fatalf("expected successful retry after one delay; result=%v err=%v calls=%d", result, err, *calls)
	}
}

func TestRunPromptWithRetryCtxClearsRateLimitDelay(t *testing.T) {
	calls := mockRetrySequence(t, "rate limit; retry-after: 1", "network connection failed", "")
	ctx, cancel := context.WithTimeout(context.Background(), 1800*time.Millisecond)
	defer cancel()
	result, err := NewClient("claude").RunPromptWithRetryCtx(ctx, "test", nil, &RetryPolicy{
		MaxRetries: 2, BaseDelay: 100 * time.Millisecond, MaxDelay: 100 * time.Millisecond, BackoffFactor: 1,
	})
	if err != nil || result == nil || *calls != 3 {
		t.Fatalf("rate-limit delay persisted into network retry: result=%v err=%v calls=%d", result, err, *calls)
	}
}

func TestRunPromptWithRetryCtxCancellation(t *testing.T) {
	for _, tc := range []struct {
		stderr string
		base   time.Duration
	}{
		{"rate limit; retry-after: 10", time.Millisecond},
		{"network connection failed", 10 * time.Second},
	} {
		t.Run(tc.stderr, func(t *testing.T) {
			calls := mockRetrySequence(t, tc.stderr)
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			_, err := NewClient("claude").RunPromptWithRetryCtx(ctx, "test", nil, &RetryPolicy{
				MaxRetries: 3, BaseDelay: tc.base, MaxDelay: 10 * time.Second,
			})
			if !errors.Is(err, context.DeadlineExceeded) || *calls != 1 {
				t.Fatalf("expected canceled wait without retry, got err=%v calls=%d", err, *calls)
			}
		})
	}
	t.Run("already_canceled", func(t *testing.T) {
		calls := mockRetrySequence(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := NewClient("claude").RunPromptWithRetryCtx(ctx, "test", nil, nil)
		if !errors.Is(err, context.Canceled) || *calls != 0 {
			t.Fatalf("expected no CLI launch, got err=%v calls=%d", err, *calls)
		}
	})
}

func TestRunPromptWithRetryCtxNonretryable(t *testing.T) {
	for _, tc := range []struct {
		name      string
		responses []string
		opts      *RunOptions
		wantCalls int
		wantType  ErrorType
	}{
		{"authentication", []string{"authentication failed"}, nil, 1, ErrorAuthentication},
		{"validation", nil, &RunOptions{AllowedTools: []string{"mcp__badtool"}}, 0, ErrorValidation},
		{"invalid_json", []string{""}, &RunOptions{Format: JSONOutput}, 1, ErrorValidation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := mockRetrySequence(t, tc.responses...)
			_, err := NewClient("claude").RunPromptWithRetryCtx(context.Background(), "test", tc.opts, nil)
			if err == nil || strings.Contains(err.Error(), "max retries (") || *calls != tc.wantCalls {
				t.Fatalf("expected original error without retries, got err=%v calls=%d", err, *calls)
			}
			ce, ok := err.(*ClaudeError)
			if !ok || ce.Type != tc.wantType {
				t.Fatalf("expected error type %v, got %v", tc.wantType, err)
			}
		})
	}
	t.Run("non_claude_error", func(t *testing.T) {
		calls := mockRetrySequence(t)
		wantErr := errors.New("plugin initialization failed")
		plugin := newMockPlugin("retry-test", "test")
		plugin.initErr = wantErr
		pm := NewPluginManager()
		if err := pm.Register(plugin, nil); err != nil {
			t.Fatal(err)
		}
		_, err := NewClient("claude").RunPromptWithRetryCtx(context.Background(), "test", &RunOptions{PluginManager: pm}, nil)
		if !errors.Is(err, wantErr) || strings.Contains(err.Error(), "max retries (") || *calls != 0 {
			t.Fatalf("expected original plugin error without CLI launch, got err=%v calls=%d", err, *calls)
		}
	})
}

func TestRetryConvenienceMethodsSuccess(t *testing.T) {
	for _, method := range []string{"retry", "enhanced", "enhanced_ctx"} {
		t.Run(method, func(t *testing.T) {
			calls := mockRetrySequence(t, "")
			client := NewClient("claude")
			var result *ClaudeResult
			var err error
			switch method {
			case "retry":
				result, err = client.RunPromptWithRetry("test", nil, nil)
			case "enhanced":
				result, err = client.RunPromptEnhanced("test", nil)
			case "enhanced_ctx":
				result, err = client.RunPromptEnhancedCtx(context.Background(), "test", nil)
			}
			if err != nil || result == nil || result.Result != "success" || *calls != 1 {
				t.Fatalf("result=%v err=%v calls=%d", result, err, *calls)
			}
		})
	}
}

func TestRetryPolicyRetryDelay(t *testing.T) {
	for _, tc := range []struct {
		name     string
		base     time.Duration
		lastErr  *ClaudeError
		min, max time.Duration
	}{
		{"hint_exceeds_backoff", 100 * time.Millisecond, &ClaudeError{Type: ErrorRateLimit, Details: map[string]interface{}{"retry_after": 1}}, time.Second, time.Second},
		{"backoff_exceeds_hint", 2 * time.Second, &ClaudeError{Type: ErrorRateLimit, Details: map[string]interface{}{"retry_after": 1}}, 1600 * time.Millisecond, 2400 * time.Millisecond},
		{"default_hint_exceeds_max_delay", time.Second, &ClaudeError{Type: ErrorRateLimit}, 60 * time.Second, 60 * time.Second},
		{"network_uses_backoff", 100 * time.Millisecond, &ClaudeError{Type: ErrorNetwork}, 80 * time.Millisecond, 120 * time.Millisecond},
		{"mcp_uses_backoff", 100 * time.Millisecond, &ClaudeError{Type: ErrorMCP, Message: "connection failed"}, 80 * time.Millisecond, 120 * time.Millisecond},
		{"no_error_uses_backoff", 100 * time.Millisecond, nil, 80 * time.Millisecond, 120 * time.Millisecond},
		{"nonpositive_hint_uses_backoff", 100 * time.Millisecond, &ClaudeError{Type: ErrorRateLimit, Details: map[string]interface{}{"retry_after": -1}}, 80 * time.Millisecond, 120 * time.Millisecond},
	} {
		t.Run(tc.name, func(t *testing.T) {
			policy := &RetryPolicy{BaseDelay: tc.base, MaxDelay: tc.base, BackoffFactor: 2}
			delay := policy.retryDelay(1, tc.lastErr)
			if delay < tc.min || delay > tc.max {
				t.Fatalf("retry delay = %v, want [%v, %v]", delay, tc.min, tc.max)
			}
		})
	}
}

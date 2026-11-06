package claude

import (
	"context"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
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
		name    string
		retries int
		stderr  string
	}{
		{"zero_retries_with_hint", 0, "rate limit; retry-after: 10"},
		{"zero_retries_without_hint", 0, "rate limit"},
		{"last_attempt_with_hint", 1, "rate limit; retry-after: 1"},
		{"all_default_delays", 3, "rate limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			responses := make([]string, tc.retries+1)
			for i := range responses {
				responses[i] = tc.stderr
			}
			calls := mockRetrySequence(t, responses...)
			waits := 0
			result, err := NewClient("claude").runPromptWithRetryCtx(context.Background(), "test", nil, &RetryPolicy{MaxRetries: tc.retries}, func(ctx context.Context, delay time.Duration) error {
				waits++
				if delay <= 0 {
					t.Fatalf("rate-limit retry must have a positive delay, got %v", delay)
				}
				return ctx.Err()
			})
			var ce *ClaudeError
			if result != nil || !errors.As(err, &ce) || ce.Type != ErrorRateLimit {
				t.Fatalf("expected wrapped rate-limit exhaustion, got result=%v err=%v", result, err)
			}
			if !strings.Contains(err.Error(), "max retries (") || ce.Original == nil {
				t.Fatalf("exhaustion lost its message or process error: %v", err)
			}
			if *calls != tc.retries+1 || waits != tc.retries {
				t.Fatalf("CLI invocations = %d, waits = %d; want %d calls and %d waits", *calls, waits, tc.retries+1, tc.retries)
			}
		})
	}
}

func TestRunPromptWithRetryCtxSingleDelay(t *testing.T) {
	calls := mockRetrySequence(t, "rate limit; retry-after: 2", "")
	waits := 0
	result, err := NewClient("claude").runPromptWithRetryCtx(context.Background(), "test", nil, &RetryPolicy{
		MaxRetries: 1, BaseDelay: time.Second, MaxDelay: time.Second, BackoffFactor: 2,
	}, func(ctx context.Context, delay time.Duration) error {
		waits++
		// The server floor plus positive jitter is <=2.4s. Adding the
		// separate jittered backoff would take at least 2.8s.
		if delay < 2*time.Second || delay > 2400*time.Millisecond {
			t.Fatalf("expected server minimum plus jitter, got %v", delay)
		}
		return ctx.Err()
	})
	if err != nil || result == nil || result.Result != "success" || *calls != 2 || waits != 1 {
		t.Fatalf("expected successful retry after one delay; result=%v err=%v calls=%d waits=%d", result, err, *calls, waits)
	}
}

func TestRunPromptWithRetryCtxClearsRateLimitDelay(t *testing.T) {
	calls := mockRetrySequence(t, "rate limit; retry-after: 1", "network connection failed", "")
	var delays []time.Duration
	result, err := NewClient("claude").runPromptWithRetryCtx(context.Background(), "test", nil, &RetryPolicy{
		MaxRetries: 2, BaseDelay: 100 * time.Millisecond, MaxDelay: 100 * time.Millisecond, BackoffFactor: 1,
	}, func(ctx context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return ctx.Err()
	})
	if err != nil || result == nil || *calls != 3 || len(delays) != 2 {
		t.Fatalf("result=%v err=%v calls=%d delays=%v", result, err, *calls, delays)
	}
	if delays[0] < time.Second || delays[0] > 1200*time.Millisecond || delays[1] < 80*time.Millisecond || delays[1] > 120*time.Millisecond {
		t.Fatalf("rate-limit delay persisted into network retry: %v", delays)
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			waits := 0
			_, err := NewClient("claude").runPromptWithRetryCtx(ctx, "test", nil, &RetryPolicy{
				MaxRetries: 3, BaseDelay: tc.base, MaxDelay: 10 * time.Second,
			}, func(ctx context.Context, delay time.Duration) error {
				waits++
				cancel()
				return waitForRetry(ctx, delay)
			})
			if !errors.Is(err, context.Canceled) || *calls != 1 || waits != 1 {
				t.Fatalf("expected canceled wait without retry, got err=%v calls=%d waits=%d", err, *calls, waits)
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

func TestRunPromptWithRetryCtxCancellationDuringFinalAttempt(t *testing.T) {
	for _, mode := range []string{"cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			// First fail normally, then cancel the final allowed subprocess after
			// it has emitted rate-limit stderr. Both failures must not mask ctx.Err().
			calls := mockRetrySequence(t, "network connection failed", "rate limit; retry-after: 1")
			command := execCommand
			readyFile := filepath.Join(t.TempDir(), "stderr-written")
			execCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
				cmd := command(ctx, name, args...)
				if *calls == 2 {
					cmd.Env = append(cmd.Env, "GO_HELPER_WAIT_AFTER_STDERR="+readyFile)
				}
				return cmd
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			ready := make(chan bool, 1)
			go func() {
				ticker := time.NewTicker(5 * time.Millisecond)
				defer ticker.Stop()
				for {
					if _, err := os.Stat(readyFile); err == nil {
						ready <- true
						if mode == "cancel" {
							cancel()
						}
						return
					}
					select {
					case <-ticker.C:
					case <-ctx.Done():
						ready <- false
						return
					}
				}
			}()
			_, err := NewClient("claude").RunPromptWithRetryCtx(ctx, "test", nil, &RetryPolicy{MaxRetries: 1})
			if !<-ready {
				t.Fatal("final subprocess did not emit stderr before the test deadline")
			}
			want := context.Canceled
			if mode == "deadline" {
				want = context.DeadlineExceeded
			}
			if !errors.Is(err, want) || *calls != 2 {
				t.Fatalf("expected final-attempt context error %v, got %v (calls=%d)", want, err, *calls)
			}
		})
	}
}

func TestRateLimitRetryDelayRetainsJitter(t *testing.T) {
	policy := DefaultRetryPolicy()
	seen := make(map[time.Duration]bool)
	for i := 0; i < 8; i++ {
		delay := policy.retryDelay(1, &ClaudeError{Type: ErrorRateLimit})
		if delay < 60*time.Second || delay > 72*time.Second {
			t.Fatalf("rate-limit delay %v is outside the server floor plus 20%% jitter", delay)
		}
		seen[delay] = true
	}
	if len(seen) < 2 {
		t.Fatal("server-dominated delays lost their jitter")
	}
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
		{"hint_exceeds_backoff", 100 * time.Millisecond, &ClaudeError{Type: ErrorRateLimit, Details: map[string]interface{}{"retry_after": 1}}, time.Second, 1200 * time.Millisecond},
		{"backoff_exceeds_hint", 2 * time.Second, &ClaudeError{Type: ErrorRateLimit, Details: map[string]interface{}{"retry_after": 1}}, 1600 * time.Millisecond, 2400 * time.Millisecond},
		{"default_hint_exceeds_max_delay", time.Second, &ClaudeError{Type: ErrorRateLimit}, 60 * time.Second, 72 * time.Second},
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

func TestRetryPolicyRetryDelaySaturates(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("overflow-sized retry-after requires 64-bit integers")
	}
	maxSeconds := int64(math.MaxInt64) / int64(time.Second)
	for _, seconds := range []int64{maxSeconds, maxSeconds + 1, math.MaxInt64} {
		policy := DefaultRetryPolicy()
		delay := policy.retryDelay(1, &ClaudeError{Type: ErrorRateLimit, Details: map[string]interface{}{"retry_after": int(seconds)}})
		floor := time.Duration(math.MaxInt64)
		if seconds <= maxSeconds {
			floor = time.Duration(seconds) * time.Second
		}
		if delay < floor {
			t.Fatalf("retry-after %d overflowed to %v, below %v", seconds, delay, floor)
		}
	}
}

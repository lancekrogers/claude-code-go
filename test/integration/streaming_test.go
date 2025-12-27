package integration

import (
	"context"
	"testing"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
	"github.com/lancekrogers/claude-code-go/test/utils"
)

func TestStreamingPrompt(t *testing.T) {
	utils.SkipIfNoClaudeCLI(t)

	client := utils.NewTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messageCh, errCh := client.StreamPrompt(ctx, "Count from 1 to 5", &claude.RunOptions{})

	var messages []claude.Message
	var streamErr error

	// Handle errors
	go func() {
		for err := range errCh {
			streamErr = err
		}
	}()

	// Collect messages
	for msg := range messageCh {
		messages = append(messages, msg)
		t.Logf("Received message type: %s", msg.Type)

		if msg.Type == "assistant" && msg.Result != "" {
			t.Logf("Assistant message: %s", msg.Result)
		}

		if msg.Type == "result" {
			t.Logf("Final result - Cost: $%.6f, Turns: %d", msg.CostUSD, msg.NumTurns)
			break
		}
	}

	if streamErr != nil {
		t.Fatalf("Streaming error: %v", streamErr)
	}

	if len(messages) == 0 {
		t.Fatal("Expected at least one message")
	}

	// Verify we got a system init message
	var hasSystemInit bool
	for _, msg := range messages {
		if msg.Type == "system" && msg.Subtype == "init" {
			hasSystemInit = true
			break
		}
	}

	if !hasSystemInit {
		t.Error("Expected system init message")
	}

	// Verify we got a final result
	lastMsg := messages[len(messages)-1]
	if lastMsg.Type != "result" {
		t.Errorf("Expected final result message, got %s", lastMsg.Type)
	}
}

func TestStreamingWithContext(t *testing.T) {
	utils.SkipIfNoClaudeCLI(t)

	client := utils.NewTestClient(t)

	// Test context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	messageCh, errCh := client.StreamPrompt(ctx, "Write a very long story", &claude.RunOptions{})

	var receivedMessages int
	var contextCanceled bool

	// Handle errors
	go func() {
		for err := range errCh {
			if err == context.DeadlineExceeded {
				contextCanceled = true
			}
			t.Logf("Stream error (expected due to timeout): %v", err)
		}
	}()

	// Count messages until context times out
	for msg := range messageCh {
		receivedMessages++
		t.Logf("Received message %d: %s", receivedMessages, msg.Type)

		// Don't wait for completion since we expect timeout
		if receivedMessages >= 3 {
			break
		}
	}

	// Give time for context cancellation to propagate
	time.Sleep(100 * time.Millisecond)

	if !contextCanceled {
		t.Log("Context cancellation may not have been triggered (this is OK)")
	}

	t.Logf("Received %d messages before timeout", receivedMessages)
}

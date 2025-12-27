package claude

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"time"
)

// ValidateSessionID checks if a session ID is a valid UUID format
// This is stricter than isValidSessionID and should be used for new SessionID field
func ValidateSessionID(id string) error {
	if id == "" {
		return nil // Optional field
	}

	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	// 8-4-4-4-12 = 36 characters including hyphens
	if len(id) != 36 {
		return fmt.Errorf("invalid session ID: must be a valid UUID (36 characters)")
	}

	// Check format positions
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return fmt.Errorf("invalid session ID: must be a valid UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)")
	}

	// Check all characters are valid hex or hyphen
	for i, c := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue // Skip hyphen positions
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("invalid session ID: contains non-hexadecimal character at position %d", i)
		}
	}

	return nil
}

// GenerateSessionID creates a new random session UUID (version 4)
func GenerateSessionID() string {
	// Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx (version 4 UUID)
	uuid := make([]byte, 16)

	// Use crypto/rand for secure random generation
	_, err := cryptorand.Read(uuid)
	if err != nil {
		// Fallback to time-based if crypto/rand fails (shouldn't happen)
		for i := range uuid {
			uuid[i] = byte(time.Now().UnixNano() >> (i * 4))
		}
	}

	// Set version (4) and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:16])
}

// RunWithSession executes with a specific session ID
func (c *ClaudeClient) RunWithSession(prompt, sessionID string, opts *RunOptions) (*ClaudeResult, error) {
	return c.RunWithSessionCtx(context.Background(), prompt, sessionID, opts)
}

// RunWithSessionCtx executes with a specific session ID with context support
func (c *ClaudeClient) RunWithSessionCtx(ctx context.Context, prompt, sessionID string, opts *RunOptions) (*ClaudeResult, error) {
	if opts == nil {
		opts = &RunOptions{}
	}
	runOpts := *opts
	runOpts.SessionID = sessionID
	if runOpts.Format == "" {
		runOpts.Format = JSONOutput
	}
	return c.RunPromptCtx(ctx, prompt, &runOpts)
}

// RunEphemeral executes without persisting the session
func (c *ClaudeClient) RunEphemeral(prompt string, opts *RunOptions) (*ClaudeResult, error) {
	return c.RunEphemeralCtx(context.Background(), prompt, opts)
}

// RunEphemeralCtx executes without persisting the session with context support
func (c *ClaudeClient) RunEphemeralCtx(ctx context.Context, prompt string, opts *RunOptions) (*ClaudeResult, error) {
	if opts == nil {
		opts = &RunOptions{}
	}
	runOpts := *opts
	runOpts.NoSessionPersistence = true
	runOpts.PrintMode = true // Required for no-session-persistence
	if runOpts.Format == "" {
		runOpts.Format = JSONOutput
	}
	return c.RunPromptCtx(ctx, prompt, &runOpts)
}

// ForkAndRun continues a session but creates a new fork
func (c *ClaudeClient) ForkAndRun(prompt, originalSessionID string, opts *RunOptions) (*ClaudeResult, error) {
	return c.ForkAndRunCtx(context.Background(), prompt, originalSessionID, opts)
}

// ForkAndRunCtx continues a session but creates a new fork with context support
func (c *ClaudeClient) ForkAndRunCtx(ctx context.Context, prompt, originalSessionID string, opts *RunOptions) (*ClaudeResult, error) {
	if opts == nil {
		opts = &RunOptions{}
	}
	runOpts := *opts
	runOpts.ResumeID = originalSessionID
	runOpts.ForkSession = true
	if runOpts.Format == "" {
		runOpts.Format = JSONOutput
	}
	return c.RunPromptCtx(ctx, prompt, &runOpts)
}

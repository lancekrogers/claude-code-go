package claude

import (
	cryptorand "crypto/rand"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// OutputFormat defines the output format for Claude Code responses
type OutputFormat string

const (
	// TextOutput returns plain text responses
	TextOutput OutputFormat = "text"
	// JSONOutput returns structured JSON responses
	JSONOutput OutputFormat = "json"
	// StreamJSONOutput streams JSON responses as they arrive
	StreamJSONOutput OutputFormat = "stream-json"
)

// RunOptions configures how Claude Code is executed
type RunOptions struct {
	// Format specifies the output format (text, json, stream-json)
	Format OutputFormat
	// SystemPrompt overrides the default system prompt
	SystemPrompt string
	// AppendPrompt appends to the default system prompt
	AppendPrompt string
	// MCPConfigPath is the path to the MCP configuration file
	MCPConfigPath string
	// AllowedTools is a list of tools that Claude is allowed to use
	// Supports both legacy format ("Bash") and enhanced format ("Bash(git log:*)")
	AllowedTools []string
	// DisallowedTools is a list of tools that Claude is not allowed to use
	// Supports both legacy format ("Bash") and enhanced format ("Bash(git log:*)")
	DisallowedTools []string
	// PermissionTool is the MCP tool for handling permission prompts
	PermissionTool string
	// ResumeID is the session ID to resume
	ResumeID string
	// Continue indicates whether to continue the most recent conversation
	Continue bool
	// MaxTurns limits the number of agentic turns in non-interactive mode
	MaxTurns int
	// Verbose enables verbose logging
	Verbose bool
	// Model specifies the model to use (full model name)
	Model string

	// Enhanced options for 100% CLI support
	// ModelAlias specifies model using alias ("sonnet", "opus", "haiku")
	ModelAlias string
	// Timeout specifies the maximum duration for command execution
	Timeout time.Duration
	// ConfigFile specifies path to Claude configuration file
	ConfigFile string
	// Help shows help information
	Help bool
	// Version shows version information
	Version bool
	// DisableAutoUpdate disables automatic updates
	DisableAutoUpdate bool
	// Theme specifies the UI theme
	Theme string
	// IncludePartialMessages enables streaming of partial message chunks as they arrive
	// Only works with --print and --output-format=stream-json
	IncludePartialMessages bool

	// PermissionMode controls default permission handling
	// "default" - standard checks, "acceptEdits" - auto-approve edits, "bypassPermissions" - skip all
	PermissionMode PermissionMode
	// PermissionCallback is called before each tool use to determine permission
	// If nil, default behavior based on PermissionMode is used
	PermissionCallback PermissionCallback `json:"-"`

	// MaxBudgetUSD sets the maximum spending limit in USD
	// Execution stops if this limit is exceeded
	MaxBudgetUSD float64
	// BudgetTracker tracks cumulative spending across sessions
	// If nil, a new tracker is created for each execution
	BudgetTracker *BudgetTracker `json:"-"`

	// Agents defines specialized sub-agents that can be invoked by the main agent
	// Each agent has its own description, prompt, allowed tools, and model
	// The main agent uses descriptions to decide which subagent to invoke
	Agents map[string]*SubagentConfig `json:"-"`

	// PluginManager manages plugins that hook into the execution lifecycle
	// Plugins can intercept tool calls, messages, and completion events
	PluginManager *PluginManager `json:"-"`

	// Parsed tool permissions (computed from AllowedTools/DisallowedTools)
	// This field is populated automatically and should not be set directly
	ParsedAllowedTools    []ToolPermission `json:"-"`
	ParsedDisallowedTools []ToolPermission `json:"-"`

	// Session control options
	// SessionID specifies a UUID for the conversation session
	// Must be a valid UUID format
	SessionID string
	// ForkSession creates a new session ID when resuming
	// Use with Continue or ResumeID options
	ForkSession bool
	// NoSessionPersistence disables session saving to disk
	// Sessions cannot be resumed when this is enabled
	NoSessionPersistence bool

	// MCP configuration options (enhanced)
	// MCPConfigs specifies multiple MCP server configurations
	// Can be file paths or JSON strings
	MCPConfigs []string
	// StrictMCPConfig only uses servers from MCPConfigs
	// Ignores all other MCP configurations
	StrictMCPConfig bool

	// Additional CLI flags
	// AddDirectories specifies additional directories to include in context
	AddDirectories []string
	// WorkingDirectory sets the process working directory for Claude CLI execution
	WorkingDirectory string
	// PrintMode enables print mode output (required for some flags)
	PrintMode bool

	// JSONSchema specifies a JSON Schema for structured output validation
	// The response will be validated against this schema
	// Example: {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
	JSONSchema string

	// FallbackModel specifies a model to use when the primary model is overloaded
	// Only works with --print mode
	// Example: "sonnet" or "claude-sonnet-4-5-20250929"
	FallbackModel string

	// Debug enables debug mode with optional category filtering
	// Example: "api,mcp" or "!statsig,!file" (exclude categories)
	Debug string
}

// validateMCPToolName validates that MCP tool names follow the correct pattern: mcp__<serverName>__<toolName>
func validateMCPToolName(tool string) bool {
	return strings.HasPrefix(tool, "mcp__") && strings.Count(tool, "__") >= 2
}

// validateMCPTools validates all MCP tools in the slice
func validateMCPTools(tools []string) error {
	for _, tool := range tools {
		if strings.HasPrefix(tool, "mcp__") && !validateMCPToolName(tool) {
			return fmt.Errorf("invalid MCP tool name: %s (must follow pattern: mcp__<serverName>__<toolName>)", tool)
		}
	}
	return nil
}

// PreprocessOptions validates and preprocesses RunOptions before execution
func PreprocessOptions(opts *RunOptions) error {
	if opts == nil {
		return nil
	}

	// Validate and parse allowed tools
	if len(opts.AllowedTools) > 0 {
		parsed, err := ParseToolPermissions(opts.AllowedTools)
		if err != nil {
			return NewValidationError("Invalid allowed tool permissions", "AllowedTools", opts.AllowedTools)
		}
		opts.ParsedAllowedTools = parsed

		// Validate MCP tools in allowed tools
		if err := validateMCPTools(opts.AllowedTools); err != nil {
			return NewValidationError(err.Error(), "AllowedTools", opts.AllowedTools)
		}
	}

	// Validate and parse disallowed tools
	if len(opts.DisallowedTools) > 0 {
		parsed, err := ParseToolPermissions(opts.DisallowedTools)
		if err != nil {
			return NewValidationError("Invalid disallowed tool permissions", "DisallowedTools", opts.DisallowedTools)
		}
		opts.ParsedDisallowedTools = parsed

		// Validate MCP tools in disallowed tools
		if err := validateMCPTools(opts.DisallowedTools); err != nil {
			return NewValidationError(err.Error(), "DisallowedTools", opts.DisallowedTools)
		}
	}

	// Validate model alias
	if opts.ModelAlias != "" {
		if !isValidModelAlias(opts.ModelAlias) {
			return NewValidationError("Invalid model alias", "ModelAlias", opts.ModelAlias)
		}
	}

	// Validate timeout
	if opts.Timeout < 0 {
		return NewValidationError("Timeout cannot be negative", "Timeout", opts.Timeout)
	}

	// Validate session ID format if provided
	if opts.ResumeID != "" {
		if !isValidSessionID(opts.ResumeID) {
			return NewValidationError("Invalid session ID format", "ResumeID", opts.ResumeID)
		}
	}

	// Validate SessionID (stricter UUID validation for new field)
	if opts.SessionID != "" {
		if err := ValidateSessionID(opts.SessionID); err != nil {
			return NewValidationError(err.Error(), "SessionID", opts.SessionID)
		}
	}

	// Validate subagent configurations
	if len(opts.Agents) > 0 {
		for name, config := range opts.Agents {
			if config == nil {
				return NewValidationError("Subagent config cannot be nil", "Agents", name)
			}
			if err := config.Validate(); err != nil {
				return NewValidationError(fmt.Sprintf("Invalid subagent '%s': %v", name, err), "Agents", name)
			}
		}
	}

	return nil
}

// isValidModelAlias checks if the model alias is supported
func isValidModelAlias(alias string) bool {
	validAliases := []string{"sonnet", "opus", "haiku"}
	for _, valid := range validAliases {
		if alias == valid {
			return true
		}
	}
	return false
}

// isValidSessionID validates session ID format (should be UUID-like)
func isValidSessionID(sessionID string) bool {
	// Be more lenient with session ID validation to avoid breaking existing usage
	// Just check for basic non-empty string for now
	return strings.TrimSpace(sessionID) != ""
}

// ValidateOptions validates RunOptions without executing a command
func ValidateOptions(opts *RunOptions) error {
	return PreprocessOptions(opts)
}

// RetryPolicy defines the retry behavior for failed requests
type RetryPolicy struct {
	MaxRetries    int           // Maximum number of retry attempts
	BaseDelay     time.Duration // Base delay between retries
	MaxDelay      time.Duration // Maximum delay between retries
	BackoffFactor float64       // Exponential backoff factor
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
	}
}

// calculateBackoff calculates the delay for a given retry attempt
func (rp *RetryPolicy) calculateBackoff(attempt int) time.Duration {
	if attempt == 0 {
		return 0
	}

	delay := float64(rp.BaseDelay) * math.Pow(rp.BackoffFactor, float64(attempt-1))

	result := time.Duration(delay)
	if result > rp.MaxDelay {
		result = rp.MaxDelay
	}

	return result
}

// cryptoRandFloat64 returns a cryptographically secure random float64 in [0, 1)
func cryptoRandFloat64() float64 {
	var b [8]byte
	_, err := cryptorand.Read(b[:])
	if err != nil {
		// Fallback to math/rand if crypto/rand fails (shouldn't happen)
		return rand.Float64()
	}
	// Convert bytes to uint64 and normalize to [0, 1)
	n := uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
	return float64(n) / float64(math.MaxUint64)
}

// calculateBackoffWithJitter adds ±20% randomization to prevent thundering herd
// Uses crypto/rand for secure randomization
func (rp *RetryPolicy) calculateBackoffWithJitter(attempt int) time.Duration {
	delay := rp.calculateBackoff(attempt)
	if delay == 0 {
		return 0
	}

	// Add jitter: ±20% randomization using crypto/rand
	// Formula: delay + (delay * 0.2 * (random value from -1 to 1))
	jitter := float64(delay) * 0.2 * (2*cryptoRandFloat64() - 1)
	result := delay + time.Duration(jitter)

	// Ensure we don't go negative (shouldn't happen with 20% jitter, but be safe)
	if result < 0 {
		result = 0
	}

	return result
}

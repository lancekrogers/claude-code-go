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

// InputFormat defines the input format for Claude Code requests.
type InputFormat string

const (
	// TextInput sends plain text on stdin.
	TextInput InputFormat = "text"
	// StreamJSONInput sends streaming JSON events on stdin.
	StreamJSONInput InputFormat = "stream-json"
)

// EffortLevel defines the Claude reasoning effort for the current session.
type EffortLevel string

const (
	EffortLow    EffortLevel = "low"
	EffortMedium EffortLevel = "medium"
	EffortHigh   EffortLevel = "high"
	EffortXHigh  EffortLevel = "xhigh"
	EffortMax    EffortLevel = "max"
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
	// Deprecated: Claude Code no longer exposes --permission-prompt-tool on the
	// current CLI surface. Setting this field returns a validation error.
	PermissionTool string
	// ResumeID is the session ID to resume
	ResumeID string
	// Continue indicates whether to continue the most recent conversation
	Continue bool
	// Deprecated: Claude Code no longer exposes --max-turns on the current CLI
	// surface. Setting this field returns a validation error.
	MaxTurns int
	// Verbose enables verbose logging
	Verbose bool
	// Model specifies the model to use (full model name)
	Model string
	// Agent selects the active named agent for the current request
	Agent string

	// Enhanced options for 100% CLI support
	// ModelAlias specifies model using alias ("sonnet", "opus", "haiku")
	ModelAlias string
	// Effort specifies the reasoning effort level for the current session
	Effort EffortLevel
	// Timeout specifies the maximum duration for command execution
	Timeout time.Duration
	// Deprecated: Claude Code no longer exposes --config on the current CLI
	// surface. Setting this field returns a validation error.
	ConfigFile string
	// Help shows help information
	Help bool
	// Version shows version information
	Version bool
	// Deprecated: Claude Code no longer exposes --disable-autoupdate on the
	// current CLI surface. Setting this field returns a validation error.
	DisableAutoUpdate bool
	// Deprecated: Claude Code no longer exposes --theme on the current CLI
	// surface. Setting this field returns a validation error.
	Theme string
	// InputFormat specifies stdin input mode for print runs
	InputFormat InputFormat
	// IncludeHookEvents includes hook lifecycle events in stream-json output
	IncludeHookEvents bool
	// IncludePartialMessages enables streaming of partial message chunks as they arrive
	// Only works with --print and --output-format=stream-json
	IncludePartialMessages bool
	// ReplayUserMessages re-emits user messages on stdout when using stream-json IO
	ReplayUserMessages bool
	// DebugFile writes debug logs to a specific file path
	DebugFile string
	// Bare enables Claude's minimal mode
	Bare bool
	// Brief enables the SendUserMessage tool for agent-to-user communication
	Brief bool
	// Betas includes beta headers in API requests
	Betas []string
	// Files downloads file resources at startup in file_id:path form
	Files []string
	// ExcludeDynamicSystemPromptSections moves machine-specific system sections
	// into the first user message for better cache reuse
	ExcludeDynamicSystemPromptSections bool
	// AllowDangerouslySkipPermissions enables the bypass option without enabling
	// it by default for the session
	AllowDangerouslySkipPermissions bool

	// PermissionMode controls default permission handling
	// Supported values: "default", "acceptEdits", "auto", "bypassPermissions", "dontAsk", and "plan"
	PermissionMode PermissionMode
	// PermissionCallback is called before each tool use to determine permission
	// Deprecated: the current Claude CLI no longer exposes a wrapper-safe
	// permission callback injection point. Setting this field returns a
	// validation error.
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
	// AgentsJSON provides a raw JSON string for --agents. If set, it takes
	// precedence over Agents.
	AgentsJSON string

	// PluginManager manages plugins that hook into the execution lifecycle
	// Plugins can intercept tool calls, messages, and completion events
	PluginManager *PluginManager `json:"-"`

	// Parsed tool permissions (computed on internal processed copies of
	// RunOptions used during execution). Callers should not set these directly.
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
	// SettingSources controls which setting sources Claude loads
	SettingSources []string
	// Settings specifies a settings file path or inline JSON string
	Settings string
	// Tools limits the available built-in tool set
	Tools []string
	// Name sets a display name for the current session
	Name string
	// PluginDirs loads plugins from one or more directories
	PluginDirs []string
	// WorkingDirectory sets the process working directory (cmd.Dir) for the
	// Claude CLI subprocess. If empty, the subprocess inherits the parent
	// process's current directory (backward compatible). Must be an
	// absolute path that exists at the time Run/Stream is called; otherwise
	// exec will fail to start the subprocess. Set per call — no shared
	// state across invocations.
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

	if err := validateDeprecatedOptions(opts); err != nil {
		return err
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

	// Validate effort level
	if opts.Effort != "" && !isValidEffortLevel(opts.Effort) {
		return NewValidationError("Invalid effort level", "Effort", opts.Effort)
	}

	// Validate input format
	if opts.InputFormat != "" && !isValidInputFormat(opts.InputFormat) {
		return NewValidationError("Invalid input format", "InputFormat", opts.InputFormat)
	}

	// Validate permission mode
	if opts.PermissionMode != "" {
		if opts.PermissionMode == PermissionModeDelegate {
			return NewValidationError("PermissionModeDelegate is deprecated and no longer supported by the Claude CLI", "PermissionMode", opts.PermissionMode)
		}
		if !isValidPermissionMode(opts.PermissionMode) {
			return NewValidationError("Invalid permission mode", "PermissionMode", opts.PermissionMode)
		}
	}

	// Validate settings sources
	if len(opts.SettingSources) > 0 {
		for _, source := range opts.SettingSources {
			if !isValidSettingSource(source) {
				return NewValidationError("Invalid setting source", "SettingSources", opts.SettingSources)
			}
		}
	}

	// Validate timeout
	if opts.Timeout < 0 {
		return NewValidationError("Timeout cannot be negative", "Timeout", opts.Timeout)
	}

	// Validate budget limit
	if opts.MaxBudgetUSD < 0 {
		return NewValidationError("MaxBudgetUSD cannot be negative", "MaxBudgetUSD", opts.MaxBudgetUSD)
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
		if opts.AgentsJSON != "" {
			return NewValidationError("Agents and AgentsJSON are mutually exclusive", "AgentsJSON", opts.AgentsJSON)
		}
		for name, config := range opts.Agents {
			if config == nil {
				return NewValidationError("Subagent config cannot be nil", "Agents", name)
			}
			if err := config.Validate(); err != nil {
				return NewValidationError(fmt.Sprintf("Invalid subagent '%s': %v", name, err), "Agents", name)
			}
		}
	}

	// Validate stream-specific combinations
	if opts.IncludeHookEvents && opts.Format != StreamJSONOutput {
		return NewValidationError("IncludeHookEvents requires stream-json output", "IncludeHookEvents", opts.IncludeHookEvents)
	}
	if opts.IncludePartialMessages && opts.Format != StreamJSONOutput {
		return NewValidationError("IncludePartialMessages requires stream-json output", "IncludePartialMessages", opts.IncludePartialMessages)
	}
	if opts.ReplayUserMessages {
		if opts.Format != StreamJSONOutput || opts.InputFormat != StreamJSONInput {
			return NewValidationError("ReplayUserMessages requires stream-json input and output", "ReplayUserMessages", opts.ReplayUserMessages)
		}
	}

	return nil
}

func validateDeprecatedOptions(opts *RunOptions) error {
	switch {
	case opts.PermissionTool != "":
		return NewValidationError("PermissionTool is deprecated and no longer supported by the Claude CLI", "PermissionTool", opts.PermissionTool)
	case opts.MaxTurns != 0:
		return NewValidationError("MaxTurns is deprecated and no longer supported by the Claude CLI", "MaxTurns", opts.MaxTurns)
	case opts.ConfigFile != "":
		return NewValidationError("ConfigFile is deprecated and no longer supported by the Claude CLI", "ConfigFile", opts.ConfigFile)
	case opts.DisableAutoUpdate:
		return NewValidationError("DisableAutoUpdate is deprecated and no longer supported by the Claude CLI", "DisableAutoUpdate", opts.DisableAutoUpdate)
	case opts.Theme != "":
		return NewValidationError("Theme is deprecated and no longer supported by the Claude CLI", "Theme", opts.Theme)
	case opts.PermissionCallback != nil:
		return NewValidationError("PermissionCallback is deprecated and no longer supported by the Claude CLI", "PermissionCallback", "set")
	default:
		return nil
	}
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

func isValidEffortLevel(level EffortLevel) bool {
	validLevels := []EffortLevel{EffortLow, EffortMedium, EffortHigh, EffortXHigh, EffortMax}
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}

func isValidInputFormat(format InputFormat) bool {
	return format == TextInput || format == StreamJSONInput
}

func isValidPermissionMode(mode PermissionMode) bool {
	switch mode {
	case PermissionModeDefault,
		PermissionModeAcceptEdits,
		PermissionModeAuto,
		PermissionModeBypassPermissions,
		PermissionModeDontAsk,
		PermissionModePlan:
		return true
	default:
		return false
	}
}

func isValidSettingSource(source string) bool {
	switch source {
	case "user", "project", "local":
		return true
	default:
		return false
	}
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

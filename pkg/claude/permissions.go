package claude

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// PermissionBehavior defines how to handle a tool permission request
type PermissionBehavior string

const (
	// PermissionAllow allows the tool to execute
	PermissionAllow PermissionBehavior = "allow"
	// PermissionDeny blocks the tool from executing
	PermissionDeny PermissionBehavior = "deny"
	// PermissionAsk prompts the user for confirmation
	PermissionAsk PermissionBehavior = "ask"
)

// PermissionResult is returned by the permission callback
type PermissionResult struct {
	// Behavior specifies whether to allow, deny, or ask for the tool
	Behavior PermissionBehavior `json:"behavior"`
	// Message is an optional message explaining the decision (used for deny/ask)
	Message string `json:"message,omitempty"`
}

// ToolInput represents the input parameters for a tool call
// Fields are populated based on the tool type
type ToolInput struct {
	// Command is the command to execute (for Bash tool)
	Command string `json:"command,omitempty"`
	// FilePath is the file path (for Read/Write/Edit tools)
	FilePath string `json:"file_path,omitempty"`
	// Pattern is the search pattern (for Glob/Grep tools)
	Pattern string `json:"pattern,omitempty"`
	// Content is the content to write (for Write tool)
	Content string `json:"content,omitempty"`
	// OldString is the text to replace (for Edit tool)
	OldString string `json:"old_string,omitempty"`
	// NewString is the replacement text (for Edit tool)
	NewString string `json:"new_string,omitempty"`
	// Raw contains the full input as a map for custom processing
	Raw map[string]interface{} `json:"raw,omitempty"`
}

// PermissionCallback is called when Claude wants to use a tool
// It receives the tool name and input, and returns a decision
type PermissionCallback func(ctx context.Context, toolName string, input ToolInput) (PermissionResult, error)

// PermissionMode controls default permission handling
type PermissionMode string

const (
	// PermissionModeDefault uses standard permission checks
	PermissionModeDefault PermissionMode = "default"
	// PermissionModeAcceptEdits auto-approves file edit operations
	PermissionModeAcceptEdits PermissionMode = "acceptEdits"
	// PermissionModeBypassPermissions skips all permission checks (use with caution)
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// Allow returns a PermissionResult that allows the tool
func Allow() PermissionResult {
	return PermissionResult{Behavior: PermissionAllow}
}

// Deny returns a PermissionResult that denies the tool with an optional message
func Deny(message string) PermissionResult {
	return PermissionResult{Behavior: PermissionDeny, Message: message}
}

// Ask returns a PermissionResult that prompts for confirmation
func Ask(message string) PermissionResult {
	return PermissionResult{Behavior: PermissionAsk, Message: message}
}

// ReadOnlyCallback returns a permission callback that allows only read-only tools
func ReadOnlyCallback() PermissionCallback {
	return func(ctx context.Context, toolName string, input ToolInput) (PermissionResult, error) {
		readOnlyTools := map[string]bool{
			"Read": true,
			"Grep": true,
			"Glob": true,
		}
		if readOnlyTools[toolName] {
			return Allow(), nil
		}
		return Deny("Only read-only operations are allowed"), nil
	}
}

// SafeBashCallback returns a permission callback that blocks dangerous bash commands
func SafeBashCallback(blockedPatterns []string) PermissionCallback {
	if len(blockedPatterns) == 0 {
		// Default blocked patterns
		blockedPatterns = []string{
			"rm -rf",
			"rm -r",
			"> /dev/",
			"dd if=",
			"mkfs",
			":(){:|:&};:",
			"chmod -R 777",
			"curl | sh",
			"wget | sh",
		}
	}
	return func(ctx context.Context, toolName string, input ToolInput) (PermissionResult, error) {
		if toolName != "Bash" {
			return Allow(), nil
		}
		for _, pattern := range blockedPatterns {
			if strings.Contains(input.Command, pattern) {
				return Deny(fmt.Sprintf("Blocked dangerous command pattern: %s", pattern)), nil
			}
		}
		return Allow(), nil
	}
}

// FilePathCallback returns a permission callback that restricts file operations to allowed paths
func FilePathCallback(allowedPaths []string, deniedPaths []string) PermissionCallback {
	return func(ctx context.Context, toolName string, input ToolInput) (PermissionResult, error) {
		fileTools := map[string]bool{
			"Read":  true,
			"Write": true,
			"Edit":  true,
		}
		if !fileTools[toolName] {
			return Allow(), nil
		}

		filePath := input.FilePath
		if filePath == "" {
			return Allow(), nil
		}

		// Check denied paths first
		for _, denied := range deniedPaths {
			if strings.HasPrefix(filePath, denied) {
				return Deny(fmt.Sprintf("Access to path %s is denied", denied)), nil
			}
		}

		// If allowed paths are specified, check them
		if len(allowedPaths) > 0 {
			allowed := false
			for _, path := range allowedPaths {
				if strings.HasPrefix(filePath, path) {
					allowed = true
					break
				}
			}
			if !allowed {
				return Deny(fmt.Sprintf("File path %s is not in allowed paths", filePath)), nil
			}
		}

		return Allow(), nil
	}
}

// ChainCallbacks chains multiple permission callbacks together
// All callbacks must allow for the tool to be allowed
// The first deny or ask result is returned
func ChainCallbacks(callbacks ...PermissionCallback) PermissionCallback {
	return func(ctx context.Context, toolName string, input ToolInput) (PermissionResult, error) {
		for _, cb := range callbacks {
			if cb == nil {
				continue
			}
			result, err := cb(ctx, toolName, input)
			if err != nil {
				return PermissionResult{}, err
			}
			if result.Behavior != PermissionAllow {
				return result, nil
			}
		}
		return Allow(), nil
	}
}

// ToolPermission represents a parsed tool permission with optional command and pattern constraints
type ToolPermission struct {
	Tool     string // e.g., "Bash", "Write", "mcp__filesystem__read_file"
	Command  string // e.g., "git log", "npm install" (optional)
	Pattern  string // e.g., "*", "src/**" (optional)
	Original string // Original permission string as provided
}

// ParseToolPermission parses tool permission strings supporting both legacy and enhanced formats
//
// Supported formats:
//   - Legacy: "Bash", "Write", "mcp__filesystem__read_file"
//   - Enhanced: "Bash(git log)", "Bash(git log:*)", "Write(src/**)"
//   - Complex: "Bash(npm install:package.json)", "Write(/src/**:/test/**)"
func ParseToolPermission(permission string) (*ToolPermission, error) {
	if permission == "" {
		return nil, fmt.Errorf("empty permission string")
	}

	// Handle legacy format: "Bash", "Write", etc.
	if !strings.Contains(permission, "(") {
		return &ToolPermission{
			Tool:     strings.TrimSpace(permission),
			Original: permission,
		}, nil
	}

	// Parse enhanced format: "Tool(command:pattern)" or "Tool(command)"
	// Regex explanation:
	// ^([^(]+) - Capture tool name (everything before first '(')
	// \( - Literal opening parenthesis
	// ([^:)]+) - Capture command (everything until ':' or ')')
	// (?::([^:)]+))? - Optional group: ':' followed by pattern (no more colons allowed)
	// \)$ - Literal closing parenthesis at end
	re := regexp.MustCompile(`^([^(]+)\(([^:)]+)(?::([^:)]+))?\)$`)
	matches := re.FindStringSubmatch(permission)

	if len(matches) < 3 {
		return nil, fmt.Errorf("invalid tool permission format: %s (expected format: Tool(command) or Tool(command:pattern))", permission)
	}

	tool := strings.TrimSpace(matches[1])
	command := strings.TrimSpace(matches[2])
	pattern := ""
	if len(matches) > 3 && matches[3] != "" {
		pattern = strings.TrimSpace(matches[3])
	}

	// Validate tool name is not empty
	if tool == "" {
		return nil, fmt.Errorf("tool name cannot be empty in permission: %s", permission)
	}

	// Validate command is not empty when specified
	if command == "" {
		return nil, fmt.Errorf("command cannot be empty in permission: %s", permission)
	}

	return &ToolPermission{
		Tool:     tool,
		Command:  command,
		Pattern:  pattern,
		Original: permission,
	}, nil
}

// ParseToolPermissions parses a slice of tool permission strings
func ParseToolPermissions(permissions []string) ([]ToolPermission, error) {
	var parsed []ToolPermission
	for i, perm := range permissions {
		parsedPerm, err := ParseToolPermission(perm)
		if err != nil {
			return nil, fmt.Errorf("error parsing permission at index %d: %w", i, err)
		}
		parsed = append(parsed, *parsedPerm)
	}
	return parsed, nil
}

// ValidateToolPermissions validates that all tool permissions are correctly formatted
func ValidateToolPermissions(permissions []string) error {
	_, err := ParseToolPermissions(permissions)
	return err
}

// String returns the original permission string representation
func (tp *ToolPermission) String() string {
	return tp.Original
}

// IsLegacyFormat returns true if this permission uses the legacy format (tool name only)
func (tp *ToolPermission) IsLegacyFormat() bool {
	return tp.Command == "" && tp.Pattern == ""
}

// HasCommand returns true if this permission specifies a command constraint
func (tp *ToolPermission) HasCommand() bool {
	return tp.Command != ""
}

// HasPattern returns true if this permission specifies a pattern constraint
func (tp *ToolPermission) HasPattern() bool {
	return tp.Pattern != ""
}

// ToLegacyFormat converts the permission to legacy format (tool name only)
// This is useful for backward compatibility with older CLI versions
func (tp *ToolPermission) ToLegacyFormat() string {
	return tp.Tool
}

// MatchesTool returns true if the given tool name matches this permission's tool
func (tp *ToolPermission) MatchesTool(tool string) bool {
	return tp.Tool == tool
}

// MatchesCommand returns true if the given command matches this permission's command constraint
// If no command constraint is specified, returns true (allows all commands)
func (tp *ToolPermission) MatchesCommand(command string) bool {
	if !tp.HasCommand() {
		return true // No command constraint means all commands allowed
	}
	return tp.Command == command
}

// MatchesPattern returns true if the given path/pattern matches this permission's pattern constraint
// If no pattern constraint is specified, returns true (allows all patterns)
func (tp *ToolPermission) MatchesPattern(path string) bool {
	if !tp.HasPattern() {
		return true // No pattern constraint means all patterns allowed
	}

	// Simple glob-like matching for now
	// TODO: Implement full glob pattern matching if needed
	if tp.Pattern == "*" {
		return true
	}

	// Check for exact match first
	if tp.Pattern == path {
		return true
	}

	// Check for prefix match with double wildcard
	if strings.HasSuffix(tp.Pattern, "**") {
		prefix := strings.TrimSuffix(tp.Pattern, "**")
		return strings.HasPrefix(path, prefix)
	}

	// Check for prefix match with single wildcard
	if strings.HasSuffix(tp.Pattern, "*") {
		prefix := strings.TrimSuffix(tp.Pattern, "*")
		return strings.HasPrefix(path, prefix)
	}

	// Check for suffix match (e.g., "*.go" matches "main.go")
	if strings.HasPrefix(tp.Pattern, "*") {
		suffix := strings.TrimPrefix(tp.Pattern, "*")
		return strings.HasSuffix(path, suffix)
	}

	return false
}

// Matches returns true if the given tool, command, and path all match this permission
func (tp *ToolPermission) Matches(tool, command, path string) bool {
	return tp.MatchesTool(tool) && tp.MatchesCommand(command) && tp.MatchesPattern(path)
}

package claude

import (
	"fmt"
	"regexp"
	"strings"
)

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
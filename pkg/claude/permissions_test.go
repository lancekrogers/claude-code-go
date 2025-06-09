package claude

import (
	"testing"
)

func TestParseToolPermission(t *testing.T) {
	tests := []struct {
		name        string
		permission  string
		want        ToolPermission
		expectError bool
	}{
		{
			name:       "Legacy format - simple tool",
			permission: "Bash",
			want: ToolPermission{
				Tool:     "Bash",
				Command:  "",
				Pattern:  "",
				Original: "Bash",
			},
			expectError: false,
		},
		{
			name:       "Legacy format - MCP tool",
			permission: "mcp__filesystem__read_file",
			want: ToolPermission{
				Tool:     "mcp__filesystem__read_file",
				Command:  "",
				Pattern:  "",
				Original: "mcp__filesystem__read_file",
			},
			expectError: false,
		},
		{
			name:       "Enhanced format - tool with command",
			permission: "Bash(git log)",
			want: ToolPermission{
				Tool:     "Bash",
				Command:  "git log",
				Pattern:  "",
				Original: "Bash(git log)",
			},
			expectError: false,
		},
		{
			name:       "Enhanced format - tool with command and pattern",
			permission: "Bash(git log:*)",
			want: ToolPermission{
				Tool:     "Bash",
				Command:  "git log",
				Pattern:  "*",
				Original: "Bash(git log:*)",
			},
			expectError: false,
		},
		{
			name:       "Enhanced format - Write with path pattern",
			permission: "Write(src/**)",
			want: ToolPermission{
				Tool:     "Write",
				Command:  "src/**",
				Pattern:  "",
				Original: "Write(src/**)",
			},
			expectError: false,
		},
		{
			name:       "Enhanced format - complex pattern",
			permission: "Bash(npm install:package.json)",
			want: ToolPermission{
				Tool:     "Bash",
				Command:  "npm install",
				Pattern:  "package.json",
				Original: "Bash(npm install:package.json)",
			},
			expectError: false,
		},
		{
			name:        "Error - empty string",
			permission:  "",
			want:        ToolPermission{},
			expectError: true,
		},
		{
			name:        "Error - malformed parentheses",
			permission:  "Bash(git log",
			want:        ToolPermission{},
			expectError: true,
		},
		{
			name:        "Error - empty tool name",
			permission:  "(git log)",
			want:        ToolPermission{},
			expectError: true,
		},
		{
			name:        "Error - empty command",
			permission:  "Bash()",
			want:        ToolPermission{},
			expectError: true,
		},
		{
			name:        "Error - invalid format",
			permission:  "Bash(git log:pattern:extra)",
			want:        ToolPermission{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseToolPermission(tt.permission)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseToolPermission() expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("ParseToolPermission() unexpected error: %v", err)
				return
			}
			
			if got.Tool != tt.want.Tool {
				t.Errorf("ParseToolPermission() Tool = %v, want %v", got.Tool, tt.want.Tool)
			}
			if got.Command != tt.want.Command {
				t.Errorf("ParseToolPermission() Command = %v, want %v", got.Command, tt.want.Command)
			}
			if got.Pattern != tt.want.Pattern {
				t.Errorf("ParseToolPermission() Pattern = %v, want %v", got.Pattern, tt.want.Pattern)
			}
			if got.Original != tt.want.Original {
				t.Errorf("ParseToolPermission() Original = %v, want %v", got.Original, tt.want.Original)
			}
		})
	}
}

func TestParseToolPermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		expectError bool
		wantCount   int
	}{
		{
			name: "Valid mixed permissions",
			permissions: []string{
				"Bash",
				"Bash(git log)",
				"Write(src/**)",
				"mcp__filesystem__read_file",
			},
			expectError: false,
			wantCount:   4,
		},
		{
			name: "Empty slice",
			permissions: []string{},
			expectError: false,
			wantCount:   0,
		},
		{
			name: "Invalid permission in slice",
			permissions: []string{
				"Bash",
				"Invalid()",
				"Write",
			},
			expectError: true,
			wantCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseToolPermissions(tt.permissions)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseToolPermissions() expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("ParseToolPermissions() unexpected error: %v", err)
				return
			}
			
			if len(got) != tt.wantCount {
				t.Errorf("ParseToolPermissions() count = %v, want %v", len(got), tt.wantCount)
			}
		})
	}
}

func TestToolPermission_Methods(t *testing.T) {
	tests := []struct {
		name       string
		permission ToolPermission
		testCases  []struct {
			method   string
			input    []string
			expected bool
		}
	}{
		{
			name: "Legacy format permission",
			permission: ToolPermission{
				Tool:     "Bash",
				Original: "Bash",
			},
			testCases: []struct {
				method   string
				input    []string
				expected bool
			}{
				{"IsLegacyFormat", nil, true},
				{"HasCommand", nil, false},
				{"HasPattern", nil, false},
				{"MatchesTool", []string{"Bash"}, true},
				{"MatchesTool", []string{"Write"}, false},
				{"MatchesCommand", []string{"git log"}, true}, // No constraint means allow all
				{"MatchesPattern", []string{"src/file.go"}, true}, // No constraint means allow all
			},
		},
		{
			name: "Enhanced format with command and pattern",
			permission: ToolPermission{
				Tool:     "Bash",
				Command:  "git log",
				Pattern:  "src/**",
				Original: "Bash(git log:src/**)",
			},
			testCases: []struct {
				method   string
				input    []string
				expected bool
			}{
				{"IsLegacyFormat", nil, false},
				{"HasCommand", nil, true},
				{"HasPattern", nil, true},
				{"MatchesTool", []string{"Bash"}, true},
				{"MatchesTool", []string{"Write"}, false},
				{"MatchesCommand", []string{"git log"}, true},
				{"MatchesCommand", []string{"git status"}, false},
				{"MatchesPattern", []string{"src/file.go"}, true},
				{"MatchesPattern", []string{"test/file.go"}, false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, tc := range tt.testCases {
				t.Run(tc.method, func(t *testing.T) {
					var got bool
					switch tc.method {
					case "IsLegacyFormat":
						got = tt.permission.IsLegacyFormat()
					case "HasCommand":
						got = tt.permission.HasCommand()
					case "HasPattern":
						got = tt.permission.HasPattern()
					case "MatchesTool":
						got = tt.permission.MatchesTool(tc.input[0])
					case "MatchesCommand":
						got = tt.permission.MatchesCommand(tc.input[0])
					case "MatchesPattern":
						got = tt.permission.MatchesPattern(tc.input[0])
					default:
						t.Errorf("Unknown test method: %s", tc.method)
						return
					}
					
					if got != tc.expected {
						t.Errorf("%s() = %v, want %v", tc.method, got, tc.expected)
					}
				})
			}
		})
	}
}

func TestToolPermission_PatternMatching(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		testPath string
		want     bool
	}{
		{"Wildcard all", "*", "any/path", true},
		{"Directory wildcard", "src/**", "src/file.go", true},
		{"Directory wildcard miss", "src/**", "test/file.go", false},
		{"File wildcard", "*.go", "main.go", true},
		{"File wildcard miss", "*.go", "main.js", false},
		{"Exact match", "package.json", "package.json", true},
		{"Exact match miss", "package.json", "package-lock.json", false},
		{"Empty pattern (no constraint)", "", "any/path", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm := &ToolPermission{Pattern: tt.pattern}
			got := perm.MatchesPattern(tt.testPath)
			if got != tt.want {
				t.Errorf("MatchesPattern(%q) with pattern %q = %v, want %v", 
					tt.testPath, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestValidateToolPermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		expectError bool
	}{
		{
			name: "All valid permissions",
			permissions: []string{
				"Bash",
				"Bash(git log)",
				"Write(src/**)",
				"mcp__filesystem__read_file",
			},
			expectError: false,
		},
		{
			name: "Contains invalid permission",
			permissions: []string{
				"Bash",
				"Invalid()",
			},
			expectError: true,
		},
		{
			name:        "Empty slice",
			permissions: []string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolPermissions(tt.permissions)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateToolPermissions() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
// Package main demonstrates the permission system in the Claude Code Go SDK.
// This example shows how to:
// 1. Use pre-built permission callbacks (ReadOnly, SafeBash, FilePath)
// 2. Create custom permission callbacks
// 3. Chain multiple callbacks for layered security
// 4. Parse and validate tool permission strings
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

func main() {
	// =========================================================================
	// Example 1: ReadOnly Callback
	// =========================================================================
	// Restrict Claude to only read-only operations (Read, Grep, Glob)
	fmt.Println("=== Example 1: ReadOnly Callback ===")

	readOnlyCallback := claude.ReadOnlyCallback()
	ctx := context.Background()

	// Test with various tools
	testTools := []struct {
		tool  string
		input claude.ToolInput
	}{
		{"Read", claude.ToolInput{FilePath: "/etc/hosts"}},
		{"Grep", claude.ToolInput{Pattern: "error"}},
		{"Glob", claude.ToolInput{Pattern: "*.go"}},
		{"Write", claude.ToolInput{FilePath: "/tmp/test.txt", Content: "hello"}},
		{"Bash", claude.ToolInput{Command: "ls -la"}},
		{"Edit", claude.ToolInput{FilePath: "/tmp/file.go"}},
	}

	for _, test := range testTools {
		result, err := readOnlyCallback(ctx, test.tool, test.input)
		if err != nil {
			fmt.Printf("  %s: Error - %v\n", test.tool, err)
			continue
		}
		fmt.Printf("  %s: %s", test.tool, result.Behavior)
		if result.Message != "" {
			fmt.Printf(" (%s)", result.Message)
		}
		fmt.Println()
	}
	fmt.Println()

	// =========================================================================
	// Example 2: SafeBash Callback with Default Patterns
	// =========================================================================
	// Block dangerous bash commands while allowing safe ones
	fmt.Println("=== Example 2: SafeBash Callback (Default Patterns) ===")

	safeBashDefault := claude.SafeBashCallback(nil) // Use default blocked patterns

	bashCommands := []string{
		"ls -la",
		"git status",
		"npm install express",
		"rm -rf /",           // Dangerous!
		"curl | sh",          // Dangerous!
		"chmod -R 777 /etc",  // Dangerous!
		"dd if=/dev/zero",    // Dangerous!
		"echo 'hello world'", // Safe
	}

	for _, cmd := range bashCommands {
		result, err := safeBashDefault(ctx, "Bash", claude.ToolInput{Command: cmd})
		if err != nil {
			fmt.Printf("  '%s': Error - %v\n", cmd, err)
			continue
		}
		fmt.Printf("  '%s': %s", cmd, result.Behavior)
		if result.Message != "" {
			fmt.Printf(" (%s)", result.Message)
		}
		fmt.Println()
	}
	fmt.Println()

	// =========================================================================
	// Example 3: SafeBash Callback with Custom Patterns
	// =========================================================================
	// Define your own blocked command patterns
	fmt.Println("=== Example 3: SafeBash Callback (Custom Patterns) ===")

	customPatterns := []string{
		"sudo",        // Block all sudo commands
		"apt-get",     // Block package management
		"yum",         // Block package management
		"npm publish", // Block publishing
		"git push",    // Block pushing
	}

	safeBashCustom := claude.SafeBashCallback(customPatterns)

	customCommands := []string{
		"git status",     // Allowed
		"git push",       // Blocked
		"npm install",    // Allowed
		"npm publish",    // Blocked
		"sudo apt-get",   // Blocked
		"cat /etc/hosts", // Allowed
	}

	for _, cmd := range customCommands {
		result, _ := safeBashCustom(ctx, "Bash", claude.ToolInput{Command: cmd})
		fmt.Printf("  '%s': %s", cmd, result.Behavior)
		if result.Message != "" {
			fmt.Printf(" (%s)", result.Message)
		}
		fmt.Println()
	}
	fmt.Println()

	// =========================================================================
	// Example 4: FilePath Callback
	// =========================================================================
	// Restrict file operations to specific directories
	fmt.Println("=== Example 4: FilePath Callback ===")

	allowedPaths := []string{
		"/home/user/project",
		"/tmp",
	}
	deniedPaths := []string{
		"/etc",
		"/var",
		"/root",
	}

	filePathCallback := claude.FilePathCallback(allowedPaths, deniedPaths)

	filePaths := []struct {
		tool string
		path string
	}{
		{"Read", "/home/user/project/main.go"}, // Allowed (in allowed paths)
		{"Write", "/tmp/test.txt"},             // Allowed (in allowed paths)
		{"Edit", "/etc/passwd"},                // Denied (in denied paths)
		{"Read", "/var/log/syslog"},            // Denied (in denied paths)
		{"Write", "/usr/bin/test"},             // Denied (not in allowed paths)
		{"Bash", "ls"},                         // Allowed (not a file tool)
	}

	for _, fp := range filePaths {
		result, _ := filePathCallback(ctx, fp.tool, claude.ToolInput{FilePath: fp.path})
		fmt.Printf("  %s %s: %s", fp.tool, fp.path, result.Behavior)
		if result.Message != "" {
			fmt.Printf(" (%s)", result.Message)
		}
		fmt.Println()
	}
	fmt.Println()

	// =========================================================================
	// Example 5: Chaining Callbacks
	// =========================================================================
	// Combine multiple callbacks for layered security
	fmt.Println("=== Example 5: Chained Callbacks ===")

	// Create a chain of:
	// 1. File path restrictions
	// 2. Safe bash patterns
	chainedCallback := claude.ChainCallbacks(
		claude.FilePathCallback([]string{"/home/user/project"}, []string{"/etc", "/var"}),
		claude.SafeBashCallback(nil),
	)

	chainedTests := []struct {
		tool  string
		input claude.ToolInput
	}{
		{"Read", claude.ToolInput{FilePath: "/home/user/project/main.go"}}, // Allowed
		{"Write", claude.ToolInput{FilePath: "/etc/passwd"}},               // Denied by FilePath
		{"Bash", claude.ToolInput{Command: "ls -la"}},                      // Allowed
		{"Bash", claude.ToolInput{Command: "rm -rf /"}},                    // Denied by SafeBash
	}

	for _, test := range chainedTests {
		result, _ := chainedCallback(ctx, test.tool, test.input)
		display := test.input.FilePath
		if test.input.Command != "" {
			display = test.input.Command
		}
		fmt.Printf("  %s '%s': %s", test.tool, display, result.Behavior)
		if result.Message != "" {
			fmt.Printf(" (%s)", result.Message)
		}
		fmt.Println()
	}
	fmt.Println()

	// =========================================================================
	// Example 6: Custom Permission Callback
	// =========================================================================
	// Create your own permission logic
	fmt.Println("=== Example 6: Custom Permission Callback ===")

	// Custom callback that:
	// - Denies any command containing "secret" or "password"
	// - Asks for confirmation on destructive operations
	// - Allows everything else
	customCallback := func(ctx context.Context, toolName string, input claude.ToolInput) (claude.PermissionResult, error) {
		// Check for sensitive content
		sensitivePatterns := []string{"secret", "password", "api_key", "token"}
		content := strings.ToLower(input.Command + input.Content + input.FilePath)

		for _, pattern := range sensitivePatterns {
			if strings.Contains(content, pattern) {
				return claude.Deny(fmt.Sprintf("Access to '%s' content is blocked for security", pattern)), nil
			}
		}

		// Ask for confirmation on destructive operations
		if toolName == "Write" || toolName == "Edit" {
			return claude.Ask("This operation will modify files. Continue?"), nil
		}

		// Allow everything else
		return claude.Allow(), nil
	}

	customTests := []struct {
		tool  string
		input claude.ToolInput
	}{
		{"Read", claude.ToolInput{FilePath: "/home/user/config.yaml"}}, // Allowed
		{"Read", claude.ToolInput{FilePath: "/home/user/.secrets"}},    // Denied
		{"Bash", claude.ToolInput{Command: "echo $API_KEY"}},           // Denied
		{"Write", claude.ToolInput{FilePath: "/tmp/output.txt"}},       // Ask
		{"Bash", claude.ToolInput{Command: "git status"}},              // Allowed
	}

	for _, test := range customTests {
		result, _ := customCallback(ctx, test.tool, test.input)
		display := test.input.FilePath
		if test.input.Command != "" {
			display = test.input.Command
		}
		fmt.Printf("  %s '%s': %s", test.tool, display, result.Behavior)
		if result.Message != "" {
			fmt.Printf(" (%s)", result.Message)
		}
		fmt.Println()
	}
	fmt.Println()

	// =========================================================================
	// Example 7: Tool Permission Parsing
	// =========================================================================
	// Parse and validate tool permission strings
	fmt.Println("=== Example 7: Tool Permission Parsing ===")

	permissions := []string{
		"Bash",                  // Legacy format
		"Write",                 // Legacy format
		"Bash(git log:*)",       // Enhanced: any git log command
		"Bash(npm install)",     // Enhanced: npm install only
		"Write(src/**)",         // Enhanced: write to src/ directory
		"mcp__filesystem__read", // MCP tool
	}

	for _, perm := range permissions {
		parsed, err := claude.ParseToolPermission(perm)
		if err != nil {
			fmt.Printf("  '%s': Error - %v\n", perm, err)
			continue
		}
		fmt.Printf("  '%s' -> Tool=%s, Command=%s, Pattern=%s, Legacy=%v\n",
			perm, parsed.Tool, parsed.Command, parsed.Pattern, parsed.IsLegacyFormat())
	}
	fmt.Println()

	// =========================================================================
	// Example 8: Using Permission Callback with Claude Client
	// =========================================================================
	fmt.Println("=== Example 8: Using with Claude Client ===")

	cc := claude.NewClient("claude")

	// Example RunOptions with permission callback
	opts := &claude.RunOptions{
		Format: claude.JSONOutput,
		// Set permission mode
		PermissionMode: claude.PermissionModeDefault,
		// Set custom permission callback
		PermissionCallback: claude.ChainCallbacks(
			claude.ReadOnlyCallback(),
			claude.SafeBashCallback(nil),
		),
		// Or use allowed/disallowed tools for simpler cases
		AllowedTools: []string{
			"Read",
			"Grep",
			"Glob",
			"Bash(git status:*)",
			"Bash(git log:*)",
		},
		DisallowedTools: []string{
			"Write",
			"Edit",
			"Bash(rm:*)",
		},
	}

	fmt.Println("RunOptions configured with:")
	fmt.Printf("  PermissionMode: %s\n", opts.PermissionMode)
	fmt.Printf("  AllowedTools: %v\n", opts.AllowedTools)
	fmt.Printf("  DisallowedTools: %v\n", opts.DisallowedTools)
	fmt.Println("  PermissionCallback: ChainCallbacks(ReadOnlyCallback, SafeBashCallback)")
	fmt.Println()

	// Validate the tool permissions
	if err := claude.ValidateToolPermissions(opts.AllowedTools); err != nil {
		fmt.Printf("  Validation error: %v\n", err)
	} else {
		fmt.Println("  AllowedTools validated successfully")
	}

	// Note: In real usage, you would call:
	// result, err := client.RunPrompt("Your prompt", opts)
	_ = cc
	fmt.Println()

	// =========================================================================
	// Summary
	// =========================================================================
	fmt.Println("=== Permission System Summary ===")
	fmt.Println(`Permission Behaviors:
- PermissionAllow - Allow the tool to execute
- PermissionDeny  - Block the tool with a message
- PermissionAsk   - Prompt user for confirmation

Pre-built Callbacks:
- ReadOnlyCallback()           - Only allow Read, Grep, Glob
- SafeBashCallback(patterns)   - Block dangerous bash commands
- FilePathCallback(allow,deny) - Restrict file paths

Callback Chaining:
- ChainCallbacks(cb1, cb2, ...) - All must allow

Permission Modes:
- PermissionModeDefault           - Standard checks
- PermissionModeAcceptEdits       - Auto-approve edits
- PermissionModeBypassPermissions - Skip all checks (dangerous!)

Tool Permission Formats:
- Legacy:   "Bash", "Write"
- Enhanced: "Bash(git log:*)", "Write(src/**)"
- MCP:      "mcp__server__tool"

Usage in RunOptions:
- PermissionMode     - Set default behavior
- PermissionCallback - Custom callback function
- AllowedTools       - List of permitted tools
- DisallowedTools    - List of blocked tools`)
}

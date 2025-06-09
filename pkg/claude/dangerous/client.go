package dangerous

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

// DangerousClient provides access to unsafe Claude Code operations
// WARNING: This client can bypass critical security controls
// REQUIREMENT: Must set CLAUDE_ENABLE_DANGEROUS="i-accept-all-risks"
// REQUIREMENT: Must not be used in production (checked via env vars)
type DangerousClient struct {
	*claude.ClaudeClient
	securityGate *SecurityGate
	envVars      map[string]string
	mcpDebug     bool
}

// SecurityGate enforces access controls for dangerous operations
type SecurityGate struct {
	confirmed       bool
	productionCheck bool
}

// NewDangerousClient creates a client with dangerous capabilities
// Returns error unless security requirements are met
func NewDangerousClient(binPath string) (*DangerousClient, error) {
	gate := &SecurityGate{}

	// Check environment variable confirmation
	if os.Getenv("CLAUDE_ENABLE_DANGEROUS") != "i-accept-all-risks" {
		return nil, fmt.Errorf("dangerous client requires CLAUDE_ENABLE_DANGEROUS=i-accept-all-risks")
	}
	gate.confirmed = true

	// Prevent production usage
	if env := os.Getenv("NODE_ENV"); env == "production" {
		return nil, fmt.Errorf("dangerous client forbidden in production environment (NODE_ENV=%s)", env)
	}
	if env := os.Getenv("GO_ENV"); env == "production" {
		return nil, fmt.Errorf("dangerous client forbidden in production environment (GO_ENV=%s)", env)
	}
	if env := os.Getenv("ENVIRONMENT"); env == "production" || env == "prod" {
		return nil, fmt.Errorf("dangerous client forbidden in production environment (ENVIRONMENT=%s)", env)
	}
	gate.productionCheck = true

	return &DangerousClient{
		ClaudeClient: claude.NewClient(binPath),
		securityGate: gate,
		envVars:      make(map[string]string),
		mcpDebug:     false,
	}, nil
}

// BYPASS_ALL_PERMISSIONS disables Claude's permission system
// WARNING: This completely removes safety guardrails
// WARNING: Can allow arbitrary file system access and command execution
// WARNING: Should never be used with untrusted input
func (c *DangerousClient) BYPASS_ALL_PERMISSIONS(prompt string, opts *claude.RunOptions) (*claude.ClaudeResult, error) {
	if !c.securityGate.confirmed {
		return nil, fmt.Errorf("security gate not confirmed")
	}

	if opts == nil {
		opts = &claude.RunOptions{}
	}

	// Create a copy to avoid modifying the original
	dangerousOpts := *opts

	// Execute with warning
	fmt.Fprintf(os.Stderr, "🚨 WARNING: Executing Claude with ALL PERMISSIONS BYPASSED 🚨\n")
	fmt.Fprintf(os.Stderr, "This removes all safety controls and allows unrestricted access.\n")

	return c.runWithDangerousFlags(context.Background(), prompt, &dangerousOpts, true, false)
}

// BYPASS_ALL_PERMISSIONS_CTX is the context-aware version of BYPASS_ALL_PERMISSIONS
func (c *DangerousClient) BYPASS_ALL_PERMISSIONS_CTX(ctx context.Context, prompt string, opts *claude.RunOptions) (*claude.ClaudeResult, error) {
	if !c.securityGate.confirmed {
		return nil, fmt.Errorf("security gate not confirmed")
	}

	if opts == nil {
		opts = &claude.RunOptions{}
	}

	// Create a copy to avoid modifying the original
	dangerousOpts := *opts

	// Execute with warning
	fmt.Fprintf(os.Stderr, "🚨 WARNING: Executing Claude with ALL PERMISSIONS BYPASSED 🚨\n")
	fmt.Fprintf(os.Stderr, "This removes all safety controls and allows unrestricted access.\n")

	return c.runWithDangerousFlags(ctx, prompt, &dangerousOpts, true, false)
}

// SET_ENVIRONMENT_VARIABLES injects environment variables into Claude process
// WARNING: Can expose sensitive environment variables to Claude
// WARNING: Can modify Claude's runtime behavior in unexpected ways
func (c *DangerousClient) SET_ENVIRONMENT_VARIABLES(envVars map[string]string) error {
	if !c.securityGate.confirmed {
		return fmt.Errorf("security gate not confirmed")
	}

	// Validate environment variables for obvious security risks
	for key := range envVars {
		if strings.Contains(strings.ToUpper(key), "PASSWORD") ||
			strings.Contains(strings.ToUpper(key), "SECRET") ||
			strings.Contains(strings.ToUpper(key), "TOKEN") ||
			strings.Contains(strings.ToUpper(key), "KEY") {
			fmt.Fprintf(os.Stderr, "⚠️  WARNING: Setting potentially sensitive environment variable: %s\n", key)
		}
		if strings.Contains(strings.ToUpper(key), "PATH") {
			fmt.Fprintf(os.Stderr, "⚠️  WARNING: Modifying PATH environment variable can affect executable resolution\n")
		}
	}

	// Store env vars for later use in command execution
	for key, value := range envVars {
		c.envVars[key] = value
	}

	fmt.Fprintf(os.Stderr, "🔧 SET: %d environment variables configured for Claude process\n", len(envVars))
	return nil
}

// ENABLE_MCP_DEBUG enables detailed MCP server debugging
// WARNING: May expose sensitive information in debug logs
// WARNING: Can significantly impact performance
func (c *DangerousClient) ENABLE_MCP_DEBUG() error {
	if !c.securityGate.confirmed {
		return fmt.Errorf("security gate not confirmed")
	}

	c.mcpDebug = true
	fmt.Fprintf(os.Stderr, "🐛 DEBUG: MCP debugging enabled - sensitive information may be logged\n")
	return nil
}

// DANGEROUS_RunWithEnvironment combines environment injection with execution
// WARNING: This method combines multiple security risks
func (c *DangerousClient) DANGEROUS_RunWithEnvironment(prompt string, opts *claude.RunOptions, envVars map[string]string) (*claude.ClaudeResult, error) {
	return c.DANGEROUS_RunWithEnvironmentCtx(context.Background(), prompt, opts, envVars)
}

// DANGEROUS_RunWithEnvironmentCtx is the context-aware version
func (c *DangerousClient) DANGEROUS_RunWithEnvironmentCtx(ctx context.Context, prompt string, opts *claude.RunOptions, envVars map[string]string) (*claude.ClaudeResult, error) {
	if !c.securityGate.confirmed {
		return nil, fmt.Errorf("security gate not confirmed")
	}

	// Set environment variables
	if err := c.SET_ENVIRONMENT_VARIABLES(envVars); err != nil {
		return nil, err
	}

	// Execute with custom environment
	return c.runWithDangerousFlags(ctx, prompt, opts, false, true)
}

// runWithDangerousFlags executes Claude with dangerous flags enabled
func (c *DangerousClient) runWithDangerousFlags(ctx context.Context, prompt string, opts *claude.RunOptions, skipPermissions bool, useCustomEnv bool) (*claude.ClaudeResult, error) {
	if opts == nil {
		opts = &claude.RunOptions{}
	}

	// Validate and preprocess options using enhanced features
	if err := claude.PreprocessOptions(opts); err != nil {
		return nil, fmt.Errorf("dangerous options validation failed: %w", err)
	}

	// Build arguments using the main package's enhanced BuildArgs
	args := claude.BuildArgs(prompt, opts)

	// Add dangerous-specific flags after the standard args
	if skipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	if c.mcpDebug {
		args = append(args, "--mcp-debug")
		// Ensure verbose is enabled for debug
		if !opts.Verbose {
			args = append(args, "--verbose")
		}
	}

	// Create command with context support
	cmd := exec.CommandContext(ctx, c.ClaudeClient.BinPath, args...)

	// Set custom environment if requested
	if useCustomEnv && len(c.envVars) > 0 {
		// Start with current environment
		cmd.Env = os.Environ()
		// Add custom variables
		for key, value := range c.envVars {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
		fmt.Fprintf(os.Stderr, "🌍 ENV: Using custom environment with %d additional variables\n", len(c.envVars))
	}

	// Execute command with enhanced error handling
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Use enhanced error parsing from main package
		var exitCode int
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
		
		claudeErr := claude.ParseError(stderr.String(), exitCode)
		claudeErr.Original = err
		return nil, fmt.Errorf("dangerous claude command failed: %w", claudeErr)
	}

	// Parse response based on format
	if opts.Format == claude.JSONOutput {
		var result claude.ClaudeResult
		if err := json.Unmarshal([]byte(stdout.String()), &result); err != nil {
			return nil, claude.NewClaudeError(claude.ErrorValidation, fmt.Sprintf("failed to parse JSON response: %v", err))
		}
		return &result, nil
	}

	// For text output, return the raw text
	return &claude.ClaudeResult{
		Result:  stdout.String(),
		IsError: false,
	}, nil
}

// GetSecurityWarnings returns a list of active security bypasses
func (c *DangerousClient) GetSecurityWarnings() []string {
	warnings := []string{}

	if len(c.envVars) > 0 {
		warnings = append(warnings, fmt.Sprintf("Environment injection active (%d variables)", len(c.envVars)))
	}
	if c.mcpDebug {
		warnings = append(warnings, "MCP debug logging enabled")
	}

	return warnings
}

// ResetDangerousSettings clears all dangerous configurations
func (c *DangerousClient) ResetDangerousSettings() {
	c.envVars = make(map[string]string)
	c.mcpDebug = false
	fmt.Fprintf(os.Stderr, "🔄 RESET: All dangerous settings cleared\n")
}

<p align="left">
  <img src="docs/assets/logo.svg" alt="Claude Code Go SDK" height="90">
</p>

# Claude Code Go SDK

[![CI](https://github.com/lancekrogers/claude-code-go/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/lancekrogers/claude-code-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/lancekrogers/claude-code-go.svg)](https://pkg.go.dev/github.com/lancekrogers/claude-code-go)

A comprehensive Go library for programmatically integrating the [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) into Go applications. Build AI-powered coding assistants, automated workflows, and intelligent agents with full control over Claude's capabilities.

## Features

### Core Capabilities

- **Full CLI Wrapper**: Complete access to all Claude Code features
- **Streaming Support**: Real-time response streaming with context cancellation
- **Session Management**: Multi-turn conversations with custom IDs, forking, and persistence control
- **MCP Integration**: Model Context Protocol support for extending Claude with external tools

### Advanced Features

- **Plugin System**: Extensible architecture with logging, metrics, audit, and tool filtering plugins
- **Budget Tracking**: Cost control with spending limits, warnings, and callbacks
- **Subagent Orchestration**: Specialized agents for different tasks (security, code review, testing)
- **Retry & Error Handling**: Configurable retry policies with exponential backoff and jitter
- **Permission Control**: Fine-grained tool permissions with allowlists, blocklists, and modes

### Developer Experience

- **9 Interactive Demos**: Ready-to-run examples showcasing every feature
- **Comprehensive Testing**: Unit and integration tests with mock server support
- **Multiple Output Formats**: Text, JSON, and streaming JSON outputs

## Installation

```bash
go get github.com/lancekrogers/claude-code-go
```

## Prerequisites

- **Claude Max Subscription**: Required for Claude Code CLI
  - [Sign up for Claude Max](https://claude.ai/referral/UKHPp7nGJw)
- **Claude Code CLI**: Installed and accessible in PATH
  - [Installation Guide](https://docs.anthropic.com/en/docs/claude-code/getting-started)

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/lancekrogers/claude-code-go/pkg/claude"
)

func main() {
    client := claude.NewClient("claude")

    result, err := client.RunPrompt("Write a function to calculate Fibonacci numbers", nil)
    if err != nil {
        log.Fatalf("Error: %v", err)
    }

    fmt.Println(result.Result)
}
```

## Interactive Demos

Try the interactive demos to explore SDK capabilities:

```bash
# Clone and run
git clone https://github.com/lancekrogers/claude-code-go
cd claude-code-go

# Core demos
just demo streaming    # Real-time streaming (default)
just demo basic        # Basic JSON output

# Feature demos
just demo sessions     # Session management and forking
just demo mcp          # MCP server integration
just demo retry        # Retry and error handling
just demo permissions  # Permission control system
just demo budget       # Budget tracking with spending limits
just demo plugins      # Plugin system with logging/metrics
just demo subagents    # Multi-agent orchestration
```

See [Demo Showcase](#demo-showcase) below for animated GIFs of each demo.

## Core Features

### Basic Usage

```go
client := claude.NewClient("claude")

// Simple prompt
result, err := client.RunPrompt("Generate a hello world function", &claude.RunOptions{
    Format: claude.JSONOutput,
})

// With custom system prompt
result, err = client.RunWithSystemPrompt(
    "Create a database schema",
    "You are a database architect. Use PostgreSQL best practices.",
    nil,
)
```

### Streaming Responses

```go
ctx := context.Background()
messageCh, errCh := client.StreamPrompt(ctx, "Build a React component", &claude.RunOptions{})

go func() {
    for err := range errCh {
        log.Printf("Error: %v", err)
    }
}()

for msg := range messageCh {
    switch msg.Type {
    case "assistant":
        fmt.Println("Claude:", msg.Result)
    case "result":
        fmt.Printf("Done! Cost: $%.4f\n", msg.CostUSD)
    }
}
```

### Session Management

```go
// Generate a custom session ID
sessionID := claude.GenerateSessionID()

// Start a new session with custom ID
result, err := client.RunPrompt("Write a fibonacci function", &claude.RunOptions{
    SessionID: sessionID,
    Format:    claude.JSONOutput,
})

// Resume the conversation
followup, err := client.ResumeConversation("Now optimize it for performance", result.SessionID)

// Fork a session (create a branch)
forked, err := client.RunPrompt("Try a different approach", &claude.RunOptions{
    ResumeID:    result.SessionID,
    ForkSession: true,
})

// Ephemeral session (no disk persistence)
ephemeral, err := client.RunPrompt("Quick question", &claude.RunOptions{
    NoSessionPersistence: true,
})
```

### MCP Integration

```go
// Single MCP config
result, err := client.RunWithMCP(
    "List files in the project",
    "mcp-config.json",
    []string{"mcp__filesystem__list_directory"},
)

// Multiple MCP configs
result, err = client.RunWithMCPConfigs("Use both tools", []string{
    "filesystem-mcp.json",
    "database-mcp.json",
}, nil)

// Strict mode (only use specified MCP servers)
result, err = client.RunWithStrictMCP("Isolated environment", []string{
    "secure-mcp.json",
}, nil)
```

## Advanced Features

### Plugin System

```go
// Create plugin manager
pm := claude.NewPluginManager()

// Add logging plugin
logger := claude.NewLoggingPlugin(log.Printf)
logger.SanitizeSecrets = true  // Redact API keys, tokens, etc.
logger.TruncateLength = 500    // Limit log output
pm.Register(logger, nil)

// Add metrics plugin
metrics := claude.NewMetricsPlugin()
pm.Register(metrics, nil)

// Add tool filter (block dangerous tools)
filter := claude.NewToolFilterPlugin(map[string]string{
    "Bash(rm*)": "Deletion commands blocked",
})
pm.Register(filter, nil)

// Add audit plugin
audit := claude.NewAuditPlugin(1000) // Keep last 1000 records
pm.Register(audit, nil)

// Use with client
result, err := client.RunPrompt("Do something", &claude.RunOptions{
    PluginManager: pm,
})

// Get metrics
stats := metrics.GetMetrics()
fmt.Printf("Total cost: $%.4f\n", stats["total_cost"])

// Get audit records
records := audit.GetRecords()
```

### Budget Tracking

```go
// Create budget tracker with callbacks
tracker := claude.NewBudgetTracker(&claude.BudgetConfig{
    MaxBudgetUSD:     10.00, // $10 limit
    WarningThreshold: 0.8,   // Warn at 80%
    OnBudgetWarning: func(current, max float64) {
        fmt.Printf("Warning: Budget at %.0f%%\n", (current/max)*100)
    },
    OnBudgetExceeded: func(current, max float64) {
        fmt.Printf("Budget exceeded: $%.2f > $%.2f\n", current, max)
    },
})

// Use with client
result, err := client.RunPrompt("Generate code", &claude.RunOptions{
    MaxBudgetUSD:  10.00,
    BudgetTracker: tracker,
})

// Check budget status
fmt.Printf("Spent: $%.4f, Remaining: $%.4f\n",
    tracker.TotalSpent(), tracker.RemainingBudget())
```

### Subagent Orchestration

```go
// Define specialized agents
agents := map[string]*claude.SubagentConfig{
    "security": {
        Description:   "Security analysis and vulnerability detection",
        SystemPrompt:  "You are a security expert. Analyze code for vulnerabilities.",
        AllowedTools:  []string{"Read(*)", "Grep(*)"},
        Model:         "opus",
    },
    "testing": {
        Description:   "Test generation and coverage analysis",
        SystemPrompt:  "You are a testing expert. Generate comprehensive tests.",
        AllowedTools:  []string{"Read(*)", "Write(*)", "Bash(go test*)"},
    },
}

// Use agents
result, err := client.RunPrompt("Analyze this code", &claude.RunOptions{
    Agents: agents,
})
```

### Retry & Error Handling

```go
// Custom retry policy
policy := &claude.RetryPolicy{
    MaxRetries:    5,
    BaseDelay:     100 * time.Millisecond,
    MaxDelay:      10 * time.Second,
    BackoffFactor: 2.0,
}

// With automatic retry
result, err := client.RunPromptWithRetry("Do something", nil, policy)

// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err = client.RunPromptCtx(ctx, "Quick task", &claude.RunOptions{
    Timeout: 30 * time.Second,
})

// Error classification
if err != nil {
    if claudeErr, ok := err.(*claude.ClaudeError); ok {
        if claudeErr.IsRetryable() {
            // Handle retryable error
        }
        fmt.Printf("Error type: %s\n", claudeErr.Type)
    }
}
```

### Permission Control

```go
// Permission modes
result, err := client.RunPrompt("Edit files", &claude.RunOptions{
    PermissionMode: claude.PermissionModeAcceptEdits, // Auto-approve edits
})

// Tool allowlisting (with glob patterns)
result, err = client.RunPrompt("Work with git", &claude.RunOptions{
    AllowedTools: []string{
        "Read(*)",
        "Bash(git status*)",
        "Bash(git log*)",
        "Bash(git diff*)",
    },
})

// Tool blocklisting
result, err = client.RunPrompt("Safe operations only", &claude.RunOptions{
    DisallowedTools: []string{
        "Bash(rm*)",
        "Bash(curl*)",
        "Write(*)",
    },
})
```

## API Reference

### RunOptions

```go
type RunOptions struct {
    // Output format
    Format       OutputFormat  // text, json, stream-json

    // Prompts
    SystemPrompt string        // Override default system prompt
    AppendPrompt string        // Append to default system prompt

    // Session control
    SessionID            string // Custom session UUID
    ResumeID             string // Resume existing session
    Continue             bool   // Continue most recent conversation
    ForkSession          bool   // Fork from resumed session
    NoSessionPersistence bool   // Don't save to disk

    // MCP configuration
    MCPConfigPath   string   // Single MCP config path
    MCPConfigs      []string // Multiple MCP configs
    StrictMCPConfig bool     // Only use specified MCP servers

    // Tool permissions
    AllowedTools    []string       // Tools Claude can use
    DisallowedTools []string       // Tools Claude cannot use
    PermissionMode  PermissionMode // default, acceptEdits, bypassPermissions

    // Model selection
    Model      string // Full model name
    ModelAlias string // sonnet, opus, haiku

    // Execution control
    MaxTurns int           // Limit agentic turns
    Timeout  time.Duration // Request timeout

    // Budget control
    MaxBudgetUSD  float64        // Spending limit
    BudgetTracker *BudgetTracker // Shared tracker

    // Extensions
    Agents        map[string]*SubagentConfig // Specialized agents
    PluginManager *PluginManager             // Plugin system
}
```

### Core Methods

```go
// Basic execution
func (c *ClaudeClient) RunPrompt(prompt string, opts *RunOptions) (*ClaudeResult, error)
func (c *ClaudeClient) RunPromptCtx(ctx context.Context, prompt string, opts *RunOptions) (*ClaudeResult, error)

// Streaming
func (c *ClaudeClient) StreamPrompt(ctx context.Context, prompt string, opts *RunOptions) (<-chan Message, <-chan error)

// Stdin processing
func (c *ClaudeClient) RunFromStdin(stdin io.Reader, prompt string, opts *RunOptions) (*ClaudeResult, error)

// With retry
func (c *ClaudeClient) RunPromptWithRetry(prompt string, opts *RunOptions, policy *RetryPolicy) (*ClaudeResult, error)

// Session convenience
func (c *ClaudeClient) ContinueConversation(prompt string) (*ClaudeResult, error)
func (c *ClaudeClient) ResumeConversation(prompt, sessionID string) (*ClaudeResult, error)

// MCP convenience
func (c *ClaudeClient) RunWithMCP(prompt, configPath string, tools []string) (*ClaudeResult, error)
func (c *ClaudeClient) RunWithMCPConfigs(prompt string, configs []string, opts *RunOptions) (*ClaudeResult, error)
func (c *ClaudeClient) RunWithStrictMCP(prompt string, configs []string, opts *RunOptions) (*ClaudeResult, error)
```

## Security-Sensitive Features

For advanced use cases requiring bypassed safety controls:

```go
import "github.com/lancekrogers/claude-code-go/pkg/claude/dangerous"

// SECURITY REVIEW REQUIRED
client, err := dangerous.NewDangerousClient("claude")
if err != nil {
    // Fails unless CLAUDE_ENABLE_DANGEROUS="i-accept-all-risks"
    return err
}

// Bypass all permission prompts
result, err := client.BYPASS_ALL_PERMISSIONS("trusted prompt", nil)
```

**Requirements:**

- Set `CLAUDE_ENABLE_DANGEROUS="i-accept-all-risks"`
- Cannot run in production environments
- See [pkg/claude/dangerous/README.md](pkg/claude/dangerous/README.md)

## Testing

```bash
# All tests
just test all

# Unit tests only
just test lib

# Integration tests (mock server)
just test integration

# Coverage report
just coverage
```

## Development

[Just](https://github.com/casey/just) is the primary command runner:

```bash
# Show available commands
just --list

# Build everything
just build all

# Run linting
just lint
```

## Official Documentation

- [Claude Code Overview](https://docs.anthropic.com/en/docs/claude-code/overview)
- [CLI Usage Reference](https://docs.anthropic.com/en/docs/claude-code/cli-usage)
- [SDK Patterns](https://docs.anthropic.com/en/docs/claude-code/sdk)
- [Getting Started](https://docs.anthropic.com/en/docs/claude-code/getting-started)

## Contributing

See [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) for guidelines.

## License

MIT License - see LICENSE file.

## Acknowledgments

- Anthropic for creating Claude Code
- The Go community for excellent tooling

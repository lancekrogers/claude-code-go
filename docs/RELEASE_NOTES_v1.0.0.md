# v1.0.0 - CLI Feature Parity Release

This major release brings the Go SDK to full feature parity with the Claude Code CLI, adding comprehensive SDK-level abstractions for building production applications.

## Highlights

- **Plugin System** - Extensible hooks for logging, metrics, filtering, and auditing
- **Budget Tracking** - Spending limits with warnings and callbacks
- **Session Management** - Multi-turn conversation state
- **Streaming Support** - Real-time response processing
- **Subagent Orchestration** - Spawn and manage multiple Claude instances
- **MCP Integration** - Full Model Context Protocol support
- **Structured Output** - JSON Schema validation for reliable outputs
- **Model Fallback** - Automatic fallback when primary model is overloaded
- **9 Example Programs** - Complete working examples for all features

## New Features

### Plugin System

Build extensible applications with lifecycle hooks:

```go
client := claude.NewClient("claude")

// Add logging plugin
logging := claude.NewLoggingPlugin(log.Printf)
client.AddPlugin(logging)

// Add metrics collection
metrics := claude.NewMetricsPlugin()
client.AddPlugin(metrics)

result, _ := client.RunPrompt(ctx, "Hello", nil)
fmt.Printf("Total cost: $%.4f\n", metrics.TotalCost)
```

**Built-in Plugins:**

- `LoggingPlugin` - Configurable logging with optional secret sanitization
- `MetricsPlugin` - Collect tool call counts, message counts, and costs
- `ToolFilterPlugin` - Block specific tools from being executed
- `AuditPlugin` - Record all tool calls for compliance auditing

### Budget Tracking

Control spending with configurable limits:

```go
budget := claude.NewBudgetTracker(&claude.BudgetConfig{
    MaxBudgetUSD:     10.0,
    WarningThreshold: 0.8,
    OnBudgetWarning: func(current, max float64) {
        log.Printf("Warning: %.0f%% of budget used", (current/max)*100)
    },
})

// Check before expensive operations
if budget.CanSpend(0.50) {
    result, _ := client.RunPrompt(ctx, prompt, nil)
    budget.AddSpend("session1", result.CostUSD)
}
```

### Session Management

Maintain conversation state across multiple turns:

```go
sessions := claude.NewSessionManager()

// First turn
result1, _ := client.RunPrompt(ctx, "My name is Alice", nil)
sessions.SaveSession("chat1", result1.SessionID)

// Later turn - resume the conversation
opts := &claude.RunOptions{
    ResumeID: sessions.GetSession("chat1"),
}
result2, _ := client.RunPrompt(ctx, "What's my name?", opts)
// Claude remembers: "Your name is Alice"
```

### Streaming Support

Process responses in real-time:

```go
msgChan, errChan := client.StreamPrompt(ctx, "Write a story", nil)

for msg := range msgChan {
    switch msg.Type {
    case "assistant":
        fmt.Print(msg.Message) // Print as it arrives
    case "result":
        fmt.Printf("\nCost: $%.4f\n", msg.CostUSD)
    }
}
```

### Subagent Orchestration

Spawn and coordinate multiple Claude instances:

```go
manager := claude.NewSubagentManager(client, &claude.SubagentConfig{
    MaxConcurrent: 3,
})

tasks := []claude.SubagentTask{
    {ID: "research", Prompt: "Research topic A"},
    {ID: "analyze", Prompt: "Analyze topic B"},
    {ID: "summarize", Prompt: "Summarize topic C"},
}

results := manager.ExecuteAll(ctx, tasks)
for _, r := range results {
    fmt.Printf("%s: %s\n", r.TaskID, r.Result)
}
```

### MCP Integration

Use Model Context Protocol tools:

```go
opts := &claude.RunOptions{
    MCPConfigPath: "./mcp-config.json",
    AllowedTools: []string{
        "mcp__filesystem__read_file",
        "mcp__filesystem__write_file",
    },
}

result, _ := client.RunPrompt(ctx, "Read config.json", opts)
```

### Structured Output & Reliability

New options for production reliability:

```go
// Structured output with JSON Schema validation
result, err := client.RunPrompt(ctx, "Generate user profile", &claude.RunOptions{
    Format: claude.JSONOutput,
    JSONSchema: `{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}`,
})

// Automatic fallback when primary model is overloaded
result, err := client.RunPrompt(ctx, "Analyze code", &claude.RunOptions{
    Model:         "opus",
    FallbackModel: "sonnet", // Use sonnet if opus is overloaded
})

// Debug mode with category filtering
result, err := client.RunPrompt(ctx, "Test", &claude.RunOptions{
    Debug: "api,mcp", // Only debug api and mcp categories
})
```

## Examples

Complete working examples are included in `examples/`:

| Example        | Description                                         |
| -------------- | --------------------------------------------------- |
| `budget/`      | Budget tracking with warnings                       |
| `mcp/`         | MCP server integration                              |
| `permissions/` | Permission modes (default, accept-edits, full-auto) |
| `plugins/`     | Plugin system with all built-ins                    |
| `retry/`       | Exponential backoff with jitter                     |
| `sessions/`    | Multi-turn conversation management                  |
| `streaming/`   | Real-time response streaming                        |
| `subagents/`   | Parallel agent execution                            |
| `workflow/`    | Complex multi-step workflows                        |

## Installation

```bash
go get github.com/lancekrogers/claude-code-go@v1.0.0
```

## Upgrade Notes

This release is **fully backward compatible** with v0.1.x. No code changes required to upgrade.

## Full Changelog

**[v0.1.3...v1.0.0](https://github.com/lancekrogers/claude-code-go/compare/v0.1.3...v2.0.0)**

// Package claude is a Go SDK for the Claude Code CLI, letting you build
// AI-powered coding assistants, automated workflows, and intelligent agents on
// top of Claude Code's non-interactive prompt (claude -p) surface.
//
// The SDK shells out to the claude binary and wraps its text, JSON, and
// streaming-JSON outputs behind a typed, context-aware API. It intentionally
// targets the prompt-oriented workflow; interactive sessions and management
// subcommands (auth, mcp, plugins, install, update) are out of scope.
//
// Install:
//
//	go get github.com/lancekrogers/claude-code-go
//
// Quick start:
//
//	cc := claude.NewClient("claude")
//	result, err := cc.RunPrompt("Write a function to calculate Fibonacci numbers", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(result.Result)
//
// # Core API
//
// NewClient returns a [ClaudeClient], the entry point for every operation:
//
//   - [ClaudeClient.RunPrompt] / [ClaudeClient.RunPromptCtx]: run a single
//     prompt and return a [ClaudeResult].
//   - [ClaudeClient.StreamPrompt]: stream [Message] values over channels with
//     context cancellation.
//   - [ClaudeClient.RunWithSession] / [ClaudeClient.RunEphemeral]: multi-turn
//     conversations with resume and forking.
//   - [ClaudeClient.RunWithMCP] / [ClaudeClient.RunWithMCPConfigs]: attach
//     Model Context Protocol servers and tools.
//   - [ClaudeClient.RunPromptWithRetry]: configurable retry policies with
//     exponential backoff.
//
// Behavior is configured through [RunOptions] (output format, model, allowed and
// disallowed tools, permission mode, budget, and more).
//
// # Additional features
//
//   - [SubagentManager] orchestrates specialized agents for tasks such as
//     review or testing.
//   - [PluginManager] adds cross-cutting logging, metrics, audit, and
//     tool-filtering plugins.
//   - [BudgetTracker] enforces spending limits with warnings and callbacks.
//   - [MCPConfigBuilder] constructs MCP server configurations programmatically.
//
// The sub-package [github.com/lancekrogers/claude-code-go/pkg/claude/dangerous]
// exposes security-sensitive operations that bypass Claude's safety mechanisms
// and require explicit opt-in.
package claude

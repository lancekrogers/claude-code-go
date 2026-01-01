# Claude Code Go SDK Demos

This directory contains interactive demo implementations showcasing different aspects of the Claude Code Go SDK.

## 🚀 Streaming Demo (Default)

**Location**: `streaming/`
**Command**: `just demo streaming`

Real-time visibility into Claude's actions using streaming JSON output.

### Features

- ✅ Real-time tool execution display
- ✅ Progress indicators and step descriptions
- ✅ Professional-grade monitoring capabilities

## 📝 Basic Demo

**Location**: `basic/`
**Command**: `just demo basic`

Simple SDK usage with standard JSON output for learning fundamentals.

## 💰 Budget Tracking Demo

**Location**: `budget/`
**Command**: `just demo budget`

Demonstrates the SDK's budget tracking capabilities for cost control.

### Features

- ✅ Real-time spending visualization with progress bar
- ✅ Warning thresholds and exceeded callbacks
- ✅ Per-session cost tracking
- ✅ Budget exceeded protection

## 🔌 Plugin System Demo

**Location**: `plugins/`
**Command**: `just demo plugins`

Showcases the extensible plugin architecture for customizing SDK behavior.

### Features

- ✅ Logging, metrics, and audit plugins
- ✅ Tool filtering for security
- ✅ Real-time plugin callbacks
- ✅ Interactive commands: `metrics`, `audit`

## 🤖 Subagent Orchestration Demo

**Location**: `subagents/`
**Command**: `just demo subagents`

Demonstrates multi-agent orchestration with specialized agents.

### Features

- ✅ Pre-built specialized agents (security, code-review, testing, performance, docs)
- ✅ Agent switching with `@agent` syntax
- ✅ Session persistence and resumption
- ✅ Interactive commands: `agents`, `resume`

## 🔄 Session Management Demo

**Location**: `sessions/`
**Command**: `just demo sessions`

Showcases session control features for conversation management.

### Features

- ✅ Custom session IDs with UUID generation
- ✅ Session forking (branch conversations)
- ✅ Ephemeral sessions (no disk persistence)
- ✅ Session resumption and history tracking
- ✅ Interactive commands: `/custom`, `/fork`, `/ephemeral`, `/history`

## 🔌 MCP Integration Demo

**Location**: `mcp/`
**Command**: `just demo mcp`

Demonstrates Model Context Protocol (MCP) server integration.

### Features

- ✅ MCP server configuration loading
- ✅ Strict mode for isolated environments
- ✅ Tool allowlisting for security
- ✅ Example filesystem MCP setup
- ✅ Interactive commands: `/config`, `/strict`, `/allow`

## 🔁 Retry & Error Handling Demo

**Location**: `retry/`
**Command**: `just demo retry`

Shows the SDK's retry and error handling capabilities.

### Features

- ✅ Configurable retry policies
- ✅ Exponential backoff with jitter
- ✅ Error classification (retryable vs non-retryable)
- ✅ Preset policies (aggressive, conservative)
- ✅ Interactive commands: `/retries`, `/delay`, `/enhanced`

## 🔒 Permission Control Demo

**Location**: `permissions/`
**Command**: `just demo permissions`

Demonstrates permission and tool control features.

### Features

- ✅ Permission modes (default, acceptEdits, bypass)
- ✅ Tool allowlisting with glob patterns
- ✅ Tool blocklisting for security
- ✅ Preset configurations (readonly, safe, git, full)
- ✅ Interactive commands: `/mode`, `/allow`, `/deny`, `/preset`

## Quick Start

```bash
# List all demo commands
just demo

# Core demos
just demo streaming    # Streaming demo (default)
just demo basic        # Basic JSON output

# Feature demos
just demo budget       # Budget tracking
just demo plugins      # Plugin system
just demo subagents    # Subagent orchestration
just demo sessions     # Session management
just demo mcp          # MCP integration
just demo retry        # Retry & error handling
just demo permissions  # Permission control
```

## Demo Overview

| Demo | Focus | Best For |
|------|-------|----------|
| streaming | Real-time output | Production monitoring |
| basic | Simple patterns | Learning fundamentals |
| budget | Cost control | API spending limits |
| plugins | Extensibility | Custom integrations |
| subagents | Multi-agent | Specialized workflows |
| sessions | Conversation control | Multi-turn apps |
| mcp | External tools | Tool integrations |
| retry | Error handling | Production resilience |
| permissions | Security | Enterprise deployments |

---

**💡 Tip**: Start with the streaming demo to see the full power of the SDK, then explore feature demos based on your use case.

# Claude Code Go SDK Demos

This directory contains interactive demo implementations showcasing different aspects of the Claude Code Go SDK.

## 🚀 Streaming Demo (Default)

**Location**: `streaming/`
**Command**: `make demo` or `make demo-streaming`

Real-time visibility into Claude's actions using streaming JSON output.

### Features

- ✅ Real-time tool execution display
- ✅ Progress indicators and step descriptions
- ✅ Professional-grade monitoring capabilities

## 📝 Basic Demo

**Location**: `basic/`
**Command**: `make demo-basic`

Simple SDK usage with standard JSON output for learning fundamentals.

## 💰 Budget Tracking Demo

**Location**: `budget/`
**Command**: `make demo-budget`

Demonstrates the SDK's budget tracking capabilities for cost control.

### Features

- ✅ Real-time spending visualization with progress bar
- ✅ Warning thresholds and exceeded callbacks
- ✅ Per-session cost tracking
- ✅ Budget exceeded protection

## 🔌 Plugin System Demo

**Location**: `plugins/`
**Command**: `make demo-plugins`

Showcases the extensible plugin architecture for customizing SDK behavior.

### Features

- ✅ Logging, metrics, and audit plugins
- ✅ Tool filtering for security
- ✅ Real-time plugin callbacks
- ✅ Interactive commands: `metrics`, `audit`

## 🤖 Subagent Orchestration Demo

**Location**: `subagents/`
**Command**: `make demo-subagents`

Demonstrates multi-agent orchestration with specialized agents.

### Features

- ✅ Pre-built specialized agents (security, code-review, testing, performance, docs)
- ✅ Agent switching with `@agent` syntax
- ✅ Session persistence and resumption
- ✅ Interactive commands: `agents`, `resume`

## Quick Start

```bash
# Core demos
make demo              # Streaming demo (default)
make demo-basic        # Basic JSON output

# Feature demos
make demo-budget       # Budget tracking
make demo-plugins      # Plugin system
make demo-subagents    # Subagent orchestration
```

## Demo Overview

| Demo | Focus | Best For |
|------|-------|----------|
| streaming | Real-time output | Production monitoring |
| basic | Simple patterns | Learning fundamentals |
| budget | Cost control | API spending limits |
| plugins | Extensibility | Custom integrations |
| subagents | Multi-agent | Specialized workflows |

---

**💡 Tip**: Start with the streaming demo to see the full power of the SDK, then explore feature demos based on your use case.


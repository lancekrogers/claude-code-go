# Contributing to Claude Code Go SDK

Thank you for your interest in contributing! This guide will help you understand the project architecture and how to contribute effectively.

## 🎯 Project Overview

This project is a **Go SDK library** that provides programmatic access to the Claude Code CLI. It enables Go developers to integrate Claude Code into their applications via a clean, well-tested API.

**Import Path**: `github.com/lancekrogers/claude-code-go/pkg/claude`

For official Claude Code documentation and SDK patterns, see the [Claude Code SDK Documentation](https://docs.anthropic.com/en/docs/claude-code/sdk).

## 🏗️ Architecture Principles

### SDK Library Focus

**What this project IS:**

- Go SDK library (`pkg/claude/`)
- Subprocess wrapper around official Claude Code CLI
- JSON/streaming response parser
- Convenience methods for common operations
- MCP tool integration

**What this project IS NOT:**

- CLI distribution or replacement
- Standalone application for end users
- Alternative to the official `claude` command

### Key Design Decisions

#### 1. No Custom CLI

We **do not** provide CLI interfaces. Users should use the official `claude` command directly.

**Rationale:**

- Avoids user confusion
- Reduces maintenance burden
- Prevents circular dependencies
- Maintains clear project scope

#### 2. Subprocess Pattern

The SDK executes `claude` CLI as subprocess and parses responses.

**Implementation:**

- Uses `exec.CommandContext()` with proper context handling
- Builds command arguments programmatically
- Parses JSON/text/streaming responses
- Comprehensive error handling

## 🛠️ Development Setup

### Prerequisites

- Go 1.21+ (latest version recommended)
- Claude Code CLI installed
- Make or Task runner

### Quick Start

```bash
# Clone the repository
git clone https://github.com/lancekrogers/claude-code-go
cd claude-code-go

# Install dependencies
go mod download

# Run tests
make test-local

# Build examples
make build-examples

# Run integration tests
make test-integration
```

### Development Commands

```bash
# Core development
make build          # Build library and examples
make test-local     # Run all tests
make coverage       # Generate coverage report

# Examples and demos
make demo           # Run interactive demo
make build-examples # Build all examples

# Testing variants
make test-lib       # Core library tests only
make test-dangerous # Test dangerous package
make test-integration # Integration tests with mock server
```

## 📝 Code Guidelines

### Go Standards

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting
- Run `go vet` before submitting
- Write comprehensive tests

### API Design

- All blocking operations must accept `context.Context`
- Provide both context-aware and convenience methods:

  ```go
  // Context-aware (preferred)
  func (c *Client) RunPromptCtx(ctx context.Context, prompt string, opts *RunOptions) (*Result, error)
  
  // Convenience wrapper
  func (c *Client) RunPrompt(prompt string, opts *RunOptions) (*Result, error) {
      return c.RunPromptCtx(context.Background(), prompt, opts)
  }
  ```

- Use proper error wrapping with `fmt.Errorf("operation failed: %w", err)`

### Testing Requirements

- **Minimum 80% test coverage** for new code
- Unit tests for all public methods
- Integration tests for end-to-end workflows
- Mock subprocess execution in unit tests
- Use table-driven tests where appropriate

### Dependencies

- **Minimize external dependencies**
- Only add dependencies that enhance core SDK functionality
- No CLI frameworks or command-line parsing libraries
- Prefer standard library when possible

## 🧪 Testing Strategy

### Test Types

#### Unit Tests (`pkg/claude/*_test.go`)

- Test core SDK functionality
- Mock command execution using dependency injection
- Validate argument building and response parsing
- Test error conditions and edge cases

#### Integration Tests (`test/integration/`)

- Test full end-to-end workflows
- Use mock Claude server for reliable testing
- Validate MCP integration
- Test streaming functionality

#### Example Tests

- Ensure all examples compile and run
- Validate usage patterns
- Test documentation examples

### Running Tests

```bash
# All tests (recommended)
make test-local

# Specific test suites
make test-lib              # Core library only
make test-dangerous        # Dangerous package only
make test-integration      # Integration tests with mock

# With coverage
make coverage
open coverage/coverage.html
```

### Mock Server Testing

We provide a mock Claude server for integration testing:

- HTTP server simulating Claude Code responses
- Enables testing without Claude CLI dependency
- Located in `test/mockserver/`

## 🎬 Demo GIF Generation

Interactive demos showcase the SDK features. The GIFs are recordings of real Go demos running with actual Claude Code CLI interactions.

### Prerequisites

```bash
brew install expect asciinema just
cargo install --git https://github.com/asciinema/agg
```

### Generating GIFs

```bash
# Generate single demo GIF
just demo gif basic

# Generate all demo GIFs
just demo gif-all

# List available demos
just demo gif-list
```

> **Note:** Recording demos makes real API calls via Claude Code CLI. Keep demos short to control costs.

### Adding a New Demo

1. Create interactive demo: `examples/demo/<name>/cmd/demo/main.go`
2. Create expect script: `scripts/demo-expect/<name>.exp`
3. Add build targets to `.justfiles/demos.just`
4. Add documentation section to `docs/DEMOS.md`
5. Generate the GIF: `just demo gif <name>`

## 📂 Project Structure

```
claude-code-go/
├── pkg/claude/           # Core SDK library
│   ├── claude.go         # Main client implementation
│   ├── claude_test.go    # Unit tests
│   └── dangerous/        # Advanced/unsafe features
├── examples/             # Usage examples
│   ├── basic/           # Simple usage patterns
│   ├── advanced/        # MCP and advanced features
│   ├── testing/         # Testing utilities
│   └── demo/            # Interactive demos
├── test/                # Integration tests
│   ├── integration/     # End-to-end tests
│   ├── mockserver/      # Mock Claude server
│   └── fixtures/        # Test data
├── Makefile             # Build automation
├── Taskfile.yml         # Alternative task runner
└── go.mod               # Go module definition
```

## 🚀 Contributing Process

### 1. Before Starting

- Check existing issues and discussions
- Discuss major changes in an issue first
- Ensure your Go version meets requirements

### 2. Development Workflow

```bash
# Fork and clone your fork
git clone https://github.com/YOUR_USERNAME/claude-code-go
cd claude-code-go

# Create feature branch
git checkout -b feature/your-feature-name

# Make changes and test
make test-local
make build-examples

# Commit with clear messages
git commit -m "Add streaming timeout support

- Add context timeout handling in StreamPrompt
- Update tests to verify timeout behavior  
- Add example demonstrating timeout usage"
```

### 3. Pull Request Guidelines

- **Clear title** describing the change
- **Comprehensive description** with motivation and approach
- **Link related issues** using "Fixes #123" or "Related to #456"
- **Add tests** for new functionality
- **Update examples** if API changes
- **Ensure CI passes** before requesting review

### 4. Code Review Process

- Maintainers will review for design, correctness, and style
- Address feedback promptly and thoroughly
- Be responsive to questions and suggestions
- Squash commits before merge if requested

## 📋 Common Contribution Areas

### High-Impact Contributions

- **New convenience methods** for common use cases
- **Improved error handling** and error messages
- **Performance optimizations** for subprocess execution
- **Better streaming support** and real-time features
- **Enhanced MCP integration** and tool support

### Documentation Improvements

- **Code examples** showing real-world usage
- **API documentation** improvements
- **Architecture diagrams** and explanations
- **Testing guides** and best practices

### Testing and Quality

- **Edge case testing** for error conditions
- **Performance benchmarks** and load testing
- **Cross-platform compatibility** testing
- **Memory leak detection** and fixes

## 🚫 What NOT to Contribute

- **CLI interfaces** or command-line tools
- **Standalone applications** for end users
- **Alternative Claude implementations**
- **Unrelated utilities** that don't enhance the SDK

## 🆘 Getting Help

- **GitHub Discussions** for questions and ideas
- **GitHub Issues** for bugs and feature requests
- **Code examples** in `examples/` directory
- **Existing tests** as reference implementations

## 📜 Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and improve
- Assume good intentions

---

## 🎉 Recognition

Contributors will be recognized in:

- GitHub contributors page
- Release notes for significant contributions
- Project documentation and examples

Thank you for helping make Claude Code Go SDK better! 🚀

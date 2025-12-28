// Package main demonstrates MCP (Model Context Protocol) configuration in the Claude Code Go SDK.
// This example shows how to:
// 1. Use MCPConfigBuilder to create configurations programmatically
// 2. Configure HTTP, SSE, and Stdio MCP servers
// 3. Add environment variables to server configurations
// 4. Merge configurations and perform file I/O
// 5. Use MCP servers with Claude Code execution
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

func main() {
	// =========================================================================
	// Example 1: Building MCP Configuration with the Builder Pattern
	// =========================================================================
	// MCPConfigBuilder provides a fluent API for constructing MCP server
	// configurations programmatically.
	fmt.Println("=== Example 1: MCPConfigBuilder Basics ===")

	// Create a new builder instance
	builder := claude.NewMCPConfigBuilder()

	// Add an HTTP-based MCP server
	// HTTP servers are useful for REST-style API integrations
	builder.AddHTTPServer("api-server", "https://api.example.com/mcp")

	// Add an SSE (Server-Sent Events) server
	// SSE servers provide real-time event streaming
	builder.AddSSEServer("events-server", "https://events.example.com/stream")

	// Add a stdio-based server (local process)
	// Stdio servers run as local processes and communicate via stdin/stdout
	builder.AddStdioServer("local-db", "sqlite-mcp", "--database", "./data.db")

	// Check the configuration
	fmt.Printf("Servers configured: %d\n", builder.ServerCount())
	fmt.Printf("Has 'api-server': %v\n", builder.HasServer("api-server"))
	fmt.Printf("Has 'unknown': %v\n", builder.HasServer("unknown"))
	fmt.Println()

	// =========================================================================
	// Example 2: Adding Environment Variables to Servers
	// =========================================================================
	fmt.Println("=== Example 2: Environment Variables ===")

	// Create a builder with a server that needs credentials
	// SECURITY WARNING: Never hardcode credentials in production code.
	// Use environment variables or secret managers instead:
	//   apiKey := os.Getenv("API_KEY")
	//   apiSecret := os.Getenv("API_SECRET")
	secureBuilder := claude.NewMCPConfigBuilder().
		AddHTTPServer("secure-api", "https://secure.example.com/mcp").
		WithEnv("secure-api", map[string]string{
			"API_KEY":    "your-api-key-here",    // In production: os.Getenv("API_KEY")
			"API_SECRET": "your-api-secret-here", // In production: os.Getenv("API_SECRET")
		})

	// You can also add a stdio server with environment variables
	secureBuilder.
		AddStdioServer("postgres-mcp", "postgres-mcp-server").
		WithEnv("postgres-mcp", map[string]string{
			"DATABASE_URL": "postgres://localhost:5432/mydb",
			"LOG_LEVEL":    "debug",
		})

	// Build and show the JSON configuration
	jsonConfig, err := secureBuilder.BuildJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building JSON: %v\n", err)
	} else {
		fmt.Println("Generated MCP Configuration:")
		fmt.Println(jsonConfig)
	}
	fmt.Println()

	// =========================================================================
	// Example 3: Different Server Types
	// =========================================================================
	fmt.Println("=== Example 3: Server Type Comparison ===")

	// HTTP Server - for REST APIs
	// Best for: APIs that respond to individual requests
	httpBuilder := claude.NewMCPConfigBuilder().
		AddHTTPServer("rest-api", "https://api.example.com/v1/mcp")

	// SSE Server - for event streaming
	// Best for: Real-time updates, notifications, log streaming
	sseBuilder := claude.NewMCPConfigBuilder().
		AddSSEServer("realtime", "https://stream.example.com/events")

	// Stdio Server - for local processes
	// Best for: Local tools, databases, file operations
	stdioBuilder := claude.NewMCPConfigBuilder().
		AddStdioServer("filesystem", "fs-mcp", "--root", "/home/user/project")

	// Show the different configurations
	fmt.Println("HTTP Server Config:")
	httpJSON, _ := httpBuilder.BuildJSON()
	fmt.Println(httpJSON)

	fmt.Println("\nSSE Server Config:")
	sseJSON, _ := sseBuilder.BuildJSON()
	fmt.Println(sseJSON)

	fmt.Println("\nStdio Server Config:")
	stdioJSON, _ := stdioBuilder.BuildJSON()
	fmt.Println(stdioJSON)
	fmt.Println()

	// =========================================================================
	// Example 4: Custom Server Configuration
	// =========================================================================
	fmt.Println("=== Example 4: Custom Server Configuration ===")

	// For more complex configurations, use AddServer with MCPServerConfig
	customBuilder := claude.NewMCPConfigBuilder().
		AddServer("custom-server", claude.MCPServerConfig{
			Command: "custom-mcp-tool",
			Args:    []string{"--config", "/etc/mcp/config.yaml", "--verbose"},
			Env: map[string]string{
				"CUSTOM_VAR":  "value",
				"DEBUG_MODE":  "true",
				"CONFIG_PATH": "/etc/mcp",
			},
			Type: claude.MCPServerTypeStdio,
		})

	customJSON, _ := customBuilder.BuildJSON()
	fmt.Println("Custom Server Config:")
	fmt.Println(customJSON)
	fmt.Println()

	// =========================================================================
	// Example 5: Config File I/O
	// =========================================================================
	fmt.Println("=== Example 5: Configuration File I/O ===")

	// Create a temporary directory for our config files
	tempDir, err := os.MkdirTemp("", "mcp-example-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir)

	// Write configuration to a file
	configPath := filepath.Join(tempDir, "mcp-config.json")
	err = builder.WriteFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
	} else {
		fmt.Printf("Configuration written to: %s\n", configPath)
	}

	// Read configuration from a file
	loadedConfig, err := claude.LoadMCPConfigFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
	} else {
		fmt.Printf("Loaded %d servers from file\n", len(loadedConfig.MCPServers))
		fmt.Printf("Server names: %v\n", loadedConfig.ServerNames())
	}
	fmt.Println()

	// =========================================================================
	// Example 6: Configuration Merging
	// =========================================================================
	fmt.Println("=== Example 6: Configuration Merging ===")

	// Create a base configuration
	baseConfig := claude.NewMCPConfigBuilder().
		AddHTTPServer("base-api", "https://base.example.com/mcp").
		AddStdioServer("base-tool", "base-mcp-tool").
		Build()

	// Create an overlay configuration (e.g., for a specific environment)
	overlayConfig := claude.NewMCPConfigBuilder().
		AddHTTPServer("base-api", "https://production.example.com/mcp").  // Override
		AddSSEServer("monitoring", "https://monitor.example.com/events"). // New
		Build()

	// Merge configurations (overlay takes precedence)
	mergedConfig := baseConfig.Merge(overlayConfig)

	fmt.Printf("Base servers: %v\n", baseConfig.ServerNames())
	fmt.Printf("Overlay servers: %v\n", overlayConfig.ServerNames())
	fmt.Printf("Merged servers: %v\n", mergedConfig.ServerNames())

	// Access a specific server from the merged config
	if server, ok := mergedConfig.GetServer("base-api"); ok {
		fmt.Printf("base-api URL after merge: %s\n", server.URL)
	}
	fmt.Println()

	// =========================================================================
	// Example 7: Using MCP with Claude Code Execution
	// =========================================================================
	fmt.Println("=== Example 7: Using MCP with Claude Code ===")

	// Write a config file to use with Claude
	mcpConfigPath := filepath.Join(tempDir, "claude-mcp.json")
	exampleConfig := claude.NewMCPConfigBuilder().
		AddStdioServer("filesystem", "fs-mcp", "--root", ".").
		Build()

	err = exampleConfig.WriteFile(mcpConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing MCP config: %v\n", err)
		return
	}

	// Create Claude client
	client := claude.NewClient("claude")

	// Method 1: Use a single MCP config file
	fmt.Println("Method 1: Single MCP config file")
	fmt.Printf("Would execute with: --mcp-config %s\n", mcpConfigPath)

	// Example usage (commented out as it requires claude CLI):
	// result, err := client.RunPrompt("List files in the current directory", &claude.RunOptions{
	//     Format:        claude.JSONOutput,
	//     MCPConfigPath: mcpConfigPath,
	//     AllowedTools:  []string{"mcp__filesystem__list_files"},
	// })

	// Method 2: Use inline JSON config
	fmt.Println("\nMethod 2: Inline JSON config")
	inlineConfig, _ := exampleConfig.ToJSON()
	fmt.Printf("Would execute with: --mcp-config '%s'\n", inlineConfig[:50]+"...")

	// Example usage:
	// result, err := client.RunWithMCPConfigs("List files", []string{inlineConfig}, nil)

	// Method 3: Use multiple MCP configurations
	fmt.Println("\nMethod 3: Multiple MCP configurations")
	config1Path := filepath.Join(tempDir, "config1.json")
	config2Path := filepath.Join(tempDir, "config2.json")

	claude.NewMCPConfigBuilder().
		AddStdioServer("tool1", "tool1-mcp").
		WriteFile(config1Path)

	claude.NewMCPConfigBuilder().
		AddStdioServer("tool2", "tool2-mcp").
		WriteFile(config2Path)

	fmt.Printf("Would execute with multiple configs: %s, %s\n", config1Path, config2Path)

	// Example usage:
	// result, err := client.RunWithMCPConfigs("Use both tools", []string{config1Path, config2Path}, nil)

	// Method 4: Strict MCP mode (only use specified servers)
	fmt.Println("\nMethod 4: Strict MCP mode")
	fmt.Println("Would execute with: --strict-mcp-config (ignores other MCP configs)")

	// Example usage:
	// result, err := client.RunWithStrictMCP("Use only these tools", []string{mcpConfigPath}, nil)
	_ = client // Avoid unused variable warning
	fmt.Println()

	// =========================================================================
	// Summary
	// =========================================================================
	fmt.Println("=== MCP Configuration Summary ===")
	fmt.Println(`MCP Server Types:
- HTTP:  REST API endpoints (MCPServerTypeHTTP)
- SSE:   Server-Sent Events for streaming (MCPServerTypeSSE)
- Stdio: Local processes via stdin/stdout (MCPServerTypeStdio)

MCPConfigBuilder Methods:
- NewMCPConfigBuilder()           - Create new builder
- AddHTTPServer(name, url)        - Add HTTP server
- AddSSEServer(name, url)         - Add SSE server
- AddStdioServer(name, cmd, args) - Add stdio server
- AddServer(name, config)         - Add custom config
- WithEnv(name, envMap)           - Set environment vars
- WithArgs(name, args)            - Add command args
- Build()                         - Get MCPConfig struct
- BuildJSON()                     - Get JSON string
- WriteFile(path)                 - Write to file

MCPConfig Methods:
- LoadMCPConfigFile(path)         - Load from file
- ParseMCPConfig(json)            - Parse JSON string
- Merge(other)                    - Merge configurations
- ServerNames()                   - List server names
- GetServer(name)                 - Get server config
- ToJSON()                        - Convert to JSON
- WriteFile(path)                 - Write to file

Usage with ClaudeClient:
- MCPConfigPath      - Single config file path
- MCPConfigs[]       - Multiple config paths/JSON strings
- StrictMCPConfig    - Only use specified servers
- AllowedTools       - Include mcp__<server>__<tool> format`)
}

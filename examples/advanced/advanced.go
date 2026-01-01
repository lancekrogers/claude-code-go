package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

// MCPConfig represents the MCP server configuration
type MCPConfig struct {
	MCPServers map[string]struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env,omitempty"`
	} `json:"mcpServers"`
}

func main() {
	// Create a Claude client
	cc := claude.NewClient("claude")

	// Create an MCP configuration
	mcpConfig := MCPConfig{
		MCPServers: map[string]struct {
			Command string            `json:"command"`
			Args    []string          `json:"args"`
			Env     map[string]string `json:"env,omitempty"`
		}{
			"filesystem": {
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "./"},
			},
			"github": {
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-github"},
				Env: map[string]string{
					"GITHUB_TOKEN": os.Getenv("GITHUB_TOKEN"),
				},
			},
		},
	}

	// Write MCP config to a temporary file
	mcpFile, err := os.CreateTemp("", "mcp-config-*.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(mcpFile.Name())

	encoder := json.NewEncoder(mcpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(mcpConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write MCP config: %v\n", err)
		os.Exit(1)
	}
	mcpFile.Close()

	fmt.Printf("Created MCP config at %s\n", mcpFile.Name())

	// Run Claude with MCP configuration
	fmt.Println("Example: Using MCP tools")

	// List allowed MCP tools
	allowedTools := []string{
		"mcp__filesystem__read_file",
		"mcp__filesystem__list_directory",
		"mcp__github__get_repository",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use streaming to show tool usage in real-time
	messageCh, errCh := cc.StreamPrompt(
		ctx,
		"List all files in the current directory and show the contents of any go.mod files",
		&claude.RunOptions{
			Format:        claude.StreamJSONOutput,
			MCPConfigPath: mcpFile.Name(),
			AllowedTools:  allowedTools,
			MaxTurns:      5,
		},
	)

	// Handle errors
	go func() {
		for err := range errCh {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}()

	// Process messages
	for msg := range messageCh {
		// Display message type
		fmt.Printf("Message type: %s\n", msg.Type)

		// Only print full content for important messages
		if msg.Type == "system" && msg.Subtype == "init" {
			fmt.Println("Initialized with tools:", msg.Tools)
			fmt.Println("MCP servers:", msg.MCPServers)
		} else if msg.Type == "result" {
			fmt.Printf("Final result (cost: $%.4f, turns: %d):\n", msg.CostUSD, msg.NumTurns)
			fmt.Println(msg.Result)
		}
	}
}

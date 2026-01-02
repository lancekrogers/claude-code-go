package claude

import (
	"encoding/json"
	"fmt"
	"os"
)

// MCPServerType represents the type of MCP server connection
type MCPServerType string

const (
	// MCPServerTypeHTTP is for HTTP-based MCP servers
	MCPServerTypeHTTP MCPServerType = "http"
	// MCPServerTypeSSE is for Server-Sent Events based MCP servers
	MCPServerTypeSSE MCPServerType = "sse"
	// MCPServerTypeStdio is for stdio-based MCP servers (local processes)
	MCPServerTypeStdio MCPServerType = "stdio"
)

// MCPServerConfig represents an MCP server configuration
type MCPServerConfig struct {
	// Command is the command to run for stdio servers
	Command string `json:"command,omitempty"`
	// Args are the arguments for the command
	Args []string `json:"args,omitempty"`
	// Env is environment variables for the server process
	Env map[string]string `json:"env,omitempty"`
	// URL is the endpoint URL for http/sse servers
	URL string `json:"url,omitempty"`
	// Type is the server connection type (http, sse, stdio)
	Type MCPServerType `json:"type,omitempty"`
}

// MCPConfig represents the full MCP configuration file format
type MCPConfig struct {
	// MCPServers maps server names to their configurations
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPConfigBuilder helps construct MCP configurations programmatically
type MCPConfigBuilder struct {
	servers map[string]MCPServerConfig
}

// NewMCPConfigBuilder creates a new MCP configuration builder
func NewMCPConfigBuilder() *MCPConfigBuilder {
	return &MCPConfigBuilder{
		servers: make(map[string]MCPServerConfig),
	}
}

// AddHTTPServer adds an HTTP-based MCP server
func (b *MCPConfigBuilder) AddHTTPServer(name, url string) *MCPConfigBuilder {
	b.servers[name] = MCPServerConfig{
		URL:  url,
		Type: MCPServerTypeHTTP,
	}
	return b
}

// AddSSEServer adds a Server-Sent Events based MCP server
func (b *MCPConfigBuilder) AddSSEServer(name, url string) *MCPConfigBuilder {
	b.servers[name] = MCPServerConfig{
		URL:  url,
		Type: MCPServerTypeSSE,
	}
	return b
}

// AddStdioServer adds a stdio-based MCP server (local process)
func (b *MCPConfigBuilder) AddStdioServer(name, command string, args ...string) *MCPConfigBuilder {
	b.servers[name] = MCPServerConfig{
		Command: command,
		Args:    args,
		Type:    MCPServerTypeStdio,
	}
	return b
}

// AddServer adds a custom server configuration
func (b *MCPConfigBuilder) AddServer(name string, config MCPServerConfig) *MCPConfigBuilder {
	b.servers[name] = config
	return b
}

// WithEnv sets environment variables for a server
func (b *MCPConfigBuilder) WithEnv(name string, env map[string]string) *MCPConfigBuilder {
	if server, ok := b.servers[name]; ok {
		server.Env = env
		b.servers[name] = server
	}
	return b
}

// WithArgs sets additional arguments for a stdio server
func (b *MCPConfigBuilder) WithArgs(name string, args ...string) *MCPConfigBuilder {
	if server, ok := b.servers[name]; ok {
		server.Args = append(server.Args, args...)
		b.servers[name] = server
	}
	return b
}

// Build returns the MCPConfig struct
func (b *MCPConfigBuilder) Build() *MCPConfig {
	// Create a copy to prevent external modifications
	servers := make(map[string]MCPServerConfig, len(b.servers))
	for k, v := range b.servers {
		servers[k] = v
	}
	return &MCPConfig{
		MCPServers: servers,
	}
}

// BuildJSON returns the configuration as a JSON string
func (b *MCPConfigBuilder) BuildJSON() (string, error) {
	config := b.Build()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP config: %w", err)
	}
	return string(data), nil
}

// WriteFile writes the configuration to a file
func (b *MCPConfigBuilder) WriteFile(path string) error {
	jsonData, err := b.BuildJSON()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(jsonData), 0644)
}

// ServerCount returns the number of servers configured
func (b *MCPConfigBuilder) ServerCount() int {
	return len(b.servers)
}

// HasServer checks if a server with the given name exists
func (b *MCPConfigBuilder) HasServer(name string) bool {
	_, ok := b.servers[name]
	return ok
}

// RemoveServer removes a server from the configuration
func (b *MCPConfigBuilder) RemoveServer(name string) *MCPConfigBuilder {
	delete(b.servers, name)
	return b
}

// Clear removes all servers from the configuration
func (b *MCPConfigBuilder) Clear() *MCPConfigBuilder {
	b.servers = make(map[string]MCPServerConfig)
	return b
}

// ParseMCPConfig parses a JSON configuration string
func ParseMCPConfig(jsonStr string) (*MCPConfig, error) {
	var config MCPConfig
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		return nil, fmt.Errorf("failed to parse MCP config: %w", err)
	}
	return &config, nil
}

// LoadMCPConfigFile loads and parses an MCP configuration file
func LoadMCPConfigFile(path string) (*MCPConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP config file: %w", err)
	}
	return ParseMCPConfig(string(data))
}

// ToJSON converts an MCPConfig to a JSON string
func (c *MCPConfig) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP config: %w", err)
	}
	return string(data), nil
}

// WriteFile writes the MCPConfig to a file
func (c *MCPConfig) WriteFile(path string) error {
	jsonData, err := c.ToJSON()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(jsonData), 0644)
}

// Merge combines this configuration with another, with other taking precedence
func (c *MCPConfig) Merge(other *MCPConfig) *MCPConfig {
	if other == nil {
		return c
	}

	result := &MCPConfig{
		MCPServers: make(map[string]MCPServerConfig),
	}

	// Copy servers from this config
	for name, server := range c.MCPServers {
		result.MCPServers[name] = server
	}

	// Merge/override with servers from other config
	for name, server := range other.MCPServers {
		result.MCPServers[name] = server
	}

	return result
}

// ServerNames returns a list of all server names in the configuration
func (c *MCPConfig) ServerNames() []string {
	names := make([]string, 0, len(c.MCPServers))
	for name := range c.MCPServers {
		names = append(names, name)
	}
	return names
}

// GetServer returns a server configuration by name
func (c *MCPConfig) GetServer(name string) (MCPServerConfig, bool) {
	server, ok := c.MCPServers[name]
	return server, ok
}

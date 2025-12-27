package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewMCPConfigBuilder(t *testing.T) {
	builder := NewMCPConfigBuilder()
	if builder == nil {
		t.Fatal("NewMCPConfigBuilder() returned nil")
	}
	if builder.servers == nil {
		t.Fatal("Builder servers map is nil")
	}
	if len(builder.servers) != 0 {
		t.Errorf("New builder should have 0 servers, got %d", len(builder.servers))
	}
}

func TestMCPConfigBuilder_AddHTTPServer(t *testing.T) {
	builder := NewMCPConfigBuilder()
	builder.AddHTTPServer("api", "https://api.example.com")

	if !builder.HasServer("api") {
		t.Fatal("Expected server 'api' to exist")
	}

	config := builder.Build()
	server, ok := config.MCPServers["api"]
	if !ok {
		t.Fatal("Expected server 'api' in built config")
	}

	if server.URL != "https://api.example.com" {
		t.Errorf("Expected URL 'https://api.example.com', got %q", server.URL)
	}
	if server.Type != MCPServerTypeHTTP {
		t.Errorf("Expected type %q, got %q", MCPServerTypeHTTP, server.Type)
	}
}

func TestMCPConfigBuilder_AddSSEServer(t *testing.T) {
	builder := NewMCPConfigBuilder()
	builder.AddSSEServer("events", "https://sse.example.com/stream")

	config := builder.Build()
	server, ok := config.MCPServers["events"]
	if !ok {
		t.Fatal("Expected server 'events' in built config")
	}

	if server.URL != "https://sse.example.com/stream" {
		t.Errorf("Expected URL 'https://sse.example.com/stream', got %q", server.URL)
	}
	if server.Type != MCPServerTypeSSE {
		t.Errorf("Expected type %q, got %q", MCPServerTypeSSE, server.Type)
	}
}

func TestMCPConfigBuilder_AddStdioServer(t *testing.T) {
	builder := NewMCPConfigBuilder()
	builder.AddStdioServer("local", "python", "-m", "server")

	config := builder.Build()
	server, ok := config.MCPServers["local"]
	if !ok {
		t.Fatal("Expected server 'local' in built config")
	}

	if server.Command != "python" {
		t.Errorf("Expected command 'python', got %q", server.Command)
	}
	if len(server.Args) != 2 || server.Args[0] != "-m" || server.Args[1] != "server" {
		t.Errorf("Expected args [-m, server], got %v", server.Args)
	}
	if server.Type != MCPServerTypeStdio {
		t.Errorf("Expected type %q, got %q", MCPServerTypeStdio, server.Type)
	}
}

func TestMCPConfigBuilder_AddServer(t *testing.T) {
	builder := NewMCPConfigBuilder()
	custom := MCPServerConfig{
		Command: "node",
		Args:    []string{"server.js"},
		Env:     map[string]string{"PORT": "3000"},
		Type:    MCPServerTypeStdio,
	}
	builder.AddServer("custom", custom)

	config := builder.Build()
	server, ok := config.MCPServers["custom"]
	if !ok {
		t.Fatal("Expected server 'custom' in built config")
	}

	if server.Command != "node" {
		t.Errorf("Expected command 'node', got %q", server.Command)
	}
	if server.Env["PORT"] != "3000" {
		t.Errorf("Expected env PORT=3000, got %v", server.Env)
	}
}

func TestMCPConfigBuilder_WithEnv(t *testing.T) {
	builder := NewMCPConfigBuilder()
	builder.AddStdioServer("test", "python", "server.py")
	builder.WithEnv("test", map[string]string{
		"DEBUG": "true",
		"PORT":  "8080",
	})

	config := builder.Build()
	server := config.MCPServers["test"]

	if server.Env["DEBUG"] != "true" {
		t.Errorf("Expected DEBUG=true, got %q", server.Env["DEBUG"])
	}
	if server.Env["PORT"] != "8080" {
		t.Errorf("Expected PORT=8080, got %q", server.Env["PORT"])
	}
}

func TestMCPConfigBuilder_WithEnv_NonExistent(t *testing.T) {
	builder := NewMCPConfigBuilder()
	// This should not panic or error
	builder.WithEnv("nonexistent", map[string]string{"KEY": "value"})

	if builder.HasServer("nonexistent") {
		t.Error("WithEnv should not create a server")
	}
}

func TestMCPConfigBuilder_WithArgs(t *testing.T) {
	builder := NewMCPConfigBuilder()
	builder.AddStdioServer("test", "python")
	builder.WithArgs("test", "-m", "module", "--verbose")

	config := builder.Build()
	server := config.MCPServers["test"]

	expected := []string{"-m", "module", "--verbose"}
	if len(server.Args) != len(expected) {
		t.Errorf("Expected %d args, got %d", len(expected), len(server.Args))
	}
	for i, arg := range expected {
		if i < len(server.Args) && server.Args[i] != arg {
			t.Errorf("Expected arg[%d]=%q, got %q", i, arg, server.Args[i])
		}
	}
}

func TestMCPConfigBuilder_Chaining(t *testing.T) {
	config := NewMCPConfigBuilder().
		AddHTTPServer("api1", "https://api1.example.com").
		AddHTTPServer("api2", "https://api2.example.com").
		AddSSEServer("events", "https://sse.example.com").
		AddStdioServer("local", "python", "server.py").
		WithEnv("local", map[string]string{"DEBUG": "1"}).
		Build()

	if len(config.MCPServers) != 4 {
		t.Errorf("Expected 4 servers, got %d", len(config.MCPServers))
	}
}

func TestMCPConfigBuilder_ServerCount(t *testing.T) {
	builder := NewMCPConfigBuilder()
	if builder.ServerCount() != 0 {
		t.Errorf("Expected 0 servers, got %d", builder.ServerCount())
	}

	builder.AddHTTPServer("api", "https://example.com")
	if builder.ServerCount() != 1 {
		t.Errorf("Expected 1 server, got %d", builder.ServerCount())
	}

	builder.AddSSEServer("sse", "https://sse.example.com")
	if builder.ServerCount() != 2 {
		t.Errorf("Expected 2 servers, got %d", builder.ServerCount())
	}
}

func TestMCPConfigBuilder_RemoveServer(t *testing.T) {
	builder := NewMCPConfigBuilder()
	builder.AddHTTPServer("api", "https://example.com")
	builder.AddSSEServer("sse", "https://sse.example.com")

	builder.RemoveServer("api")

	if builder.HasServer("api") {
		t.Error("Server 'api' should have been removed")
	}
	if !builder.HasServer("sse") {
		t.Error("Server 'sse' should still exist")
	}
	if builder.ServerCount() != 1 {
		t.Errorf("Expected 1 server, got %d", builder.ServerCount())
	}
}

func TestMCPConfigBuilder_Clear(t *testing.T) {
	builder := NewMCPConfigBuilder()
	builder.AddHTTPServer("api1", "https://api1.example.com")
	builder.AddHTTPServer("api2", "https://api2.example.com")
	builder.Clear()

	if builder.ServerCount() != 0 {
		t.Errorf("Expected 0 servers after Clear, got %d", builder.ServerCount())
	}
}

func TestMCPConfigBuilder_BuildJSON(t *testing.T) {
	builder := NewMCPConfigBuilder()
	builder.AddHTTPServer("api", "https://api.example.com")

	jsonStr, err := builder.BuildJSON()
	if err != nil {
		t.Fatalf("BuildJSON() error: %v", err)
	}

	// Verify it's valid JSON
	var parsed MCPConfig
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse BuildJSON output: %v", err)
	}

	if _, ok := parsed.MCPServers["api"]; !ok {
		t.Error("Expected server 'api' in parsed JSON")
	}
}

func TestMCPConfigBuilder_WriteFile(t *testing.T) {
	builder := NewMCPConfigBuilder()
	builder.AddHTTPServer("api", "https://api.example.com")

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "mcp-config.json")

	if err := builder.WriteFile(filePath); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	// Read and verify
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	var parsed MCPConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse written file: %v", err)
	}

	if _, ok := parsed.MCPServers["api"]; !ok {
		t.Error("Expected server 'api' in written file")
	}
}

func TestParseMCPConfig(t *testing.T) {
	jsonStr := `{
		"mcpServers": {
			"api": {
				"url": "https://api.example.com",
				"type": "http"
			},
			"local": {
				"command": "python",
				"args": ["-m", "server"],
				"type": "stdio"
			}
		}
	}`

	config, err := ParseMCPConfig(jsonStr)
	if err != nil {
		t.Fatalf("ParseMCPConfig() error: %v", err)
	}

	if len(config.MCPServers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(config.MCPServers))
	}

	api := config.MCPServers["api"]
	if api.URL != "https://api.example.com" {
		t.Errorf("Expected URL 'https://api.example.com', got %q", api.URL)
	}

	local := config.MCPServers["local"]
	if local.Command != "python" {
		t.Errorf("Expected command 'python', got %q", local.Command)
	}
}

func TestParseMCPConfig_Invalid(t *testing.T) {
	_, err := ParseMCPConfig("invalid json")
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestLoadMCPConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "config.json")

	content := `{
		"mcpServers": {
			"test": {
				"url": "https://test.example.com",
				"type": "http"
			}
		}
	}`

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config, err := LoadMCPConfigFile(filePath)
	if err != nil {
		t.Fatalf("LoadMCPConfigFile() error: %v", err)
	}

	if _, ok := config.MCPServers["test"]; !ok {
		t.Error("Expected server 'test' in loaded config")
	}
}

func TestLoadMCPConfigFile_NotFound(t *testing.T) {
	_, err := LoadMCPConfigFile("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}
}

func TestMCPConfig_ToJSON(t *testing.T) {
	config := &MCPConfig{
		MCPServers: map[string]MCPServerConfig{
			"api": {
				URL:  "https://api.example.com",
				Type: MCPServerTypeHTTP,
			},
		},
	}

	jsonStr, err := config.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	// Verify round-trip
	parsed, err := ParseMCPConfig(jsonStr)
	if err != nil {
		t.Fatalf("Failed to parse ToJSON output: %v", err)
	}

	if parsed.MCPServers["api"].URL != "https://api.example.com" {
		t.Error("Round-trip failed: URL mismatch")
	}
}

func TestMCPConfig_WriteFile(t *testing.T) {
	config := &MCPConfig{
		MCPServers: map[string]MCPServerConfig{
			"api": {
				URL:  "https://api.example.com",
				Type: MCPServerTypeHTTP,
			},
		},
	}

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "config.json")

	if err := config.WriteFile(filePath); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	loaded, err := LoadMCPConfigFile(filePath)
	if err != nil {
		t.Fatalf("Failed to load written file: %v", err)
	}

	if loaded.MCPServers["api"].URL != "https://api.example.com" {
		t.Error("Written config does not match original")
	}
}

func TestMCPConfig_Merge(t *testing.T) {
	config1 := &MCPConfig{
		MCPServers: map[string]MCPServerConfig{
			"api1":   {URL: "https://api1.example.com", Type: MCPServerTypeHTTP},
			"shared": {URL: "https://shared.v1.com", Type: MCPServerTypeHTTP},
		},
	}

	config2 := &MCPConfig{
		MCPServers: map[string]MCPServerConfig{
			"api2":   {URL: "https://api2.example.com", Type: MCPServerTypeHTTP},
			"shared": {URL: "https://shared.v2.com", Type: MCPServerTypeHTTP}, // Override
		},
	}

	merged := config1.Merge(config2)

	// Check all servers present
	if len(merged.MCPServers) != 3 {
		t.Errorf("Expected 3 servers in merged config, got %d", len(merged.MCPServers))
	}

	// Check api1 preserved
	if merged.MCPServers["api1"].URL != "https://api1.example.com" {
		t.Error("api1 should be preserved from config1")
	}

	// Check api2 added
	if merged.MCPServers["api2"].URL != "https://api2.example.com" {
		t.Error("api2 should be added from config2")
	}

	// Check shared was overridden by config2
	if merged.MCPServers["shared"].URL != "https://shared.v2.com" {
		t.Errorf("shared should be overridden by config2, got %q", merged.MCPServers["shared"].URL)
	}
}

func TestMCPConfig_Merge_NilOther(t *testing.T) {
	config := &MCPConfig{
		MCPServers: map[string]MCPServerConfig{
			"api": {URL: "https://api.example.com", Type: MCPServerTypeHTTP},
		},
	}

	merged := config.Merge(nil)

	// Should return same config
	if merged != config {
		t.Error("Merge(nil) should return the same config")
	}
}

func TestMCPConfig_ServerNames(t *testing.T) {
	config := &MCPConfig{
		MCPServers: map[string]MCPServerConfig{
			"api":    {URL: "https://api.example.com", Type: MCPServerTypeHTTP},
			"events": {URL: "https://sse.example.com", Type: MCPServerTypeSSE},
			"local":  {Command: "python", Type: MCPServerTypeStdio},
		},
	}

	names := config.ServerNames()

	if len(names) != 3 {
		t.Errorf("Expected 3 names, got %d", len(names))
	}

	// Check all names present (order may vary)
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	for _, expected := range []string{"api", "events", "local"} {
		if !nameMap[expected] {
			t.Errorf("Expected name %q not found", expected)
		}
	}
}

func TestMCPConfig_GetServer(t *testing.T) {
	config := &MCPConfig{
		MCPServers: map[string]MCPServerConfig{
			"api": {URL: "https://api.example.com", Type: MCPServerTypeHTTP},
		},
	}

	// Existing server
	server, ok := config.GetServer("api")
	if !ok {
		t.Fatal("Expected to find server 'api'")
	}
	if server.URL != "https://api.example.com" {
		t.Errorf("Expected URL 'https://api.example.com', got %q", server.URL)
	}

	// Non-existing server
	_, ok = config.GetServer("nonexistent")
	if ok {
		t.Error("Expected not to find 'nonexistent' server")
	}
}

func TestMCPConfigBuilder_Build_Immutability(t *testing.T) {
	builder := NewMCPConfigBuilder()
	builder.AddHTTPServer("api", "https://api.example.com")

	config1 := builder.Build()
	config2 := builder.Build()

	// Modify config1
	config1.MCPServers["api"] = MCPServerConfig{URL: "https://modified.com"}

	// config2 should not be affected
	if config2.MCPServers["api"].URL != "https://api.example.com" {
		t.Error("Build() should return independent copies")
	}
}

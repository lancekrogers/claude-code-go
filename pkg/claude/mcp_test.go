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

// Additional error path tests for MCP configuration

func TestParseMCPConfig_InvalidJSONVariants(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"array instead of object", "[]"},
		{"unclosed brace", "{"},
		{"invalid unicode", `{"key": "\uXXXX"}`},
		{"trailing comma", `{"mcpServers": {},"}`},
		{"single quotes", "{'mcpServers': {}}"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseMCPConfig(tc.input)
			if err == nil {
				t.Errorf("Expected error for %q, got nil", tc.name)
			}
		})
	}
}

func TestParseMCPConfig_NullJSON(t *testing.T) {
	// "null" is valid JSON that unmarshals to zero value struct
	// This tests that the function doesn't error but returns an empty config
	config, err := ParseMCPConfig("null")
	if err != nil {
		t.Fatalf("Unexpected error for null JSON: %v", err)
	}
	// Should have nil or empty MCPServers
	if len(config.MCPServers) != 0 {
		t.Errorf("Expected 0 servers for null JSON, got %d", len(config.MCPServers))
	}
}

func TestLoadMCPConfigFile_Errors(t *testing.T) {
	t.Run("directory instead of file", func(t *testing.T) {
		tempDir := t.TempDir()
		_, err := LoadMCPConfigFile(tempDir)
		if err == nil {
			t.Error("Expected error when loading a directory")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		tempDir := t.TempDir()
		emptyFile := filepath.Join(tempDir, "empty.json")
		if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}

		_, err := LoadMCPConfigFile(emptyFile)
		if err == nil {
			t.Error("Expected error for empty file")
		}
	})

	t.Run("invalid JSON in file", func(t *testing.T) {
		tempDir := t.TempDir()
		badFile := filepath.Join(tempDir, "bad.json")
		if err := os.WriteFile(badFile, []byte("not valid json"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		_, err := LoadMCPConfigFile(badFile)
		if err == nil {
			t.Error("Expected error for invalid JSON file")
		}
	})
}

func TestMCPConfig_Merge_EdgeCases(t *testing.T) {
	t.Run("merge with empty config", func(t *testing.T) {
		config := &MCPConfig{
			MCPServers: map[string]MCPServerConfig{
				"api": {URL: "https://api.example.com", Type: MCPServerTypeHTTP},
			},
		}
		empty := &MCPConfig{
			MCPServers: map[string]MCPServerConfig{},
		}

		merged := config.Merge(empty)
		if len(merged.MCPServers) != 1 {
			t.Errorf("Expected 1 server, got %d", len(merged.MCPServers))
		}
	})

	t.Run("merge empty with populated", func(t *testing.T) {
		empty := &MCPConfig{
			MCPServers: map[string]MCPServerConfig{},
		}
		populated := &MCPConfig{
			MCPServers: map[string]MCPServerConfig{
				"api": {URL: "https://api.example.com", Type: MCPServerTypeHTTP},
			},
		}

		merged := empty.Merge(populated)
		if len(merged.MCPServers) != 1 {
			t.Errorf("Expected 1 server, got %d", len(merged.MCPServers))
		}
	})

	t.Run("merge nil MCPServers map", func(t *testing.T) {
		config := &MCPConfig{
			MCPServers: map[string]MCPServerConfig{
				"api": {URL: "https://api.example.com", Type: MCPServerTypeHTTP},
			},
		}
		nilServers := &MCPConfig{
			MCPServers: nil,
		}

		merged := config.Merge(nilServers)
		if len(merged.MCPServers) != 1 {
			t.Errorf("Expected 1 server, got %d", len(merged.MCPServers))
		}
	})

	t.Run("merge overrides all fields", func(t *testing.T) {
		config1 := &MCPConfig{
			MCPServers: map[string]MCPServerConfig{
				"server": {
					Command: "python",
					Args:    []string{"old.py"},
					Env:     map[string]string{"OLD": "value"},
					Type:    MCPServerTypeStdio,
				},
			},
		}
		config2 := &MCPConfig{
			MCPServers: map[string]MCPServerConfig{
				"server": {
					Command: "node",
					Args:    []string{"new.js"},
					Env:     map[string]string{"NEW": "value"},
					Type:    MCPServerTypeStdio,
				},
			},
		}

		merged := config1.Merge(config2)
		server := merged.MCPServers["server"]

		if server.Command != "node" {
			t.Errorf("Expected command 'node', got %q", server.Command)
		}
		if len(server.Args) != 1 || server.Args[0] != "new.js" {
			t.Errorf("Expected args ['new.js'], got %v", server.Args)
		}
		if server.Env["NEW"] != "value" {
			t.Error("Expected NEW env var")
		}
		if _, hasOld := server.Env["OLD"]; hasOld {
			t.Error("OLD env var should not be present")
		}
	})
}

func TestMCPConfig_GetServer_EdgeCases(t *testing.T) {
	t.Run("nil MCPServers map", func(t *testing.T) {
		config := &MCPConfig{MCPServers: nil}
		_, ok := config.GetServer("any")
		if ok {
			t.Error("Expected false for nil MCPServers")
		}
	})

	t.Run("empty string key", func(t *testing.T) {
		config := &MCPConfig{
			MCPServers: map[string]MCPServerConfig{
				"": {URL: "https://empty-key.example.com", Type: MCPServerTypeHTTP},
			},
		}
		server, ok := config.GetServer("")
		if !ok {
			t.Error("Expected to find empty string key")
		}
		if server.URL != "https://empty-key.example.com" {
			t.Errorf("Unexpected URL: %q", server.URL)
		}
	})
}

func TestMCPConfig_ServerNames_EdgeCases(t *testing.T) {
	t.Run("nil MCPServers map", func(t *testing.T) {
		config := &MCPConfig{MCPServers: nil}
		names := config.ServerNames()
		if len(names) != 0 {
			t.Errorf("Expected 0 names for nil MCPServers, got %d", len(names))
		}
	})

	t.Run("empty MCPServers map", func(t *testing.T) {
		config := &MCPConfig{MCPServers: map[string]MCPServerConfig{}}
		names := config.ServerNames()
		if len(names) != 0 {
			t.Errorf("Expected 0 names for empty MCPServers, got %d", len(names))
		}
	})
}

func TestMCPConfigBuilder_WriteFile_Errors(t *testing.T) {
	t.Run("write to non-existent directory", func(t *testing.T) {
		builder := NewMCPConfigBuilder()
		builder.AddHTTPServer("api", "https://api.example.com")

		err := builder.WriteFile("/nonexistent/dir/config.json")
		if err == nil {
			t.Error("Expected error when writing to non-existent directory")
		}
	})
}

func TestMCPConfig_WriteFile_Errors(t *testing.T) {
	t.Run("write to non-existent directory", func(t *testing.T) {
		config := &MCPConfig{
			MCPServers: map[string]MCPServerConfig{
				"api": {URL: "https://api.example.com", Type: MCPServerTypeHTTP},
			},
		}

		err := config.WriteFile("/nonexistent/dir/config.json")
		if err == nil {
			t.Error("Expected error when writing to non-existent directory")
		}
	})
}

func TestMCPServerTypes(t *testing.T) {
	// Verify server type constants are properly defined
	types := []MCPServerType{
		MCPServerTypeHTTP,
		MCPServerTypeSSE,
		MCPServerTypeStdio,
	}

	seen := make(map[MCPServerType]bool)
	for _, serverType := range types {
		if serverType == "" {
			t.Error("Server type should not be empty")
		}
		if seen[serverType] {
			t.Errorf("Duplicate server type: %q", serverType)
		}
		seen[serverType] = true
	}
}

func TestMCPConfig_ToJSON_EdgeCases(t *testing.T) {
	t.Run("nil MCPServers", func(t *testing.T) {
		config := &MCPConfig{MCPServers: nil}
		jsonStr, err := config.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON() error: %v", err)
		}

		// Should still produce valid JSON
		var parsed MCPConfig
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			t.Errorf("ToJSON output is not valid JSON: %v", err)
		}
	})

	t.Run("empty MCPServers", func(t *testing.T) {
		config := &MCPConfig{MCPServers: map[string]MCPServerConfig{}}
		jsonStr, err := config.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON() error: %v", err)
		}

		var parsed MCPConfig
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			t.Errorf("ToJSON output is not valid JSON: %v", err)
		}
		if len(parsed.MCPServers) != 0 {
			t.Error("Expected empty MCPServers in parsed output")
		}
	})

	t.Run("server with special characters in name", func(t *testing.T) {
		config := &MCPConfig{
			MCPServers: map[string]MCPServerConfig{
				"server with spaces": {URL: "https://example.com", Type: MCPServerTypeHTTP},
				"server\twith\ttabs":  {URL: "https://example2.com", Type: MCPServerTypeHTTP},
			},
		}

		jsonStr, err := config.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON() error: %v", err)
		}

		// Should still parse correctly
		var parsed MCPConfig
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			t.Errorf("ToJSON output is not valid JSON: %v", err)
		}
	})
}

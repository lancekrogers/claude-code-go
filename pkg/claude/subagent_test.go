package claude

import (
	"context"
	"sync"
	"testing"
)

func TestSubagentConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *SubagentConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &SubagentConfig{
				Description: "A test agent",
				Prompt:      "You are a test agent",
				Tools:       []string{"Read", "Grep"},
				Model:       "sonnet",
			},
			wantErr: false,
		},
		{
			name: "valid config without model",
			config: &SubagentConfig{
				Description: "A test agent",
				Prompt:      "You are a test agent",
			},
			wantErr: false,
		},
		{
			name: "missing description",
			config: &SubagentConfig{
				Prompt: "You are a test agent",
			},
			wantErr: true,
			errMsg:  "description is required",
		},
		{
			name: "missing prompt",
			config: &SubagentConfig{
				Description: "A test agent",
			},
			wantErr: true,
			errMsg:  "prompt is required",
		},
		{
			name: "invalid model alias",
			config: &SubagentConfig{
				Description: "A test agent",
				Prompt:      "You are a test agent",
				Model:       "invalid-model",
			},
			wantErr: true,
			errMsg:  "invalid model alias",
		},
		{
			name: "valid MCP tool",
			config: &SubagentConfig{
				Description: "A test agent",
				Prompt:      "You are a test agent",
				Tools:       []string{"mcp__server__tool"},
			},
			wantErr: false,
		},
		{
			name: "invalid MCP tool format",
			config: &SubagentConfig{
				Description: "A test agent",
				Prompt:      "You are a test agent",
				Tools:       []string{"mcp__invalid"},
			},
			wantErr: true,
			errMsg:  "invalid MCP tool name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !containsSubstring(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestSubagentConfig_ToRunOptions(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		config := &SubagentConfig{
			Description: "Test agent",
			Prompt:      "You are a test agent",
			Tools:       []string{"Read", "Grep"},
			Model:       "haiku",
			MaxTurns:    5,
		}

		opts := config.ToRunOptions(nil)

		if opts.SystemPrompt != config.Prompt {
			t.Errorf("SystemPrompt = %q, want %q", opts.SystemPrompt, config.Prompt)
		}
		if len(opts.AllowedTools) != len(config.Tools) {
			t.Errorf("AllowedTools length = %d, want %d", len(opts.AllowedTools), len(config.Tools))
		}
		if opts.ModelAlias != "haiku" {
			t.Errorf("ModelAlias = %q, want %q", opts.ModelAlias, "haiku")
		}
		if opts.MaxTurns != 5 {
			t.Errorf("MaxTurns = %d, want %d", opts.MaxTurns, 5)
		}
		if opts.Format != StreamJSONOutput {
			t.Errorf("Format = %q, want %q", opts.Format, StreamJSONOutput)
		}
	})

	t.Run("inherits from parent", func(t *testing.T) {
		config := &SubagentConfig{
			Description: "Test agent",
			Prompt:      "You are a test agent",
		}

		parentOpts := &RunOptions{
			ModelAlias:    "opus",
			MaxTurns:      10,
			MCPConfigPath: "/path/to/mcp.json",
		}

		opts := config.ToRunOptions(parentOpts)

		if opts.ModelAlias != "opus" {
			t.Errorf("ModelAlias = %q, want inherited %q", opts.ModelAlias, "opus")
		}
		if opts.MaxTurns != 10 {
			t.Errorf("MaxTurns = %d, want inherited %d", opts.MaxTurns, 10)
		}
		if opts.MCPConfigPath != "/path/to/mcp.json" {
			t.Errorf("MCPConfigPath = %q, want inherited %q", opts.MCPConfigPath, "/path/to/mcp.json")
		}
	})

	t.Run("subagent overrides parent", func(t *testing.T) {
		config := &SubagentConfig{
			Description: "Test agent",
			Prompt:      "You are a test agent",
			Model:       "haiku",
			MaxTurns:    3,
		}

		parentOpts := &RunOptions{
			ModelAlias: "opus",
			MaxTurns:   10,
		}

		opts := config.ToRunOptions(parentOpts)

		if opts.ModelAlias != "haiku" {
			t.Errorf("ModelAlias = %q, want subagent's %q", opts.ModelAlias, "haiku")
		}
		if opts.MaxTurns != 3 {
			t.Errorf("MaxTurns = %d, want subagent's %d", opts.MaxTurns, 3)
		}
	})
}

func TestNewSubagentManager(t *testing.T) {
	client := NewClient("mock-claude")
	manager := NewSubagentManager(client)

	if manager == nil {
		t.Fatal("NewSubagentManager() returned nil")
	}
	if manager.client != client {
		t.Error("client not set correctly")
	}
	if manager.agents == nil {
		t.Error("agents map is nil")
	}
	if manager.sessions == nil {
		t.Error("sessions map is nil")
	}
}

func TestSubagentManager_RegisterAgent(t *testing.T) {
	client := NewClient("mock-claude")
	manager := NewSubagentManager(client)

	t.Run("successful registration", func(t *testing.T) {
		config := &SubagentConfig{
			Description: "Test agent",
			Prompt:      "You are a test agent",
		}

		err := manager.RegisterAgent("test-agent", config)
		if err != nil {
			t.Fatalf("RegisterAgent() error = %v", err)
		}

		retrieved, ok := manager.GetAgent("test-agent")
		if !ok {
			t.Error("GetAgent() returned false for registered agent")
		}
		if retrieved != config {
			t.Error("GetAgent() returned different config")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		config := &SubagentConfig{
			Description: "Test agent",
			Prompt:      "You are a test agent",
		}

		err := manager.RegisterAgent("", config)
		if err == nil {
			t.Error("RegisterAgent() should fail with empty name")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		err := manager.RegisterAgent("nil-agent", nil)
		if err == nil {
			t.Error("RegisterAgent() should fail with nil config")
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		config := &SubagentConfig{
			Description: "", // Missing required field
			Prompt:      "You are a test agent",
		}

		err := manager.RegisterAgent("invalid-agent", config)
		if err == nil {
			t.Error("RegisterAgent() should fail with invalid config")
		}
	})
}

func TestSubagentManager_RegisterAgents(t *testing.T) {
	client := NewClient("mock-claude")
	manager := NewSubagentManager(client)

	agents := map[string]*SubagentConfig{
		"agent1": {
			Description: "Agent 1",
			Prompt:      "You are agent 1",
		},
		"agent2": {
			Description: "Agent 2",
			Prompt:      "You are agent 2",
			Model:       "haiku",
		},
	}

	err := manager.RegisterAgents(agents)
	if err != nil {
		t.Fatalf("RegisterAgents() error = %v", err)
	}

	if manager.AgentCount() != 2 {
		t.Errorf("AgentCount() = %d, want 2", manager.AgentCount())
	}
}

func TestSubagentManager_UnregisterAgent(t *testing.T) {
	client := NewClient("mock-claude")
	manager := NewSubagentManager(client)

	config := &SubagentConfig{
		Description: "Test agent",
		Prompt:      "You are a test agent",
	}
	_ = manager.RegisterAgent("test-agent", config)
	manager.SetSession("test-agent", "session-123")

	manager.UnregisterAgent("test-agent")

	_, ok := manager.GetAgent("test-agent")
	if ok {
		t.Error("GetAgent() should return false after unregister")
	}

	_, sessionOk := manager.GetSession("test-agent")
	if sessionOk {
		t.Error("GetSession() should return false after unregister")
	}
}

func TestSubagentManager_ListAgents(t *testing.T) {
	client := NewClient("mock-claude")
	manager := NewSubagentManager(client)

	agents := map[string]*SubagentConfig{
		"security": {Description: "Security", Prompt: "Security agent"},
		"test":     {Description: "Test", Prompt: "Test agent"},
		"docs":     {Description: "Docs", Prompt: "Docs agent"},
	}
	_ = manager.RegisterAgents(agents)

	list := manager.ListAgents()
	if len(list) != 3 {
		t.Errorf("ListAgents() returned %d items, want 3", len(list))
	}

	// Check all agents are in the list
	agentMap := make(map[string]bool)
	for _, name := range list {
		agentMap[name] = true
	}
	for name := range agents {
		if !agentMap[name] {
			t.Errorf("ListAgents() missing agent %q", name)
		}
	}
}

func TestSubagentManager_GetAgentDescriptions(t *testing.T) {
	client := NewClient("mock-claude")
	manager := NewSubagentManager(client)

	agents := map[string]*SubagentConfig{
		"security": {Description: "Security expert", Prompt: "You are a security expert"},
		"test":     {Description: "Test analyst", Prompt: "You are a test analyst"},
	}
	_ = manager.RegisterAgents(agents)

	descriptions := manager.GetAgentDescriptions()

	if descriptions["security"] != "Security expert" {
		t.Errorf("Description for 'security' = %q, want %q", descriptions["security"], "Security expert")
	}
	if descriptions["test"] != "Test analyst" {
		t.Errorf("Description for 'test' = %q, want %q", descriptions["test"], "Test analyst")
	}
}

func TestSubagentManager_Sessions(t *testing.T) {
	client := NewClient("mock-claude")
	manager := NewSubagentManager(client)

	t.Run("set and get session", func(t *testing.T) {
		manager.SetSession("agent1", "session-abc")
		sessionID, ok := manager.GetSession("agent1")
		if !ok {
			t.Error("GetSession() returned false for set session")
		}
		if sessionID != "session-abc" {
			t.Errorf("GetSession() = %q, want %q", sessionID, "session-abc")
		}
	})

	t.Run("get non-existent session", func(t *testing.T) {
		_, ok := manager.GetSession("non-existent")
		if ok {
			t.Error("GetSession() should return false for non-existent session")
		}
	})

	t.Run("clear session", func(t *testing.T) {
		manager.SetSession("agent2", "session-xyz")
		manager.ClearSession("agent2")
		_, ok := manager.GetSession("agent2")
		if ok {
			t.Error("GetSession() should return false after clear")
		}
	})

	t.Run("clear all sessions", func(t *testing.T) {
		manager.SetSession("agent3", "session-1")
		manager.SetSession("agent4", "session-2")
		manager.ClearAllSessions()

		_, ok1 := manager.GetSession("agent3")
		_, ok2 := manager.GetSession("agent4")
		if ok1 || ok2 {
			t.Error("ClearAllSessions() should clear all sessions")
		}
	})
}

func TestSubagentManager_StreamAgent_UnknownAgent(t *testing.T) {
	client := NewClient("mock-claude")
	manager := NewSubagentManager(client)

	msgCh, errCh := manager.StreamAgent(context.Background(), "unknown", "test prompt", nil)

	// Drain message channel
	for range msgCh {
		t.Error("Should not receive any messages for unknown agent")
	}

	// Check error
	select {
	case err := <-errCh:
		if err == nil {
			t.Error("Expected error for unknown agent")
		}
		if !containsSubstring(err.Error(), "unknown agent") {
			t.Errorf("Error should mention 'unknown agent', got: %v", err)
		}
	default:
		t.Error("Expected error in error channel")
	}
}

func TestSubagentManager_RunAgent_UnknownAgent(t *testing.T) {
	client := NewClient("mock-claude")
	manager := NewSubagentManager(client)

	_, err := manager.RunAgent(context.Background(), "unknown", "test prompt", nil)
	if err == nil {
		t.Error("RunAgent() should fail for unknown agent")
	}
	if !containsSubstring(err.Error(), "unknown agent") {
		t.Errorf("Error should mention 'unknown agent', got: %v", err)
	}
}

func TestSubagentManager_ResumeAgent(t *testing.T) {
	client := NewClient("mock-claude")
	manager := NewSubagentManager(client)

	t.Run("no session", func(t *testing.T) {
		config := &SubagentConfig{
			Description: "Test agent",
			Prompt:      "You are a test agent",
		}
		_ = manager.RegisterAgent("test-agent", config)

		_, err := manager.ResumeAgent(context.Background(), "test-agent", "test prompt", nil)
		if err == nil {
			t.Error("ResumeAgent() should fail without session")
		}
		if !containsSubstring(err.Error(), "no session found") {
			t.Errorf("Error should mention 'no session found', got: %v", err)
		}
	})

	t.Run("unknown agent", func(t *testing.T) {
		manager.SetSession("unknown-agent", "session-123")
		_, err := manager.ResumeAgent(context.Background(), "unknown-agent", "test prompt", nil)
		if err == nil {
			t.Error("ResumeAgent() should fail for unknown agent")
		}
	})
}

func TestSubagentManager_Concurrent(t *testing.T) {
	client := NewClient("mock-claude")
	manager := NewSubagentManager(client)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			config := &SubagentConfig{
				Description: "Concurrent test agent",
				Prompt:      "You are a concurrent test agent",
			}
			_ = manager.RegisterAgent("concurrent-agent", config)
			manager.SetSession("concurrent-agent", "session")
			_, _ = manager.GetAgent("concurrent-agent")
			_, _ = manager.GetSession("concurrent-agent")
			_ = manager.ListAgents()
			_ = manager.GetAgentDescriptions()
		}(i)
	}
	wg.Wait()

	// Should not panic and manager should still work
	if manager.AgentCount() == 0 {
		t.Error("Manager should have at least one agent after concurrent operations")
	}
}

// Test pre-built agent configurations
func TestPreBuiltAgents(t *testing.T) {
	preBuiltAgents := map[string]func() *SubagentConfig{
		"SecurityReviewer":   SecurityReviewerAgent,
		"CodeReviewer":       CodeReviewerAgent,
		"TestAnalyst":        TestAnalystAgent,
		"PerformanceAnalyst": PerformanceAnalystAgent,
		"DocumentationAgent": DocumentationAgent,
	}

	for name, agentFunc := range preBuiltAgents {
		t.Run(name, func(t *testing.T) {
			agent := agentFunc()
			if agent == nil {
				t.Fatalf("%s() returned nil", name)
			}
			if err := agent.Validate(); err != nil {
				t.Errorf("%s() validation failed: %v", name, err)
			}
			if agent.Description == "" {
				t.Errorf("%s() has empty description", name)
			}
			if agent.Prompt == "" {
				t.Errorf("%s() has empty prompt", name)
			}
		})
	}
}

func TestRunOptions_SubagentValidation(t *testing.T) {
	t.Run("valid subagents", func(t *testing.T) {
		opts := &RunOptions{
			Agents: map[string]*SubagentConfig{
				"security": SecurityReviewerAgent(),
				"test":     TestAnalystAgent(),
			},
		}
		err := PreprocessOptions(opts)
		if err != nil {
			t.Errorf("PreprocessOptions() error = %v for valid subagents", err)
		}
	})

	t.Run("nil subagent config", func(t *testing.T) {
		opts := &RunOptions{
			Agents: map[string]*SubagentConfig{
				"valid": SecurityReviewerAgent(),
				"nil":   nil,
			},
		}
		err := PreprocessOptions(opts)
		if err == nil {
			t.Error("PreprocessOptions() should fail for nil subagent config")
		}
	})

	t.Run("invalid subagent config", func(t *testing.T) {
		opts := &RunOptions{
			Agents: map[string]*SubagentConfig{
				"invalid": {
					Description: "", // Missing required field
					Prompt:      "Test",
				},
			},
		}
		err := PreprocessOptions(opts)
		if err == nil {
			t.Error("PreprocessOptions() should fail for invalid subagent config")
		}
	})
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

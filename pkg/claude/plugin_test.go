package claude

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockPlugin is a test implementation of the Plugin interface
type mockPlugin struct {
	name          string
	version       string
	initErr       error
	toolCallErr   error
	messageErr    error
	completeErr   error
	shutdownErr   error
	initCalled    int
	toolCalls     []string
	messages      []Message
	results       []*ClaudeResult
	shutdownCount int
	mu            sync.Mutex
}

func newMockPlugin(name, version string) *mockPlugin {
	return &mockPlugin{
		name:      name,
		version:   version,
		toolCalls: make([]string, 0),
		messages:  make([]Message, 0),
		results:   make([]*ClaudeResult, 0),
	}
}

func (mp *mockPlugin) Name() string    { return mp.name }
func (mp *mockPlugin) Version() string { return mp.version }

func (mp *mockPlugin) Initialize(ctx context.Context) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.initCalled++
	return mp.initErr
}

func (mp *mockPlugin) OnToolCall(ctx context.Context, toolName string, input ToolInput) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.toolCalls = append(mp.toolCalls, toolName)
	return mp.toolCallErr
}

func (mp *mockPlugin) OnMessage(ctx context.Context, msg Message) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.messages = append(mp.messages, msg)
	return mp.messageErr
}

func (mp *mockPlugin) OnComplete(ctx context.Context, result *ClaudeResult) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.results = append(mp.results, result)
	return mp.completeErr
}

func (mp *mockPlugin) Shutdown(ctx context.Context) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.shutdownCount++
	return mp.shutdownErr
}

func TestPluginManagerRegister(t *testing.T) {
	pm := NewPluginManager()

	t.Run("register valid plugin", func(t *testing.T) {
		plugin := newMockPlugin("test-plugin", "1.0.0")
		err := pm.Register(plugin, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if pm.Count() != 1 {
			t.Errorf("expected 1 plugin, got %d", pm.Count())
		}
	})

	t.Run("register nil plugin", func(t *testing.T) {
		err := pm.Register(nil, nil)
		if err == nil {
			t.Error("expected error for nil plugin")
		}
	})

	t.Run("register duplicate plugin", func(t *testing.T) {
		plugin := newMockPlugin("test-plugin", "1.0.0")
		err := pm.Register(plugin, nil)
		if err == nil {
			t.Error("expected error for duplicate plugin name")
		}
	})

	t.Run("register plugin with empty name", func(t *testing.T) {
		plugin := newMockPlugin("", "1.0.0")
		pm2 := NewPluginManager()
		err := pm2.Register(plugin, nil)
		if err == nil {
			t.Error("expected error for empty plugin name")
		}
	})
}

func TestPluginManagerPriority(t *testing.T) {
	pm := NewPluginManager()

	plugin1 := newMockPlugin("high-priority", "1.0.0")
	plugin2 := newMockPlugin("low-priority", "1.0.0")
	plugin3 := newMockPlugin("medium-priority", "1.0.0")

	_ = pm.Register(plugin2, &PluginConfig{Enabled: true, Priority: 200})
	_ = pm.Register(plugin1, &PluginConfig{Enabled: true, Priority: 50})
	_ = pm.Register(plugin3, &PluginConfig{Enabled: true, Priority: 100})

	names := pm.List()
	expected := []string{"high-priority", "medium-priority", "low-priority"}

	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected plugin at index %d to be %s, got %s", i, expected[i], name)
		}
	}
}

func TestPluginManagerUnregister(t *testing.T) {
	pm := NewPluginManager()
	plugin := newMockPlugin("test-plugin", "1.0.0")
	_ = pm.Register(plugin, nil)

	t.Run("unregister existing plugin", func(t *testing.T) {
		err := pm.Unregister("test-plugin")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if pm.Count() != 0 {
			t.Errorf("expected 0 plugins, got %d", pm.Count())
		}
	})

	t.Run("unregister non-existent plugin", func(t *testing.T) {
		err := pm.Unregister("non-existent")
		if err == nil {
			t.Error("expected error for non-existent plugin")
		}
	})
}

func TestPluginManagerInitialize(t *testing.T) {
	ctx := context.Background()

	t.Run("initialize all plugins", func(t *testing.T) {
		pm := NewPluginManager()
		plugin1 := newMockPlugin("plugin1", "1.0.0")
		plugin2 := newMockPlugin("plugin2", "1.0.0")

		_ = pm.Register(plugin1, nil)
		_ = pm.Register(plugin2, nil)

		err := pm.Initialize(ctx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if plugin1.initCalled != 1 {
			t.Errorf("expected plugin1 init to be called once, got %d", plugin1.initCalled)
		}
		if plugin2.initCalled != 1 {
			t.Errorf("expected plugin2 init to be called once, got %d", plugin2.initCalled)
		}
	})

	t.Run("initialize is idempotent", func(t *testing.T) {
		pm := NewPluginManager()
		plugin := newMockPlugin("plugin", "1.0.0")
		_ = pm.Register(plugin, nil)

		_ = pm.Initialize(ctx)
		_ = pm.Initialize(ctx) // Second call should be no-op

		if plugin.initCalled != 1 {
			t.Errorf("expected init to be called once, got %d", plugin.initCalled)
		}
	})

	t.Run("skip disabled plugins", func(t *testing.T) {
		pm := NewPluginManager()
		plugin := newMockPlugin("disabled-plugin", "1.0.0")
		_ = pm.Register(plugin, &PluginConfig{Enabled: false})

		_ = pm.Initialize(ctx)

		if plugin.initCalled != 0 {
			t.Errorf("expected disabled plugin not to be initialized, got %d calls", plugin.initCalled)
		}
	})

	t.Run("initialization error", func(t *testing.T) {
		pm := NewPluginManager()
		plugin := newMockPlugin("failing-plugin", "1.0.0")
		plugin.initErr = errors.New("init failed")
		_ = pm.Register(plugin, nil)

		err := pm.Initialize(ctx)
		if err == nil {
			t.Error("expected error from failing initialization")
		}
	})
}

func TestPluginManagerOnToolCall(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginManager()
	plugin := newMockPlugin("test-plugin", "1.0.0")
	_ = pm.Register(plugin, nil)

	t.Run("tool call propagation", func(t *testing.T) {
		input := ToolInput{Command: "ls -la"}
		err := pm.OnToolCall(ctx, "Bash", input)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if len(plugin.toolCalls) != 1 || plugin.toolCalls[0] != "Bash" {
			t.Errorf("expected Bash tool call, got %v", plugin.toolCalls)
		}
	})

	t.Run("tool call error blocks execution", func(t *testing.T) {
		pm2 := NewPluginManager()
		blockingPlugin := newMockPlugin("blocking", "1.0.0")
		blockingPlugin.toolCallErr = errors.New("blocked")
		_ = pm2.Register(blockingPlugin, nil)

		err := pm2.OnToolCall(ctx, "Write", ToolInput{})
		if err == nil {
			t.Error("expected error from blocking plugin")
		}
	})
}

func TestPluginManagerOnMessage(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginManager()
	plugin := newMockPlugin("test-plugin", "1.0.0")
	_ = pm.Register(plugin, nil)

	msg := Message{Type: "assistant", Subtype: "text"}
	err := pm.OnMessage(ctx, msg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(plugin.messages) != 1 || plugin.messages[0].Type != "assistant" {
		t.Errorf("expected assistant message, got %v", plugin.messages)
	}
}

func TestPluginManagerOnComplete(t *testing.T) {
	ctx := context.Background()
	pm := NewPluginManager()
	plugin := newMockPlugin("test-plugin", "1.0.0")
	_ = pm.Register(plugin, nil)

	result := &ClaudeResult{CostUSD: 0.05, NumTurns: 3}
	err := pm.OnComplete(ctx, result)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(plugin.results) != 1 || plugin.results[0].CostUSD != 0.05 {
		t.Errorf("expected result with cost 0.05, got %v", plugin.results)
	}
}

func TestPluginManagerShutdown(t *testing.T) {
	ctx := context.Background()

	t.Run("shutdown all plugins", func(t *testing.T) {
		pm := NewPluginManager()
		plugin1 := newMockPlugin("plugin1", "1.0.0")
		plugin2 := newMockPlugin("plugin2", "1.0.0")

		_ = pm.Register(plugin1, nil)
		_ = pm.Register(plugin2, nil)

		err := pm.Shutdown(ctx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if plugin1.shutdownCount != 1 || plugin2.shutdownCount != 1 {
			t.Error("expected both plugins to be shut down")
		}
	})

	t.Run("shutdown error is returned", func(t *testing.T) {
		pm := NewPluginManager()
		plugin := newMockPlugin("failing", "1.0.0")
		plugin.shutdownErr = errors.New("shutdown failed")
		_ = pm.Register(plugin, nil)

		err := pm.Shutdown(ctx)
		if err == nil {
			t.Error("expected shutdown error")
		}
	})
}

func TestPluginManagerGet(t *testing.T) {
	pm := NewPluginManager()
	plugin := newMockPlugin("test-plugin", "1.0.0")
	_ = pm.Register(plugin, nil)

	t.Run("get existing plugin", func(t *testing.T) {
		p, found := pm.Get("test-plugin")
		if !found {
			t.Error("expected to find plugin")
		}
		if p.Name() != "test-plugin" {
			t.Errorf("expected test-plugin, got %s", p.Name())
		}
	})

	t.Run("get non-existent plugin", func(t *testing.T) {
		_, found := pm.Get("non-existent")
		if found {
			t.Error("expected not to find non-existent plugin")
		}
	})
}

func TestPluginManagerSetEnabled(t *testing.T) {
	pm := NewPluginManager()
	plugin := newMockPlugin("test-plugin", "1.0.0")
	_ = pm.Register(plugin, &PluginConfig{Enabled: true})

	t.Run("disable plugin", func(t *testing.T) {
		err := pm.SetEnabled("test-plugin", false)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Verify plugin is skipped
		ctx := context.Background()
		_ = pm.OnToolCall(ctx, "Bash", ToolInput{})

		if len(plugin.toolCalls) != 0 {
			t.Error("expected disabled plugin to be skipped")
		}
	})

	t.Run("set enabled on non-existent plugin", func(t *testing.T) {
		err := pm.SetEnabled("non-existent", true)
		if err == nil {
			t.Error("expected error for non-existent plugin")
		}
	})
}

func TestBasePlugin(t *testing.T) {
	bp := &BasePlugin{
		PluginName:    "base",
		PluginVersion: "1.0.0",
	}

	if bp.Name() != "base" {
		t.Errorf("expected name 'base', got %s", bp.Name())
	}
	if bp.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", bp.Version())
	}

	ctx := context.Background()

	// All default methods should return nil
	if err := bp.Initialize(ctx); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := bp.OnToolCall(ctx, "test", ToolInput{}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := bp.OnMessage(ctx, Message{}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := bp.OnComplete(ctx, &ClaudeResult{}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if err := bp.Shutdown(ctx); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestLoggingPlugin(t *testing.T) {
	var logs []string
	logger := func(format string, args ...interface{}) {
		logs = append(logs, format)
	}

	lp := NewLoggingPlugin(logger)

	if lp.Name() != "logging" {
		t.Errorf("expected name 'logging', got %s", lp.Name())
	}

	ctx := context.Background()

	_ = lp.OnToolCall(ctx, "Bash", ToolInput{Command: "ls"})
	_ = lp.OnMessage(ctx, Message{Type: "assistant"})
	_ = lp.OnComplete(ctx, &ClaudeResult{CostUSD: 0.01})

	if len(logs) != 3 {
		t.Errorf("expected 3 log entries, got %d", len(logs))
	}
}

func TestMetricsPlugin(t *testing.T) {
	mp := NewMetricsPlugin()

	if mp.Name() != "metrics" {
		t.Errorf("expected name 'metrics', got %s", mp.Name())
	}

	ctx := context.Background()

	// Simulate some activity
	_ = mp.OnToolCall(ctx, "Bash", ToolInput{})
	_ = mp.OnToolCall(ctx, "Bash", ToolInput{})
	_ = mp.OnToolCall(ctx, "Write", ToolInput{})
	_ = mp.OnMessage(ctx, Message{})
	_ = mp.OnMessage(ctx, Message{})
	_ = mp.OnComplete(ctx, &ClaudeResult{CostUSD: 0.05})

	metrics := mp.GetMetrics()

	toolCalls := metrics["tool_calls"].(map[string]int)
	if toolCalls["Bash"] != 2 {
		t.Errorf("expected 2 Bash calls, got %d", toolCalls["Bash"])
	}
	if toolCalls["Write"] != 1 {
		t.Errorf("expected 1 Write call, got %d", toolCalls["Write"])
	}
	if metrics["message_count"].(int) != 2 {
		t.Errorf("expected 2 messages, got %d", metrics["message_count"])
	}
	if metrics["total_cost"].(float64) != 0.05 {
		t.Errorf("expected cost 0.05, got %f", metrics["total_cost"])
	}

	// Test reset
	mp.Reset()
	metrics = mp.GetMetrics()
	if metrics["message_count"].(int) != 0 {
		t.Error("expected metrics to be reset")
	}
}

func TestToolFilterPlugin(t *testing.T) {
	blockedTools := map[string]string{
		"Bash":  "shell commands blocked",
		"Write": "",
	}
	tfp := NewToolFilterPlugin(blockedTools)

	if tfp.Name() != "tool-filter" {
		t.Errorf("expected name 'tool-filter', got %s", tfp.Name())
	}

	ctx := context.Background()

	t.Run("blocks listed tools", func(t *testing.T) {
		err := tfp.OnToolCall(ctx, "Bash", ToolInput{})
		if err == nil {
			t.Error("expected Bash to be blocked")
		}
	})

	t.Run("allows unlisted tools", func(t *testing.T) {
		err := tfp.OnToolCall(ctx, "Read", ToolInput{})
		if err != nil {
			t.Errorf("expected Read to be allowed, got %v", err)
		}
	})

	t.Run("unblock tool", func(t *testing.T) {
		tfp.UnblockTool("Bash")
		err := tfp.OnToolCall(ctx, "Bash", ToolInput{})
		if err != nil {
			t.Errorf("expected Bash to be unblocked, got %v", err)
		}
	})

	t.Run("block new tool", func(t *testing.T) {
		tfp.BlockTool("Read", "reading not allowed")
		err := tfp.OnToolCall(ctx, "Read", ToolInput{})
		if err == nil {
			t.Error("expected Read to be blocked")
		}
	})
}

func TestAuditPlugin(t *testing.T) {
	// Mock time for consistent testing
	originalTimeNow := timeNow
	timeNow = func() time.Time {
		return time.Unix(1700000000, 0)
	}
	defer func() { timeNow = originalTimeNow }()

	ap := NewAuditPlugin(5)

	if ap.Name() != "audit" {
		t.Errorf("expected name 'audit', got %s", ap.Name())
	}

	ctx := context.Background()

	// Record some tool calls
	for i := 0; i < 7; i++ {
		_ = ap.OnToolCall(ctx, "Bash", ToolInput{Raw: map[string]interface{}{"i": i}})
	}

	records := ap.GetRecords()

	// Should be limited to max 5 records
	if len(records) != 5 {
		t.Errorf("expected 5 records (max size), got %d", len(records))
	}

	// Should keep most recent records (i=2 through i=6)
	firstRecord := records[0].Input["i"].(int)
	if firstRecord != 2 {
		t.Errorf("expected first record to have i=2, got %d", firstRecord)
	}

	// Test clear
	ap.Clear()
	records = ap.GetRecords()
	if len(records) != 0 {
		t.Error("expected records to be cleared")
	}
}

func TestAuditPluginUnlimited(t *testing.T) {
	ap := NewAuditPlugin(0) // 0 = unlimited

	ctx := context.Background()

	for i := 0; i < 100; i++ {
		_ = ap.OnToolCall(ctx, "Bash", ToolInput{})
	}

	records := ap.GetRecords()
	if len(records) != 100 {
		t.Errorf("expected 100 records with unlimited size, got %d", len(records))
	}
}

func TestPluginConcurrency(t *testing.T) {
	pm := NewPluginManager()
	plugin := newMockPlugin("concurrent", "1.0.0")
	_ = pm.Register(plugin, nil)

	ctx := context.Background()
	var wg sync.WaitGroup
	iterations := 100

	// Concurrent tool calls
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = pm.OnToolCall(ctx, "Bash", ToolInput{})
		}()
	}

	// Concurrent message handling
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = pm.OnMessage(ctx, Message{})
		}()
	}

	wg.Wait()

	plugin.mu.Lock()
	defer plugin.mu.Unlock()

	if len(plugin.toolCalls) != iterations {
		t.Errorf("expected %d tool calls, got %d", iterations, len(plugin.toolCalls))
	}
	if len(plugin.messages) != iterations {
		t.Errorf("expected %d messages, got %d", iterations, len(plugin.messages))
	}
}

func TestMetricsPluginConcurrency(t *testing.T) {
	mp := NewMetricsPlugin()
	ctx := context.Background()

	var wg sync.WaitGroup
	iterations := 1000

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = mp.OnToolCall(ctx, "Bash", ToolInput{})
		}()
	}

	wg.Wait()

	metrics := mp.GetMetrics()
	toolCalls := metrics["tool_calls"].(map[string]int)
	if toolCalls["Bash"] != iterations {
		t.Errorf("expected %d Bash calls, got %d", iterations, toolCalls["Bash"])
	}
}

func TestPluginManagerConcurrentRegistration(t *testing.T) {
	pm := NewPluginManager()
	var wg sync.WaitGroup
	var successCount int32

	// Try to register 100 plugins concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			plugin := newMockPlugin("plugin", "1.0.0") // Same name - only one should succeed
			if err := pm.Register(plugin, nil); err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// Only one registration should succeed
	if successCount != 1 {
		t.Errorf("expected exactly 1 successful registration, got %d", successCount)
	}
}

func TestPluginConfigDefaults(t *testing.T) {
	pm := NewPluginManager()
	plugin := newMockPlugin("test", "1.0.0")

	// Register with nil config - should use defaults
	_ = pm.Register(plugin, nil)

	names := pm.List()
	if len(names) != 1 || names[0] != "test" {
		t.Error("plugin should be registered with default config")
	}
}

func TestPluginManagerList(t *testing.T) {
	pm := NewPluginManager()
	_ = pm.Register(newMockPlugin("alpha", "1.0.0"), &PluginConfig{Priority: 100})
	_ = pm.Register(newMockPlugin("beta", "1.0.0"), &PluginConfig{Priority: 50})
	_ = pm.Register(newMockPlugin("gamma", "1.0.0"), &PluginConfig{Priority: 150})

	names := pm.List()

	// Should be in priority order
	expected := []string{"beta", "alpha", "gamma"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected %s at index %d, got %s", expected[i], i, name)
		}
	}
}

func TestRunOptionsPluginManager(t *testing.T) {
	pm := NewPluginManager()
	plugin := newMockPlugin("test", "1.0.0")
	_ = pm.Register(plugin, nil)

	opts := &RunOptions{
		PluginManager: pm,
	}

	if opts.PluginManager == nil {
		t.Error("expected PluginManager to be set")
	}
	if opts.PluginManager.Count() != 1 {
		t.Errorf("expected 1 plugin, got %d", opts.PluginManager.Count())
	}
}

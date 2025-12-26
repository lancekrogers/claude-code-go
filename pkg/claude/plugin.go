package claude

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// timeNow is a variable to allow mocking time in tests
var timeNow = time.Now

// Plugin defines the interface for SDK extensions
// Plugins can hook into the Claude execution lifecycle to add custom behavior
type Plugin interface {
	// Name returns the unique identifier for this plugin
	Name() string
	// Version returns the plugin version string
	Version() string
	// Initialize is called once when the plugin is registered
	Initialize(ctx context.Context) error
	// OnToolCall is called before each tool execution
	// Return an error to abort the tool call
	OnToolCall(ctx context.Context, toolName string, input ToolInput) error
	// OnMessage is called for each message received during streaming
	OnMessage(ctx context.Context, msg Message) error
	// OnComplete is called when execution finishes successfully
	OnComplete(ctx context.Context, result *ClaudeResult) error
	// Shutdown is called when the plugin manager is closed
	Shutdown(ctx context.Context) error
}

// PluginConfig holds configuration options for a plugin
type PluginConfig struct {
	// Enabled controls whether the plugin is active
	Enabled bool `json:"enabled"`
	// Priority determines execution order (lower = earlier, default 100)
	Priority int `json:"priority,omitempty"`
	// Config holds plugin-specific configuration
	Config map[string]interface{} `json:"config,omitempty"`
}

// PluginManager manages the lifecycle and execution of registered plugins
type PluginManager struct {
	mu          sync.RWMutex
	plugins     []pluginEntry
	initialized bool
}

// pluginEntry holds a plugin with its configuration
type pluginEntry struct {
	plugin   Plugin
	config   *PluginConfig
	priority int
}

// NewPluginManager creates a new plugin manager
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make([]pluginEntry, 0),
	}
}

// Register adds a plugin to the manager
// Plugins are executed in order of priority (lower priority values run first)
func (pm *PluginManager) Register(plugin Plugin, config *PluginConfig) error {
	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}
	if plugin.Name() == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check for duplicate names
	for _, entry := range pm.plugins {
		if entry.plugin.Name() == plugin.Name() {
			return fmt.Errorf("plugin with name '%s' already registered", plugin.Name())
		}
	}

	// Default config if not provided
	if config == nil {
		config = &PluginConfig{
			Enabled:  true,
			Priority: 100,
		}
	}

	priority := config.Priority
	if priority == 0 {
		priority = 100
	}

	entry := pluginEntry{
		plugin:   plugin,
		config:   config,
		priority: priority,
	}

	// Insert in priority order
	inserted := false
	for i, existing := range pm.plugins {
		if priority < existing.priority {
			pm.plugins = append(pm.plugins[:i], append([]pluginEntry{entry}, pm.plugins[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		pm.plugins = append(pm.plugins, entry)
	}

	return nil
}

// Unregister removes a plugin by name
func (pm *PluginManager) Unregister(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for i, entry := range pm.plugins {
		if entry.plugin.Name() == name {
			pm.plugins = append(pm.plugins[:i], pm.plugins[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("plugin '%s' not found", name)
}

// Initialize initializes all registered plugins
func (pm *PluginManager) Initialize(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.initialized {
		return nil
	}

	for _, entry := range pm.plugins {
		if entry.config != nil && !entry.config.Enabled {
			continue
		}
		if err := entry.plugin.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize plugin '%s': %w", entry.plugin.Name(), err)
		}
	}

	pm.initialized = true
	return nil
}

// OnToolCall invokes OnToolCall on all enabled plugins
// If any plugin returns an error, execution stops and the error is returned
func (pm *PluginManager) OnToolCall(ctx context.Context, toolName string, input ToolInput) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, entry := range pm.plugins {
		if entry.config != nil && !entry.config.Enabled {
			continue
		}
		if err := entry.plugin.OnToolCall(ctx, toolName, input); err != nil {
			return fmt.Errorf("plugin '%s' rejected tool call: %w", entry.plugin.Name(), err)
		}
	}

	return nil
}

// OnMessage invokes OnMessage on all enabled plugins
func (pm *PluginManager) OnMessage(ctx context.Context, msg Message) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, entry := range pm.plugins {
		if entry.config != nil && !entry.config.Enabled {
			continue
		}
		if err := entry.plugin.OnMessage(ctx, msg); err != nil {
			return fmt.Errorf("plugin '%s' error on message: %w", entry.plugin.Name(), err)
		}
	}

	return nil
}

// OnComplete invokes OnComplete on all enabled plugins
func (pm *PluginManager) OnComplete(ctx context.Context, result *ClaudeResult) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, entry := range pm.plugins {
		if entry.config != nil && !entry.config.Enabled {
			continue
		}
		if err := entry.plugin.OnComplete(ctx, result); err != nil {
			return fmt.Errorf("plugin '%s' error on complete: %w", entry.plugin.Name(), err)
		}
	}

	return nil
}

// Shutdown shuts down all plugins in reverse order
func (pm *PluginManager) Shutdown(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var lastErr error
	// Shutdown in reverse order
	for i := len(pm.plugins) - 1; i >= 0; i-- {
		entry := pm.plugins[i]
		if err := entry.plugin.Shutdown(ctx); err != nil {
			lastErr = fmt.Errorf("failed to shutdown plugin '%s': %w", entry.plugin.Name(), err)
		}
	}

	pm.initialized = false
	return lastErr
}

// List returns the names of all registered plugins
func (pm *PluginManager) List() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, len(pm.plugins))
	for i, entry := range pm.plugins {
		names[i] = entry.plugin.Name()
	}
	return names
}

// Get returns a plugin by name
func (pm *PluginManager) Get(name string) (Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, entry := range pm.plugins {
		if entry.plugin.Name() == name {
			return entry.plugin, true
		}
	}
	return nil, false
}

// Count returns the number of registered plugins
func (pm *PluginManager) Count() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.plugins)
}

// SetEnabled enables or disables a plugin by name
func (pm *PluginManager) SetEnabled(name string, enabled bool) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for i, entry := range pm.plugins {
		if entry.plugin.Name() == name {
			if pm.plugins[i].config == nil {
				pm.plugins[i].config = &PluginConfig{Enabled: enabled, Priority: 100}
			} else {
				pm.plugins[i].config.Enabled = enabled
			}
			return nil
		}
	}
	return fmt.Errorf("plugin '%s' not found", name)
}

// BasePlugin provides a default implementation of the Plugin interface
// Embed this struct to implement only the methods you need
type BasePlugin struct {
	PluginName    string
	PluginVersion string
}

// Name returns the plugin name
func (bp *BasePlugin) Name() string {
	return bp.PluginName
}

// Version returns the plugin version
func (bp *BasePlugin) Version() string {
	return bp.PluginVersion
}

// Initialize is a no-op by default
func (bp *BasePlugin) Initialize(ctx context.Context) error {
	return nil
}

// OnToolCall allows all tool calls by default
func (bp *BasePlugin) OnToolCall(ctx context.Context, toolName string, input ToolInput) error {
	return nil
}

// OnMessage is a no-op by default
func (bp *BasePlugin) OnMessage(ctx context.Context, msg Message) error {
	return nil
}

// OnComplete is a no-op by default
func (bp *BasePlugin) OnComplete(ctx context.Context, result *ClaudeResult) error {
	return nil
}

// Shutdown is a no-op by default
func (bp *BasePlugin) Shutdown(ctx context.Context) error {
	return nil
}

// === Pre-built Plugins ===

// LoggingPlugin logs all tool calls and messages
type LoggingPlugin struct {
	BasePlugin
	Logger    func(format string, args ...interface{})
	LogTools  bool
	LogMsgs   bool
	LogResult bool
}

// NewLoggingPlugin creates a new logging plugin
func NewLoggingPlugin(logger func(format string, args ...interface{})) *LoggingPlugin {
	return &LoggingPlugin{
		BasePlugin: BasePlugin{
			PluginName:    "logging",
			PluginVersion: "1.0.0",
		},
		Logger:    logger,
		LogTools:  true,
		LogMsgs:   true,
		LogResult: true,
	}
}

// OnToolCall logs the tool call
func (lp *LoggingPlugin) OnToolCall(ctx context.Context, toolName string, input ToolInput) error {
	if lp.LogTools && lp.Logger != nil {
		lp.Logger("[logging] Tool call: %s, input: %+v", toolName, input)
	}
	return nil
}

// OnMessage logs the message
func (lp *LoggingPlugin) OnMessage(ctx context.Context, msg Message) error {
	if lp.LogMsgs && lp.Logger != nil {
		lp.Logger("[logging] Message: type=%s, subtype=%s", msg.Type, msg.Subtype)
	}
	return nil
}

// OnComplete logs the result
func (lp *LoggingPlugin) OnComplete(ctx context.Context, result *ClaudeResult) error {
	if lp.LogResult && lp.Logger != nil {
		lp.Logger("[logging] Complete: cost=$%.4f, turns=%d, error=%v", result.CostUSD, result.NumTurns, result.IsError)
	}
	return nil
}

// MetricsPlugin collects execution metrics
type MetricsPlugin struct {
	BasePlugin
	mu             sync.Mutex
	ToolCallCount  map[string]int
	MessageCount   int
	TotalCost      float64
	ExecutionCount int
}

// NewMetricsPlugin creates a new metrics plugin
func NewMetricsPlugin() *MetricsPlugin {
	return &MetricsPlugin{
		BasePlugin: BasePlugin{
			PluginName:    "metrics",
			PluginVersion: "1.0.0",
		},
		ToolCallCount: make(map[string]int),
	}
}

// OnToolCall increments the tool call counter
func (mp *MetricsPlugin) OnToolCall(ctx context.Context, toolName string, input ToolInput) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.ToolCallCount[toolName]++
	return nil
}

// OnMessage increments the message counter
func (mp *MetricsPlugin) OnMessage(ctx context.Context, msg Message) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.MessageCount++
	return nil
}

// OnComplete records execution metrics
func (mp *MetricsPlugin) OnComplete(ctx context.Context, result *ClaudeResult) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.TotalCost += result.CostUSD
	mp.ExecutionCount++
	return nil
}

// GetMetrics returns a copy of the current metrics
func (mp *MetricsPlugin) GetMetrics() map[string]interface{} {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	toolCounts := make(map[string]int)
	for k, v := range mp.ToolCallCount {
		toolCounts[k] = v
	}

	return map[string]interface{}{
		"tool_calls":      toolCounts,
		"message_count":   mp.MessageCount,
		"total_cost":      mp.TotalCost,
		"execution_count": mp.ExecutionCount,
	}
}

// Reset clears all collected metrics
func (mp *MetricsPlugin) Reset() {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.ToolCallCount = make(map[string]int)
	mp.MessageCount = 0
	mp.TotalCost = 0
	mp.ExecutionCount = 0
}

// ToolFilterPlugin blocks specified tools from being executed
type ToolFilterPlugin struct {
	BasePlugin
	BlockedTools map[string]string // tool name -> reason
}

// NewToolFilterPlugin creates a new tool filter plugin
func NewToolFilterPlugin(blockedTools map[string]string) *ToolFilterPlugin {
	if blockedTools == nil {
		blockedTools = make(map[string]string)
	}
	return &ToolFilterPlugin{
		BasePlugin: BasePlugin{
			PluginName:    "tool-filter",
			PluginVersion: "1.0.0",
		},
		BlockedTools: blockedTools,
	}
}

// OnToolCall blocks tools in the blocked list
func (tfp *ToolFilterPlugin) OnToolCall(ctx context.Context, toolName string, input ToolInput) error {
	if reason, blocked := tfp.BlockedTools[toolName]; blocked {
		if reason == "" {
			reason = "tool is blocked"
		}
		return fmt.Errorf("%s: %s", toolName, reason)
	}
	return nil
}

// BlockTool adds a tool to the blocked list
func (tfp *ToolFilterPlugin) BlockTool(name, reason string) {
	tfp.BlockedTools[name] = reason
}

// UnblockTool removes a tool from the blocked list
func (tfp *ToolFilterPlugin) UnblockTool(name string) {
	delete(tfp.BlockedTools, name)
}

// AuditPlugin records all tool calls for auditing
type AuditPlugin struct {
	BasePlugin
	mu      sync.Mutex
	Records []AuditRecord
	MaxSize int // Maximum number of records to keep (0 = unlimited)
}

// AuditRecord represents a single audit entry
type AuditRecord struct {
	Timestamp int64                  `json:"timestamp"`
	ToolName  string                 `json:"tool_name"`
	Input     map[string]interface{} `json:"input"`
	SessionID string                 `json:"session_id,omitempty"`
}

// NewAuditPlugin creates a new audit plugin
func NewAuditPlugin(maxSize int) *AuditPlugin {
	return &AuditPlugin{
		BasePlugin: BasePlugin{
			PluginName:    "audit",
			PluginVersion: "1.0.0",
		},
		Records: make([]AuditRecord, 0),
		MaxSize: maxSize,
	}
}

// OnToolCall records the tool call
func (ap *AuditPlugin) OnToolCall(ctx context.Context, toolName string, input ToolInput) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	record := AuditRecord{
		Timestamp: getCurrentTimestamp(),
		ToolName:  toolName,
		Input:     input.Raw,
	}

	ap.Records = append(ap.Records, record)

	// Trim if over max size
	if ap.MaxSize > 0 && len(ap.Records) > ap.MaxSize {
		ap.Records = ap.Records[len(ap.Records)-ap.MaxSize:]
	}

	return nil
}

// GetRecords returns a copy of all audit records
func (ap *AuditPlugin) GetRecords() []AuditRecord {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	records := make([]AuditRecord, len(ap.Records))
	copy(records, ap.Records)
	return records
}

// Clear removes all audit records
func (ap *AuditPlugin) Clear() {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.Records = make([]AuditRecord, 0)
}

// getCurrentTimestamp returns the current Unix timestamp in milliseconds
func getCurrentTimestamp() int64 {
	return timeNow().UnixMilli()
}

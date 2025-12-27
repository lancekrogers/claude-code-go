package claude

import (
	"context"
	"fmt"
	"regexp"
	"sync"
)

// === Pre-built Plugins ===

// LoggingPlugin logs all tool calls and messages
type LoggingPlugin struct {
	BasePlugin
	Logger          func(format string, args ...interface{})
	LogTools        bool
	LogMsgs         bool
	LogResult       bool
	SanitizeSecrets bool // Redact sensitive patterns (API keys, tokens, passwords)
	TruncateLength  int  // Max chars per field (0 = unlimited)
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
		sanitizedInput := lp.sanitizeInput(input)
		lp.Logger("[logging] Tool call: %s, input: %+v", toolName, sanitizedInput)
	}
	return nil
}

// sanitizeInput applies truncation and secret redaction to tool input
func (lp *LoggingPlugin) sanitizeInput(input ToolInput) ToolInput {
	if !lp.SanitizeSecrets && lp.TruncateLength <= 0 {
		return input // No sanitization needed
	}

	sanitized := ToolInput{
		Command:   lp.sanitizeString(input.Command),
		FilePath:  lp.sanitizeString(input.FilePath),
		Pattern:   lp.sanitizeString(input.Pattern),
		Content:   lp.sanitizeString(input.Content),
		OldString: lp.sanitizeString(input.OldString),
		NewString: lp.sanitizeString(input.NewString),
		Raw:       input.Raw, // Don't modify raw map
	}

	return sanitized
}

// sanitizeString applies truncation and secret redaction to a string
func (lp *LoggingPlugin) sanitizeString(s string) string {
	if s == "" {
		return s
	}

	result := s

	// Apply secret redaction first
	if lp.SanitizeSecrets {
		result = redactSecrets(result)
	}

	// Apply truncation
	if lp.TruncateLength > 0 {
		result = truncateString(result, lp.TruncateLength)
	}

	return result
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}

// secretPatterns contains regex patterns for common secrets
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(api[_-]?key|token|password|secret|credential)[=:]\s*\S+`),
	regexp.MustCompile(`(?i)bearer\s+\S+`),
	regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),                        // OpenAI-style keys
	regexp.MustCompile(`(?i)(aws[_-]?secret|aws[_-]?key)[=:]\s*\S+`), // AWS keys
	regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),                        // GitHub personal access tokens
	regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`),                        // GitHub OAuth tokens
}

// redactSecrets redacts common secret patterns from a string
func redactSecrets(s string) string {
	result := s
	for _, pattern := range secretPatterns {
		result = pattern.ReplaceAllString(result, "[REDACTED]")
	}
	return result
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

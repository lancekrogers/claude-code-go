package claude

import (
	"context"
	"fmt"
	"sync"
)

// SubagentConfig defines a specialized sub-agent configuration
type SubagentConfig struct {
	// Description explains when to use this agent
	// The main agent uses this to decide which subagent to invoke
	Description string `json:"description"`

	// Prompt is the system prompt for the agent
	// This defines the agent's personality, expertise, and behavior
	Prompt string `json:"prompt"`

	// Tools is the list of allowed tools for this agent
	// Supports both standard tools ("Read", "Bash") and MCP tools ("mcp__server__tool")
	Tools []string `json:"tools,omitempty"`

	// Model specifies the model alias to use (sonnet, opus, haiku)
	// If empty, inherits from the parent query's model
	Model string `json:"model,omitempty"`

	// MaxTurns limits the number of turns for this subagent
	// If 0, uses the default from the parent query
	MaxTurns int `json:"max_turns,omitempty"`

	// WorkingDirectory overrides the working directory for this agent
	// If empty, uses the parent query's working directory
	WorkingDirectory string `json:"working_directory,omitempty"`
}

// Validate checks that the SubagentConfig is valid
func (sc *SubagentConfig) Validate() error {
	if sc.Description == "" {
		return fmt.Errorf("subagent description is required")
	}
	if sc.Prompt == "" {
		return fmt.Errorf("subagent prompt is required")
	}
	if sc.Model != "" && !isValidModelAlias(sc.Model) {
		return fmt.Errorf("invalid model alias: %s (must be sonnet, opus, or haiku)", sc.Model)
	}
	// Validate tool names if MCP tools are specified
	for _, tool := range sc.Tools {
		if err := validateMCPTools([]string{tool}); err != nil {
			return err
		}
	}
	return nil
}

// ToRunOptions converts the SubagentConfig into a native Claude CLI
// --agent/--agents selection and returns the corresponding RunOptions.
// It preserves the subagent definition in Agents rather than flattening
// Prompt/Tools into top-level SystemPrompt/AllowedTools fields. The returned
// options select the synthetic agent name "subagent"; use ToNamedRunOptions
// when the caller needs a stable custom agent key.
func (sc *SubagentConfig) ToRunOptions(parentOpts *RunOptions) *RunOptions {
	return sc.ToNamedRunOptions("subagent", parentOpts)
}

// ToNamedRunOptions converts the SubagentConfig into native Claude CLI
// --agent/--agents selection using the provided agent name. If agentName is
// empty, it falls back to "subagent".
func (sc *SubagentConfig) ToNamedRunOptions(agentName string, parentOpts *RunOptions) *RunOptions {
	if agentName == "" {
		agentName = "subagent"
	}

	agents := map[string]*SubagentConfig{
		agentName: cloneSubagentConfig(sc),
	}

	opts := buildAgentRunOptions(agentName, parentOpts, agents)
	if sc.WorkingDirectory != "" {
		opts.WorkingDirectory = sc.WorkingDirectory
	}
	return opts
}

// SubagentManager manages the lifecycle and execution of subagents
type SubagentManager struct {
	mu       sync.RWMutex
	agents   map[string]*SubagentConfig
	client   *ClaudeClient
	sessions map[string]string // agentName -> sessionID
}

// NewSubagentManager creates a new SubagentManager
func NewSubagentManager(client *ClaudeClient) *SubagentManager {
	return &SubagentManager{
		agents:   make(map[string]*SubagentConfig),
		client:   client,
		sessions: make(map[string]string),
	}
}

// RegisterAgent registers a subagent configuration
func (sm *SubagentManager) RegisterAgent(name string, config *SubagentConfig) error {
	if name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("agent config cannot be nil")
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid agent config for %s: %w", name, err)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.agents[name] = config
	return nil
}

// RegisterAgents registers multiple subagent configurations
func (sm *SubagentManager) RegisterAgents(agents map[string]*SubagentConfig) error {
	for name, config := range agents {
		if err := sm.RegisterAgent(name, config); err != nil {
			return err
		}
	}
	return nil
}

// UnregisterAgent removes a subagent registration
func (sm *SubagentManager) UnregisterAgent(name string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.agents, name)
	delete(sm.sessions, name)
}

// GetAgent returns a registered subagent configuration
func (sm *SubagentManager) GetAgent(name string) (*SubagentConfig, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	config, ok := sm.agents[name]
	return config, ok
}

// ListAgents returns the names of all registered subagents
func (sm *SubagentManager) ListAgents() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	names := make([]string, 0, len(sm.agents))
	for name := range sm.agents {
		names = append(names, name)
	}
	return names
}

// GetAgentDescriptions returns a map of agent names to descriptions
// This is useful for providing context to the main agent about available subagents
func (sm *SubagentManager) GetAgentDescriptions() map[string]string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	descriptions := make(map[string]string, len(sm.agents))
	for name, config := range sm.agents {
		descriptions[name] = config.Description
	}
	return descriptions
}

// RunAgent executes a subagent with the given prompt
func (sm *SubagentManager) RunAgent(ctx context.Context, agentName string, prompt string, parentOpts *RunOptions) (*ClaudeResult, error) {
	if _, ok := sm.GetAgent(agentName); !ok {
		return nil, fmt.Errorf("unknown agent: %s", agentName)
	}

	opts := sm.buildRunOptions(agentName, parentOpts)
	result, err := sm.client.RunPromptCtx(ctx, prompt, opts)
	if err != nil {
		return nil, err
	}
	if result.SessionID != "" {
		sm.SetSession(agentName, result.SessionID)
	}
	return result, nil
}

// StreamAgent executes a subagent and streams the results
func (sm *SubagentManager) StreamAgent(ctx context.Context, agentName string, prompt string, parentOpts *RunOptions) (<-chan Message, <-chan error) {
	if _, ok := sm.GetAgent(agentName); !ok {
		errCh := make(chan error, 1)
		errCh <- fmt.Errorf("unknown agent: %s", agentName)
		close(errCh)
		msgCh := make(chan Message)
		close(msgCh)
		return msgCh, errCh
	}

	opts := sm.buildRunOptions(agentName, parentOpts)
	innerMsgCh, innerErrCh := sm.client.StreamPrompt(ctx, prompt, opts)

	msgCh := make(chan Message)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		for innerMsgCh != nil || innerErrCh != nil {
			select {
			case msg, ok := <-innerMsgCh:
				if !ok {
					innerMsgCh = nil
					continue
				}
				if msg.SessionID != "" {
					sm.SetSession(agentName, msg.SessionID)
				}
				msgCh <- msg
			case err, ok := <-innerErrCh:
				if !ok {
					innerErrCh = nil
					continue
				}
				errCh <- err
			}
		}
	}()

	return msgCh, errCh
}

// SetSession stores a session ID for a subagent (for conversation continuity)
func (sm *SubagentManager) SetSession(agentName string, sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sessions[agentName] = sessionID
}

// GetSession retrieves the session ID for a subagent
func (sm *SubagentManager) GetSession(agentName string) (string, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessionID, ok := sm.sessions[agentName]
	return sessionID, ok
}

// ClearSession removes the session ID for a subagent
func (sm *SubagentManager) ClearSession(agentName string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, agentName)
}

// ClearAllSessions removes all stored session IDs
func (sm *SubagentManager) ClearAllSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sessions = make(map[string]string)
}

// ResumeAgent resumes a subagent's previous conversation
func (sm *SubagentManager) ResumeAgent(ctx context.Context, agentName string, prompt string, parentOpts *RunOptions) (*ClaudeResult, error) {
	sessionID, ok := sm.GetSession(agentName)
	if !ok {
		return nil, fmt.Errorf("no session found for agent: %s", agentName)
	}

	if _, configOk := sm.GetAgent(agentName); !configOk {
		return nil, fmt.Errorf("unknown agent: %s", agentName)
	}

	opts := sm.buildRunOptions(agentName, parentOpts)
	opts.ResumeID = sessionID
	result, err := sm.client.RunPromptCtx(ctx, prompt, opts)
	if err != nil {
		return nil, err
	}
	if result.SessionID != "" {
		sm.SetSession(agentName, result.SessionID)
	}
	return result, nil
}

// AgentCount returns the number of registered subagents
func (sm *SubagentManager) AgentCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.agents)
}

// Common subagent configurations for typical use cases

// SecurityReviewerAgent returns a pre-configured security review subagent
func SecurityReviewerAgent() *SubagentConfig {
	return &SubagentConfig{
		Description: "Expert in security auditing and vulnerability analysis. Use this agent for security reviews, penetration testing insights, and identifying security flaws.",
		Prompt: `You are a security expert specializing in application security.
Focus on:
- Authentication and authorization vulnerabilities
- Injection vulnerabilities (SQL, XSS, command injection)
- Insecure dependencies and outdated packages
- Credential exposure and secrets management
- API security issues and rate limiting

Provide detailed explanations with severity levels and remediation steps.`,
		Tools: []string{"Read", "Grep", "Glob"},
		Model: "sonnet",
	}
}

// CodeReviewerAgent returns a pre-configured code review subagent
func CodeReviewerAgent() *SubagentConfig {
	return &SubagentConfig{
		Description: "Code quality and best practices expert. Use for code reviews, refactoring suggestions, and architecture analysis.",
		Prompt: `You are a senior software architect focused on code quality.
Review:
- Code organization and modularity
- Design patterns and SOLID principles
- Error handling and edge cases
- Code duplication and technical debt
- Documentation quality

Provide refactoring suggestions with examples.`,
		Tools: []string{"Read", "Grep", "Glob"},
		Model: "sonnet",
	}
}

// TestAnalystAgent returns a pre-configured test analysis subagent
func TestAnalystAgent() *SubagentConfig {
	return &SubagentConfig{
		Description: "Testing and quality assurance expert. Use for test coverage analysis, test recommendations, and QA improvements.",
		Prompt: `You are a QA and testing expert.
Evaluate:
- Test coverage completeness
- Edge cases and boundary conditions
- Integration test scenarios
- Mock and stub usage
- Test maintainability

Suggest missing tests with code examples.`,
		Tools: []string{"Read", "Grep", "Glob", "Bash"},
		Model: "haiku",
	}
}

// PerformanceAnalystAgent returns a pre-configured performance analysis subagent
func PerformanceAnalystAgent() *SubagentConfig {
	return &SubagentConfig{
		Description: "Performance optimization expert. Use for analyzing bottlenecks, memory leaks, and optimization opportunities.",
		Prompt: `You are a performance optimization specialist.
Analyze:
- Algorithm complexity and bottlenecks
- Memory usage patterns
- Database query optimization
- Caching strategies
- Resource utilization

Provide specific metrics and actionable recommendations.`,
		Tools: []string{"Read", "Grep", "Glob", "Bash"},
		Model: "sonnet",
	}
}

// DocumentationAgent returns a pre-configured documentation subagent
func DocumentationAgent() *SubagentConfig {
	return &SubagentConfig{
		Description: "Documentation specialist. Use for generating or improving documentation, API docs, and README files.",
		Prompt: `You are a technical documentation expert.
Focus on:
- Clear and concise explanations
- Code examples and usage patterns
- API documentation with parameters and return values
- README structure and content
- Inline code comments

Generate well-structured, comprehensive documentation.`,
		Tools: []string{"Read", "Grep", "Glob", "Write"},
		Model: "sonnet",
	}
}

func (sm *SubagentManager) buildRunOptions(agentName string, parentOpts *RunOptions) *RunOptions {
	sm.mu.RLock()
	agents := make(map[string]*SubagentConfig, len(sm.agents))
	for name, config := range sm.agents {
		agents[name] = cloneSubagentConfig(config)
	}
	sm.mu.RUnlock()

	return buildAgentRunOptions(agentName, parentOpts, agents)
}

func buildAgentRunOptions(agentName string, parentOpts *RunOptions, agents map[string]*SubagentConfig) *RunOptions {
	opts := cloneRunOptions(parentOpts)
	if opts.Format == "" {
		opts.Format = StreamJSONOutput
	}
	opts.Agent = agentName
	opts.AgentsJSON = ""
	opts.Agents = agents
	return opts
}

func cloneRunOptions(opts *RunOptions) *RunOptions {
	if opts == nil {
		return &RunOptions{}
	}

	cloned := *opts

	if opts.AllowedTools != nil {
		cloned.AllowedTools = append([]string(nil), opts.AllowedTools...)
	}
	if opts.DisallowedTools != nil {
		cloned.DisallowedTools = append([]string(nil), opts.DisallowedTools...)
	}
	if opts.MCPConfigs != nil {
		cloned.MCPConfigs = append([]string(nil), opts.MCPConfigs...)
	}
	if opts.AddDirectories != nil {
		cloned.AddDirectories = append([]string(nil), opts.AddDirectories...)
	}
	if opts.Betas != nil {
		cloned.Betas = append([]string(nil), opts.Betas...)
	}
	if opts.Files != nil {
		cloned.Files = append([]string(nil), opts.Files...)
	}
	if opts.SettingSources != nil {
		cloned.SettingSources = append([]string(nil), opts.SettingSources...)
	}
	if opts.Tools != nil {
		cloned.Tools = append([]string(nil), opts.Tools...)
	}
	if opts.PluginDirs != nil {
		cloned.PluginDirs = append([]string(nil), opts.PluginDirs...)
	}
	if opts.ParsedAllowedTools != nil {
		cloned.ParsedAllowedTools = append([]ToolPermission(nil), opts.ParsedAllowedTools...)
	}
	if opts.ParsedDisallowedTools != nil {
		cloned.ParsedDisallowedTools = append([]ToolPermission(nil), opts.ParsedDisallowedTools...)
	}
	if opts.Agents != nil {
		cloned.Agents = copySubagentConfigs(opts.Agents)
	}

	return &cloned
}

func copySubagentConfigs(agents map[string]*SubagentConfig) map[string]*SubagentConfig {
	if len(agents) == 0 {
		return nil
	}

	copied := make(map[string]*SubagentConfig, len(agents))
	for name, config := range agents {
		copied[name] = cloneSubagentConfig(config)
	}
	return copied
}

func cloneSubagentConfig(config *SubagentConfig) *SubagentConfig {
	if config == nil {
		return nil
	}

	cloned := *config
	if config.Tools != nil {
		cloned.Tools = append([]string(nil), config.Tools...)
	}
	return &cloned
}

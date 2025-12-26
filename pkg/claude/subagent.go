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

// ToRunOptions converts the SubagentConfig to RunOptions for execution
func (sc *SubagentConfig) ToRunOptions(parentOpts *RunOptions) *RunOptions {
	opts := &RunOptions{
		SystemPrompt: sc.Prompt,
		AllowedTools: sc.Tools,
		Format:       StreamJSONOutput,
	}

	// Use subagent's model or inherit from parent
	if sc.Model != "" {
		opts.ModelAlias = sc.Model
	} else if parentOpts != nil {
		opts.ModelAlias = parentOpts.ModelAlias
		opts.Model = parentOpts.Model
	}

	// Use subagent's max turns or inherit from parent
	if sc.MaxTurns > 0 {
		opts.MaxTurns = sc.MaxTurns
	} else if parentOpts != nil {
		opts.MaxTurns = parentOpts.MaxTurns
	}

	// Use subagent's working directory or inherit from parent
	// Note: WorkingDirectory would need to be added to RunOptions if needed

	// Inherit MCP config from parent
	if parentOpts != nil {
		opts.MCPConfigPath = parentOpts.MCPConfigPath
		opts.PermissionMode = parentOpts.PermissionMode
		opts.PermissionCallback = parentOpts.PermissionCallback
		opts.BudgetTracker = parentOpts.BudgetTracker
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
	config, ok := sm.GetAgent(agentName)
	if !ok {
		return nil, fmt.Errorf("unknown agent: %s", agentName)
	}

	opts := config.ToRunOptions(parentOpts)
	return sm.client.RunPromptCtx(ctx, prompt, opts)
}

// StreamAgent executes a subagent and streams the results
func (sm *SubagentManager) StreamAgent(ctx context.Context, agentName string, prompt string, parentOpts *RunOptions) (<-chan Message, <-chan error) {
	config, ok := sm.GetAgent(agentName)
	if !ok {
		errCh := make(chan error, 1)
		errCh <- fmt.Errorf("unknown agent: %s", agentName)
		close(errCh)
		msgCh := make(chan Message)
		close(msgCh)
		return msgCh, errCh
	}

	opts := config.ToRunOptions(parentOpts)
	return sm.client.StreamPrompt(ctx, prompt, opts)
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

	config, configOk := sm.GetAgent(agentName)
	if !configOk {
		return nil, fmt.Errorf("unknown agent: %s", agentName)
	}

	opts := config.ToRunOptions(parentOpts)
	opts.ResumeID = sessionID
	return sm.client.RunPromptCtx(ctx, prompt, opts)
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

package mom

import (
	"context"
	"fmt"
	"sync"

	"uaa/vdnd/internal/pi-go/agent"
	"uaa/vdnd/internal/pi-go/ai"
)

// RunnerConfig configures an AgentRunner.
type RunnerConfig struct {
	SandboxConfig SandboxConfig
	ChannelID     string
	ChannelDir    string
	WorkspacePath string
	BaseURL       string // Custom API base URL
	Provider      string // LLM provider (e.g. "anthropic", "deepseek")
	ModelID       string // LLM model ID
	GetAPIKey     func() (string, error)
	Settings      *SettingsManager
}

// RunResult contains the outcome of an agent run.
type RunResult struct {
	StopReason   string
	ErrorMessage string
}

// AgentRunner manages an agent session for a single channel.
type AgentRunner struct {
	mu        sync.Mutex
	config    RunnerConfig
	executor  Executor
	agent     *agent.Agent
	isRunning bool
}

// NewAgentRunner creates an AgentRunner for a channel.
func NewAgentRunner(config RunnerConfig) *AgentRunner {
	executor := CreateExecutor(config.SandboxConfig)
	cwd := executor.WorkspacePath(config.WorkspacePath)

	// Create tools via executor
	tools := CreateMomTools(executor, cwd)

	// Build system prompt
	systemPrompt := BuildMomSystemPrompt(
		cwd,
		config.ChannelID,
		"",       // Memory loaded separately
		nil, nil, // Channels/users passed at runtime
	)

	// Resolve model
	provider := config.Provider
	if provider == "" {
		provider = "anthropic"
	}
	modelID := config.ModelID
	if modelID == "" {
		modelID = "claude-sonnet-4-5"
	}

	if config.Settings != nil {
		if p := config.Settings.GetDefaultProvider(); p != "" {
			provider = p
		}
		if m := config.Settings.GetDefaultModel(); m != "" {
			modelID = m
		}
	}

	model := ai.Model{ID: modelID, Provider: provider, Api: apiForProvider(provider), BaseURL: config.BaseURL}

	opts := agent.AgentOptions{
		Model:        model,
		SystemPrompt: systemPrompt,
		Tools:        tools,
		GetApiKey: func(provider string) (string, error) {
			if config.GetAPIKey != nil {
				return config.GetAPIKey()
			}
			key := ai.GetEnvApiKey(provider)
			if key == "" {
				return "", fmt.Errorf("no API key found for provider %s", provider)
			}
			return key, nil
		},
	}

	return &AgentRunner{
		config:   config,
		executor: executor,
		agent:    agent.NewAgent(opts),
	}
}

// Run executes the agent with the given user message.
func (r *AgentRunner) Run(ctx context.Context, userMessage string) (RunResult, error) {
	r.mu.Lock()
	if r.isRunning {
		r.mu.Unlock()
		return RunResult{StopReason: "busy"}, fmt.Errorf("agent is already running in channel %s", r.config.ChannelID)
	}
	r.isRunning = true
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.isRunning = false
		r.mu.Unlock()
	}()

	err := r.agent.Prompt(ctx, userMessage)
	if err != nil {
		if ctx.Err() != nil {
			return RunResult{StopReason: "aborted"}, nil
		}
		return RunResult{StopReason: "error", ErrorMessage: err.Error()}, err
	}

	return RunResult{StopReason: "completed"}, nil
}

// Abort cancels the current agent run.
func (r *AgentRunner) Abort() {
	r.agent.Abort()
}

// Subscribe registers a listener for agent events. Returns an unsubscribe function.
func (r *AgentRunner) Subscribe(fn func(agent.Event)) func() {
	return r.agent.Subscribe(fn)
}

// IsRunning returns whether the agent is currently running.
func (r *AgentRunner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isRunning
}

// --- Runner cache (one runner per channel) ---

var (
	channelRunnersMu sync.Mutex
	channelRunners   = map[string]*AgentRunner{}
)

// GetOrCreateRunner returns the cached runner for a channel, creating one if needed.
func GetOrCreateRunner(config RunnerConfig) *AgentRunner {
	channelRunnersMu.Lock()
	defer channelRunnersMu.Unlock()

	if runner, ok := channelRunners[config.ChannelID]; ok {
		return runner
	}

	runner := NewAgentRunner(config)
	channelRunners[config.ChannelID] = runner
	return runner
}

// RemoveRunner removes a cached runner for a channel.
func RemoveRunner(channelID string) {
	channelRunnersMu.Lock()
	defer channelRunnersMu.Unlock()
	delete(channelRunners, channelID)
}

// apiForProvider maps a provider name to its API protocol identifier.
func apiForProvider(provider string) ai.Api {
	switch provider {
	case ai.ProviderAnthropic:
		return ai.ApiAnthropicMessages
	case ai.ProviderOpenAI:
		return ai.ApiOpenAIResponses
	case ai.ProviderGoogle:
		return ai.ApiGoogleGenerativeAI
	case ai.ProviderDeepSeek, ai.ProviderGroq, ai.ProviderCerebras, ai.ProviderOpenRouter, ai.ProviderXAI, ai.ProviderMistral:
		return ai.ApiOpenAICompletions
	default:
		return ai.ApiAnthropicMessages // Default fallback
	}
}

package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"uaa/vdnd/internal/pi-go/ai"
)

// DefaultConvertToLlm keeps only LLM-compatible messages (user, assistant, toolResult).
func DefaultConvertToLlm(messages []Message) ([]ai.Message, error) {
	var result []ai.Message
	for _, m := range messages {
		switch m.Role {
		case ai.RoleUser, ai.RoleAssistant, ai.RoleToolResult:
			result = append(result, m)
		}
	}
	return result, nil
}

// SteeringMode controls how queued steering messages are delivered.
type SteeringMode string

const (
	SteeringAll        SteeringMode = "all"
	SteeringOneAtATime SteeringMode = "one-at-a-time"
)

// AgentOptions configures an Agent.
type AgentOptions struct {
	SystemPrompt     string
	Model            ai.Model
	ThinkingLevel    ai.ThinkingLevel
	Tools            []Tool
	ConvertToLlm     ConvertToLlmFunc
	TransformContext TransformContextFunc
	SteeringMode     SteeringMode
	FollowUpMode     SteeringMode
	GetApiKey        GetApiKeyFunc
	SessionID        string
	ThinkingBudgets  *ai.ThinkingBudgets
	MaxRetryDelayMs  *int
}

// Agent manages the full lifecycle of an agentic conversation.
// It wraps the agent loop with state management, steering/follow-up queues, and event subscriptions.
type Agent struct {
	mu sync.Mutex

	// State
	systemPrompt     string
	model            ai.Model
	thinkingLevel    ai.ThinkingLevel
	tools            []Tool
	messages         []Message
	isStreaming      bool
	pendingToolCalls map[string]struct{}
	lastError        string

	// Config
	convertToLlm     ConvertToLlmFunc
	transformContext TransformContextFunc
	getApiKey        GetApiKeyFunc
	sessionID        string
	thinkingBudgets  *ai.ThinkingBudgets
	maxRetryDelayMs  *int
	steeringMode     SteeringMode
	followUpMode     SteeringMode

	// Queues
	steeringQueue []Message
	followUpQueue []Message

	// Event listeners
	listeners []func(Event)

	// Cancellation
	cancel context.CancelFunc
	done   chan struct{}
}

// NewAgent creates a new Agent with the given options.
func NewAgent(opts AgentOptions) *Agent {
	convertToLlm := opts.ConvertToLlm
	if convertToLlm == nil {
		convertToLlm = DefaultConvertToLlm
	}
	steeringMode := opts.SteeringMode
	if steeringMode == "" {
		steeringMode = SteeringOneAtATime
	}
	followUpMode := opts.FollowUpMode
	if followUpMode == "" {
		followUpMode = SteeringOneAtATime
	}

	return &Agent{
		systemPrompt:     opts.SystemPrompt,
		model:            opts.Model,
		thinkingLevel:    opts.ThinkingLevel,
		tools:            opts.Tools,
		convertToLlm:     convertToLlm,
		transformContext: opts.TransformContext,
		getApiKey:        opts.GetApiKey,
		sessionID:        opts.SessionID,
		thinkingBudgets:  opts.ThinkingBudgets,
		maxRetryDelayMs:  opts.MaxRetryDelayMs,
		steeringMode:     steeringMode,
		followUpMode:     followUpMode,
		pendingToolCalls: make(map[string]struct{}),
	}
}

// Subscribe registers an event listener. Returns an unsubscribe function.
func (a *Agent) Subscribe(fn func(Event)) func() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.listeners = append(a.listeners, fn)
	idx := len(a.listeners) - 1
	return func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		// Set to nil rather than removing to preserve indices
		a.listeners[idx] = nil
	}
}

func (a *Agent) emit(e Event) {
	a.mu.Lock()
	listeners := make([]func(Event), len(a.listeners))
	copy(listeners, a.listeners)
	a.mu.Unlock()
	for _, fn := range listeners {
		if fn != nil {
			fn(e)
		}
	}
}

// --- State accessors ---

func (a *Agent) SetSystemPrompt(v string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.systemPrompt = v
}

func (a *Agent) SetModel(m ai.Model) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.model = m
}

func (a *Agent) SetThinkingLevel(l ai.ThinkingLevel) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.thinkingLevel = l
}

func (a *Agent) SetTools(t []Tool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tools = t
}

func (a *Agent) Messages() []Message {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]Message, len(a.messages))
	copy(result, a.messages)
	return result
}

func (a *Agent) IsStreaming() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.isStreaming
}

func (a *Agent) LastError() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastError
}

func (a *Agent) ReplaceMessages(msgs []Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messages = make([]Message, len(msgs))
	copy(a.messages, msgs)
}

func (a *Agent) AppendMessage(m Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messages = append(a.messages, m)
}

func (a *Agent) ClearMessages() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messages = nil
}

// --- Queue management ---

// Steer queues a steering message to interrupt the agent mid-run.
func (a *Agent) Steer(m Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.steeringQueue = append(a.steeringQueue, m)
}

// FollowUp queues a follow-up message for after the agent finishes.
func (a *Agent) FollowUp(m Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.followUpQueue = append(a.followUpQueue, m)
}

func (a *Agent) ClearSteeringQueue() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.steeringQueue = nil
}

func (a *Agent) ClearFollowUpQueue() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.followUpQueue = nil
}

func (a *Agent) ClearAllQueues() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.steeringQueue = nil
	a.followUpQueue = nil
}

func (a *Agent) HasQueuedMessages() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.steeringQueue) > 0 || len(a.followUpQueue) > 0
}

func (a *Agent) dequeueSteeringMessages() []Message {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.steeringMode == SteeringOneAtATime {
		if len(a.steeringQueue) > 0 {
			first := a.steeringQueue[0]
			a.steeringQueue = a.steeringQueue[1:]
			return []Message{first}
		}
		return nil
	}
	result := a.steeringQueue
	a.steeringQueue = nil
	return result
}

func (a *Agent) dequeueFollowUpMessages() []Message {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.followUpMode == SteeringOneAtATime {
		if len(a.followUpQueue) > 0 {
			first := a.followUpQueue[0]
			a.followUpQueue = a.followUpQueue[1:]
			return []Message{first}
		}
		return nil
	}
	result := a.followUpQueue
	a.followUpQueue = nil
	return result
}

// --- Control ---

// Abort cancels the current agent loop.
func (a *Agent) Abort() {
	a.mu.Lock()
	cancel := a.cancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// WaitForIdle blocks until the agent is no longer streaming.
func (a *Agent) WaitForIdle() {
	a.mu.Lock()
	done := a.done
	a.mu.Unlock()
	if done != nil {
		<-done
	}
}

// Reset clears all agent state.
func (a *Agent) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messages = nil
	a.isStreaming = false
	a.pendingToolCalls = make(map[string]struct{})
	a.lastError = ""
	a.steeringQueue = nil
	a.followUpQueue = nil
}

// --- Prompt ---

// Prompt sends a text prompt to the agent, starting a new conversation turn.
func (a *Agent) Prompt(ctx context.Context, text string) error {
	userMsg := ai.NewUserMsg(ai.UserMessage{
		Role:      ai.RoleUser,
		Content:   []ai.ContentBlock{{Type: ai.ContentTypeText, Text: text}},
		Timestamp: time.Now().UnixMilli(),
	})
	return a.PromptMessages(ctx, []Message{userMsg})
}

// PromptMessages sends agent messages as a prompt.
func (a *Agent) PromptMessages(ctx context.Context, msgs []Message) error {
	a.mu.Lock()
	if a.isStreaming {
		a.mu.Unlock()
		return fmt.Errorf("agent is already processing a prompt; use Steer() or FollowUp() to queue messages")
	}
	a.mu.Unlock()

	return a.runLoop(ctx, msgs, false)
}

// Continue resumes from the current context (retries, queued messages).
func (a *Agent) Continue(ctx context.Context) error {
	a.mu.Lock()
	if a.isStreaming {
		a.mu.Unlock()
		return fmt.Errorf("agent is already processing; wait for completion before continuing")
	}
	msgs := a.messages
	if len(msgs) == 0 {
		a.mu.Unlock()
		return fmt.Errorf("no messages to continue from")
	}
	lastRole := msgs[len(msgs)-1].Role
	a.mu.Unlock()

	if lastRole == ai.RoleAssistant {
		// Try queued steering
		steering := a.dequeueSteeringMessages()
		if len(steering) > 0 {
			return a.runLoop(ctx, steering, true)
		}
		// Try follow-up
		followUp := a.dequeueFollowUpMessages()
		if len(followUp) > 0 {
			return a.runLoop(ctx, followUp, false)
		}
		return fmt.Errorf("cannot continue from message role: assistant")
	}

	return a.runLoop(ctx, nil, false)
}

func (a *Agent) runLoop(ctx context.Context, messages []Message, skipInitialSteeringPoll bool) error {
	a.mu.Lock()
	a.isStreaming = true
	a.lastError = ""
	a.pendingToolCalls = make(map[string]struct{})

	loopCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	a.done = make(chan struct{})
	done := a.done

	model := a.model
	reasoning := a.thinkingLevel

	agentCtx := AgentContext{
		SystemPrompt: a.systemPrompt,
		Messages:     make([]Message, len(a.messages)),
		Tools:        a.tools,
	}
	copy(agentCtx.Messages, a.messages)

	skipSteering := skipInitialSteeringPoll
	a.mu.Unlock()

	config := LoopConfig{
		Model:            model,
		ConvertToLlm:     a.convertToLlm,
		TransformContext: a.transformContext,
		GetApiKey:        a.getApiKey,
		Reasoning:        reasoning,
		ThinkingBudgets:  a.thinkingBudgets,
		SessionID:        a.sessionID,
		MaxRetryDelayMs:  a.maxRetryDelayMs,
		GetSteeringMessages: func() ([]Message, error) {
			if skipSteering {
				skipSteering = false
				return nil, nil
			}
			return a.dequeueSteeringMessages(), nil
		},
		GetFollowUpMessages: func() ([]Message, error) {
			return a.dequeueFollowUpMessages(), nil
		},
	}

	var stream *AgentEventStream
	if messages != nil {
		stream = AgentLoop(loopCtx, messages, agentCtx, config)
	} else {
		var err error
		stream, err = AgentLoopContinue(loopCtx, agentCtx, config)
		if err != nil {
			cancel()
			a.mu.Lock()
			a.isStreaming = false
			close(done)
			a.done = nil
			a.cancel = nil
			a.mu.Unlock()
			return err
		}
	}

	// Process events
	var finalErr error
	for event := range stream.Events() {
		// Update internal state based on events
		a.mu.Lock()
		switch event.Type {
		case EventMessageStart:
			// nothing specific
		case EventMessageEnd:
			if event.Message != nil {
				a.messages = append(a.messages, *event.Message)
			}
		case EventToolExecutionStart:
			if event.ToolCallID != "" {
				a.pendingToolCalls[event.ToolCallID] = struct{}{}
			}
		case EventToolExecutionEnd:
			delete(a.pendingToolCalls, event.ToolCallID)
		case EventTurnEnd:
			if event.Message != nil && event.Message.Assistant != nil && event.Message.Assistant.ErrorMessage != "" {
				a.lastError = event.Message.Assistant.ErrorMessage
			}
		case EventAgentEnd:
			a.isStreaming = false
		}
		a.mu.Unlock()

		a.emit(event)
	}

	a.mu.Lock()
	a.isStreaming = false
	a.cancel = nil
	close(done)
	a.done = nil
	a.mu.Unlock()

	cancel()
	return finalErr
}

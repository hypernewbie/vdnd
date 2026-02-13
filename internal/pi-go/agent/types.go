// Package agent provides the agent loop and Agent class for agentic LLM interactions.
package agent

import (
	"context"
	"encoding/json"

	"uaa/vdnd/internal/pi-go/ai"
)

// Message is the agent message type — a union of LLM messages + custom messages.
// At the agent level, all messages flow through this type.
// Custom message types can be added by embedding additional data.
type Message = ai.Message

// --- Agent Loop Config ---

// ConvertToLlmFunc converts agent messages to LLM-compatible messages before each LLM call.
// Non-LLM messages (notifications, UI-only) should be filtered out.
type ConvertToLlmFunc func(messages []Message) ([]ai.Message, error)

// TransformContextFunc optionally transforms messages before ConvertToLlm.
// Use for context window management, injecting external context, etc.
type TransformContextFunc func(ctx context.Context, messages []Message) ([]Message, error)

// GetApiKeyFunc resolves an API key dynamically for each LLM call.
// Useful for short-lived OAuth tokens that may expire during tool execution.
type GetApiKeyFunc func(provider string) (string, error)

// GetMessagesFunc returns messages to inject into the conversation.
// Used for steering (mid-run interrupts) and follow-ups (post-run continuations).
type GetMessagesFunc func() ([]Message, error)

// LoopConfig configures the agent loop behavior.
type LoopConfig struct {
	Model               ai.Model
	ConvertToLlm        ConvertToLlmFunc
	TransformContext    TransformContextFunc // Optional
	GetApiKey           GetApiKeyFunc        // Optional
	GetSteeringMessages GetMessagesFunc      // Optional — called after each tool to check for interrupts
	GetFollowUpMessages GetMessagesFunc      // Optional — called when agent would stop

	// SimpleStreamOptions fields
	Reasoning       ai.ThinkingLevel
	ThinkingBudgets *ai.ThinkingBudgets
	APIKey          string
	SessionID       string
	MaxRetryDelayMs *int
}

// --- Agent Tool ---

// ToolResult is the result of executing a tool.
type ToolResult struct {
	Content []ai.ContentBlock `json:"content"`
	Details json.RawMessage   `json:"details,omitempty"`
}

// ToolUpdateFunc is a callback for streaming partial tool results during execution.
type ToolUpdateFunc func(partialResult ToolResult)

// Tool extends ai.Tool with an Execute function.
type Tool struct {
	ai.Tool
	Label   string
	Execute func(ctx context.Context, toolCallID string, params map[string]any, onUpdate ToolUpdateFunc) (ToolResult, error)
}

// --- Agent Context ---

// AgentContext is the context passed to the agent loop.
type AgentContext struct {
	SystemPrompt string
	Messages     []Message
	Tools        []Tool
}

// --- Agent Events ---

// EventType discriminates between agent event types.
type EventType string

const (
	EventAgentStart          EventType = "agent_start"
	EventAgentEnd            EventType = "agent_end"
	EventTurnStart           EventType = "turn_start"
	EventTurnEnd             EventType = "turn_end"
	EventMessageStart        EventType = "message_start"
	EventMessageUpdate       EventType = "message_update"
	EventMessageEnd          EventType = "message_end"
	EventToolExecutionStart  EventType = "tool_execution_start"
	EventToolExecutionUpdate EventType = "tool_execution_update"
	EventToolExecutionEnd    EventType = "tool_execution_end"
)

// Event is emitted by the agent loop for UI updates and lifecycle tracking.
type Event struct {
	Type EventType `json:"type"`

	// agent_end
	Messages []Message `json:"messages,omitempty"`

	// message_start, message_update, message_end, turn_end
	Message *Message `json:"message,omitempty"`

	// message_update
	AssistantMessageEvent *ai.AssistantMessageEvent `json:"assistantMessageEvent,omitempty"`

	// turn_end
	ToolResults []ai.ToolResultMessage `json:"toolResults,omitempty"`

	// tool_execution_start, tool_execution_update, tool_execution_end
	ToolCallID string         `json:"toolCallId,omitempty"`
	ToolName   string         `json:"toolName,omitempty"`
	Args       map[string]any `json:"args,omitempty"`

	// tool_execution_update
	PartialResult *ToolResult `json:"partialResult,omitempty"`

	// tool_execution_end
	Result  *ToolResult `json:"result,omitempty"`
	IsError bool        `json:"isError,omitempty"`
}

// AgentEventStream is the event stream type for agent events.
type AgentEventStream = ai.EventStream[Event, []Message]

// NewAgentEventStream creates a new event stream for agent events.
func NewAgentEventStream() *AgentEventStream {
	return ai.NewEventStream[Event, []Message](
		func(event Event) bool {
			return event.Type == EventAgentEnd
		},
		func(event Event) []Message {
			if event.Type == EventAgentEnd {
				return event.Messages
			}
			return nil
		},
	)
}

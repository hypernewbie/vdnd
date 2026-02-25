package llmtypes

import (
	"context"
)

// ToolCall represents a request from the LLM to call a specific tool.
type ToolCall struct {
	ID               string
	Name             string
	Arguments        string
	ThoughtSignature string
}

// Message represents a single turn in a conversation.
type Message struct {
	Role       string // "user", "model", "tool", "system"
	Content    string
	Thinking   string     // Internal model reasoning (e.g., DeepSeek reasoning_content)
	ToolCalls  []ToolCall // If model is calling tools
	ToolCallID string     // If role is "tool", which call is being answered
	Name       string     // Tool name (for tool role)
}

// Tool represents a piece of functionality that the LLM can invoke.
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// GenerationResponse is the result of a call to GenerateWithTools.
type GenerationResponse struct {
	Content      string
	ToolCalls    []ToolCall
	Thinking     string // Internal model reasoning/thoughts
	FinishReason string // "stop" or "tool_calls"
}

// Provider is an abstraction for different LLM backends.
type Provider interface {
	Name() string
	ModelName() string
	Generate(ctx context.Context, messages []Message) (string, error)
	GenerateWithTools(ctx context.Context, messages []Message, tools []Tool) (GenerationResponse, error)
	GenerateStream(ctx context.Context, messages []Message, tools []Tool, callback func(chunk string) error) (GenerationResponse, error)
	SupportsToolCalling() bool
}

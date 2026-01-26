package llm

import (
	"context"
	"strings"
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
	SupportsToolCalling() bool
}

func ExtractThinking(content string) string {
	startTag := "<thought>"
	endTag := "</thought>"

	start := strings.Index(content, startTag)
	end := strings.Index(content, endTag)

	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(content[start+len(startTag) : end])
	}

	return ""
}

func StripThinking(content string) string {
	startTag := "<thought>"
	endTag := "</thought>"

	for {
		start := strings.Index(content, startTag)
		end := strings.Index(content, endTag)

		if start != -1 && end != -1 && end > start {
			content = content[:start] + content[end+len(endTag):]
			continue
		}
		break
	}
	return strings.TrimSpace(content)
}

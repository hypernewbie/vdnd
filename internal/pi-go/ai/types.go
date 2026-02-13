// Package ai provides core types and streaming infrastructure for LLM interactions.
package ai

import "encoding/json"

// --- API and Provider identifiers ---

// Api identifies a specific LLM API protocol (e.g. "anthropic-messages", "openai-responses").
type Api = string

// Known API identifiers.
const (
	ApiOpenAICompletions     Api = "openai-completions"
	ApiOpenAIResponses       Api = "openai-responses"
	ApiAzureOpenAIResponses  Api = "azure-openai-responses"
	ApiOpenAICodexResponses  Api = "openai-codex-responses"
	ApiAnthropicMessages     Api = "anthropic-messages"
	ApiBedrockConverseStream Api = "bedrock-converse-stream"
	ApiGoogleGenerativeAI    Api = "google-generative-ai"
	ApiGoogleGeminiCLI       Api = "google-gemini-cli"
	ApiGoogleVertex          Api = "google-vertex"
)

// Provider identifies an LLM provider (e.g. "anthropic", "openai").
type Provider = string

// Known provider identifiers.
const (
	ProviderAmazonBedrock        Provider = "amazon-bedrock"
	ProviderAnthropic            Provider = "anthropic"
	ProviderGoogle               Provider = "google"
	ProviderGoogleGeminiCLI      Provider = "google-gemini-cli"
	ProviderGoogleAntigravity    Provider = "google-antigravity"
	ProviderGoogleVertex         Provider = "google-vertex"
	ProviderOpenAI               Provider = "openai"
	ProviderAzureOpenAIResponses Provider = "azure-openai-responses"
	ProviderOpenAICodex          Provider = "openai-codex"
	ProviderGitHubCopilot        Provider = "github-copilot"
	ProviderXAI                  Provider = "xai"
	ProviderGroq                 Provider = "groq"
	ProviderCerebras             Provider = "cerebras"
	ProviderOpenRouter           Provider = "openrouter"
	ProviderVercelAIGateway      Provider = "vercel-ai-gateway"
	ProviderZAI                  Provider = "zai"
	ProviderDeepSeek             Provider = "deepseek"
	ProviderMistral              Provider = "mistral"
	ProviderMinimax              Provider = "minimax"
	ProviderMinimaxCN            Provider = "minimax-cn"
	ProviderHuggingFace          Provider = "huggingface"
	ProviderOpenCode             Provider = "opencode"
	ProviderKimiCoding           Provider = "kimi-coding"
)

// --- Thinking / reasoning ---

// ThinkingLevel controls reasoning effort for models that support it.
type ThinkingLevel string

const (
	ThinkingMinimal ThinkingLevel = "minimal"
	ThinkingLow     ThinkingLevel = "low"
	ThinkingMedium  ThinkingLevel = "medium"
	ThinkingHigh    ThinkingLevel = "high"
	ThinkingXHigh   ThinkingLevel = "xhigh"
)

// ThinkingBudgets sets token budgets for each thinking level (token-based providers only).
type ThinkingBudgets struct {
	Minimal *int `json:"minimal,omitempty"`
	Low     *int `json:"low,omitempty"`
	Medium  *int `json:"medium,omitempty"`
	High    *int `json:"high,omitempty"`
}

// --- Cache retention ---

// CacheRetention controls prompt cache retention preference.
type CacheRetention string

const (
	CacheRetentionNone  CacheRetention = "none"
	CacheRetentionShort CacheRetention = "short"
	CacheRetentionLong  CacheRetention = "long"
)

// --- Stream options ---

// StreamOptions are the base options all providers share.
type StreamOptions struct {
	Temperature     *float64          `json:"temperature,omitempty"`
	MaxTokens       *int              `json:"maxTokens,omitempty"`
	APIKey          string            `json:"apiKey,omitempty"`
	CacheRetention  CacheRetention    `json:"cacheRetention,omitempty"`
	SessionID       string            `json:"sessionId,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	MaxRetryDelayMs *int              `json:"maxRetryDelayMs,omitempty"`
	Metadata        map[string]any    `json:"metadata,omitempty"`
}

// SimpleStreamOptions extends StreamOptions with reasoning control.
type SimpleStreamOptions struct {
	StreamOptions
	Reasoning       ThinkingLevel    `json:"reasoning,omitempty"`
	ThinkingBudgets *ThinkingBudgets `json:"thinkingBudgets,omitempty"`
}

// --- Content types ---

// ContentType discriminates between content block types.
type ContentType string

const (
	ContentTypeText     ContentType = "text"
	ContentTypeThinking ContentType = "thinking"
	ContentTypeImage    ContentType = "image"
	ContentTypeToolCall ContentType = "toolCall"
)

// TextContent is a text block in a message.
type TextContent struct {
	Type          ContentType `json:"type"` // Always "text"
	Text          string      `json:"text"`
	TextSignature string      `json:"textSignature,omitempty"`
}

// ThinkingContent is an internal reasoning block.
type ThinkingContent struct {
	Type              ContentType `json:"type"` // Always "thinking"
	Thinking          string      `json:"thinking"`
	ThinkingSignature string      `json:"thinkingSignature,omitempty"`
}

// ImageContent is a base64-encoded image block.
type ImageContent struct {
	Type     ContentType `json:"type"` // Always "image"
	Data     string      `json:"data"`
	MimeType string      `json:"mimeType"`
}

// ToolCall is a request from the LLM to invoke a tool.
type ToolCall struct {
	Type             ContentType    `json:"type"` // Always "toolCall"
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Arguments        map[string]any `json:"arguments"`
	ThoughtSignature string         `json:"thoughtSignature,omitempty"`
}

// ContentBlock is a union of all content block types that can appear in messages.
// Use the Type field to discriminate.
type ContentBlock struct {
	Type ContentType `json:"type"`

	// Text fields (type == "text")
	Text          string `json:"text,omitempty"`
	TextSignature string `json:"textSignature,omitempty"`

	// Thinking fields (type == "thinking")
	Thinking          string `json:"thinking,omitempty"`
	ThinkingSignature string `json:"thinkingSignature,omitempty"`

	// Image fields (type == "image")
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`

	// ToolCall fields (type == "toolCall")
	ID               string         `json:"id,omitempty"`
	Name             string         `json:"name,omitempty"`
	Arguments        map[string]any `json:"arguments,omitempty"`
	ThoughtSignature string         `json:"thoughtSignature,omitempty"`
}

// AsText returns the TextContent view if this is a text block.
func (c *ContentBlock) AsText() *TextContent {
	if c.Type != ContentTypeText {
		return nil
	}
	return &TextContent{Type: c.Type, Text: c.Text, TextSignature: c.TextSignature}
}

// AsThinking returns the ThinkingContent view if this is a thinking block.
func (c *ContentBlock) AsThinking() *ThinkingContent {
	if c.Type != ContentTypeThinking {
		return nil
	}
	return &ThinkingContent{Type: c.Type, Thinking: c.Thinking, ThinkingSignature: c.ThinkingSignature}
}

// AsImage returns the ImageContent view if this is an image block.
func (c *ContentBlock) AsImage() *ImageContent {
	if c.Type != ContentTypeImage {
		return nil
	}
	return &ImageContent{Type: c.Type, Data: c.Data, MimeType: c.MimeType}
}

// AsToolCall returns the ToolCall view if this is a toolCall block.
func (c *ContentBlock) AsToolCall() *ToolCall {
	if c.Type != ContentTypeToolCall {
		return nil
	}
	return &ToolCall{
		Type: c.Type, ID: c.ID, Name: c.Name,
		Arguments: c.Arguments, ThoughtSignature: c.ThoughtSignature,
	}
}

// --- Usage ---

// CostBreakdown holds cost in dollars across input/output/cache categories.
type CostBreakdown struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
	Total      float64 `json:"total"`
}

// Usage tracks token usage for a generation.
type Usage struct {
	Input       int           `json:"input"`
	Output      int           `json:"output"`
	CacheRead   int           `json:"cacheRead"`
	CacheWrite  int           `json:"cacheWrite"`
	TotalTokens int           `json:"totalTokens"`
	Cost        CostBreakdown `json:"cost"`
}

// --- Stop reason ---

// StopReason indicates why generation stopped.
type StopReason string

const (
	StopReasonStop    StopReason = "stop"
	StopReasonLength  StopReason = "length"
	StopReasonToolUse StopReason = "toolUse"
	StopReasonError   StopReason = "error"
	StopReasonAborted StopReason = "aborted"
)

// --- Messages ---

// Role discriminates between message types.
type Role string

const (
	RoleUser       Role = "user"
	RoleAssistant  Role = "assistant"
	RoleToolResult Role = "toolResult"
)

// UserMessage represents a user turn in the conversation.
type UserMessage struct {
	Role      Role           `json:"role"` // Always "user"
	Content   []ContentBlock `json:"content"`
	Timestamp int64          `json:"timestamp"` // Unix timestamp in milliseconds
}

// NewUserTextMessage creates a UserMessage with a simple text content.
func NewUserTextMessage(text string, timestamp int64) UserMessage {
	return UserMessage{
		Role: RoleUser,
		Content: []ContentBlock{
			{Type: ContentTypeText, Text: text},
		},
		Timestamp: timestamp,
	}
}

// AssistantMessage represents an assistant turn (LLM response).
type AssistantMessage struct {
	Role         Role           `json:"role"` // Always "assistant"
	Content      []ContentBlock `json:"content"`
	Api          Api            `json:"api"`
	Provider     Provider       `json:"provider"`
	Model        string         `json:"model"`
	Usage        Usage          `json:"usage"`
	StopReason   StopReason     `json:"stopReason"`
	ErrorMessage string         `json:"errorMessage,omitempty"`
	Timestamp    int64          `json:"timestamp"` // Unix timestamp in milliseconds
}

// ToolResultMessage represents the result of a tool execution.
type ToolResultMessage struct {
	Role       Role            `json:"role"` // Always "toolResult"
	ToolCallID string          `json:"toolCallId"`
	ToolName   string          `json:"toolName"`
	Content    []ContentBlock  `json:"content"`
	Details    json.RawMessage `json:"details,omitempty"`
	IsError    bool            `json:"isError"`
	Timestamp  int64           `json:"timestamp"` // Unix timestamp in milliseconds
}

// Message is a union of all message types. Only one of User/Assistant/ToolResult is non-nil.
type Message struct {
	Role Role `json:"role"`

	// Exactly one of these is populated, based on Role.
	User       *UserMessage       `json:"-"`
	Assistant  *AssistantMessage  `json:"-"`
	ToolResult *ToolResultMessage `json:"-"`
}

// NewUserMsg wraps a UserMessage as a Message.
func NewUserMsg(m UserMessage) Message {
	return Message{Role: RoleUser, User: &m}
}

// NewAssistantMsg wraps an AssistantMessage as a Message.
func NewAssistantMsg(m AssistantMessage) Message {
	return Message{Role: RoleAssistant, Assistant: &m}
}

// NewToolResultMsg wraps a ToolResultMessage as a Message.
func NewToolResultMsg(m ToolResultMessage) Message {
	return Message{Role: RoleToolResult, ToolResult: &m}
}

// --- Tool ---

// Tool represents a tool the LLM can invoke.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema
}

// --- Context ---

// Context is the full context sent to an LLM for generation.
type Context struct {
	SystemPrompt string    `json:"systemPrompt,omitempty"`
	Messages     []Message `json:"messages"`
	Tools        []Tool    `json:"tools,omitempty"`
}

// --- Model ---

// ModelCost holds pricing per million tokens.
type ModelCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

// InputModality indicates what input types a model supports.
type InputModality string

const (
	InputText  InputModality = "text"
	InputImage InputModality = "image"
)

// Model describes an LLM model with its capabilities and pricing.
type Model struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Api           Api               `json:"api"`
	Provider      Provider          `json:"provider"`
	BaseURL       string            `json:"baseUrl"`
	Reasoning     bool              `json:"reasoning"`
	Input         []InputModality   `json:"input"`
	Cost          ModelCost         `json:"cost"`
	ContextWindow int               `json:"contextWindow"`
	MaxTokens     int               `json:"maxTokens"`
	Headers       map[string]string `json:"headers,omitempty"`
}

// --- AssistantMessageEvent ---

// EventType discriminates between streaming event types.
type EventType string

const (
	EventStart         EventType = "start"
	EventTextStart     EventType = "text_start"
	EventTextDelta     EventType = "text_delta"
	EventTextEnd       EventType = "text_end"
	EventThinkingStart EventType = "thinking_start"
	EventThinkingDelta EventType = "thinking_delta"
	EventThinkingEnd   EventType = "thinking_end"
	EventToolCallStart EventType = "toolcall_start"
	EventToolCallDelta EventType = "toolcall_delta"
	EventToolCallEnd   EventType = "toolcall_end"
	EventDone          EventType = "done"
	EventError         EventType = "error"
)

// AssistantMessageEvent is emitted during streaming of an assistant response.
type AssistantMessageEvent struct {
	Type EventType `json:"type"`

	// Present on most events (the in-progress message).
	Partial *AssistantMessage `json:"partial,omitempty"`

	// ContentIndex for text/thinking/toolcall start/delta/end events.
	ContentIndex int `json:"contentIndex,omitempty"`

	// Delta for text_delta, thinking_delta, toolcall_delta events.
	Delta string `json:"delta,omitempty"`

	// Content for text_end, thinking_end events (final content string).
	Content string `json:"content,omitempty"`

	// ToolCall for toolcall_end event.
	ToolCall *ToolCall `json:"toolCall,omitempty"`

	// Reason for done event.
	Reason StopReason `json:"reason,omitempty"`

	// Message for done event (the final complete message).
	Message *AssistantMessage `json:"message,omitempty"`

	// Error for error event (the message containing the error).
	Error *AssistantMessage `json:"error,omitempty"`
}

// --- OpenAI Compatibility ---

// OpenAICompletionsCompat holds compatibility settings for OpenAI-compatible completions APIs.
type OpenAICompletionsCompat struct {
	SupportsStore                    *bool  `json:"supportsStore,omitempty"`
	SupportsDeveloperRole            *bool  `json:"supportsDeveloperRole,omitempty"`
	SupportsReasoningEffort          *bool  `json:"supportsReasoningEffort,omitempty"`
	SupportsUsageInStreaming         *bool  `json:"supportsUsageInStreaming,omitempty"`
	MaxTokensField                   string `json:"maxTokensField,omitempty"` // "max_completion_tokens" or "max_tokens"
	RequiresToolResultName           *bool  `json:"requiresToolResultName,omitempty"`
	RequiresAssistantAfterToolResult *bool  `json:"requiresAssistantAfterToolResult,omitempty"`
	RequiresThinkingAsText           *bool  `json:"requiresThinkingAsText,omitempty"`
	RequiresMistralToolIds           *bool  `json:"requiresMistralToolIds,omitempty"`
	ThinkingFormat                   string `json:"thinkingFormat,omitempty"` // "openai", "zai", "qwen"
	SupportsStrictMode               *bool  `json:"supportsStrictMode,omitempty"`
}

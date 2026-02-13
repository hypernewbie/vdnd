package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"uaa/vdnd/internal/pi-go/ai"
)

// mockStreamFn registers a mock LLM provider that returns a predetermined response.
func setupMockProvider(t *testing.T, responseText string, withToolCall *ai.ContentBlock) {
	t.Helper()
	ai.ClearApiProviders()

	ai.RegisterApiProvider(ai.ApiProviderImpl{
		Api: "test-api",
		StreamSimple: func(ctx context.Context, model ai.Model, llmCtx ai.Context, options *ai.SimpleStreamOptions) *ai.AssistantMessageEventStream {
			stream := ai.NewAssistantMessageEventStream()
			go func() {
				content := []ai.ContentBlock{{Type: ai.ContentTypeText, Text: responseText}}
				if withToolCall != nil {
					content = append(content, *withToolCall)
				}

				msg := ai.AssistantMessage{
					Role:       ai.RoleAssistant,
					Content:    content,
					Api:        model.Api,
					Provider:   model.Provider,
					Model:      model.ID,
					Usage:      ai.Usage{Input: 100, Output: 50},
					StopReason: ai.StopReasonStop,
					Timestamp:  time.Now().UnixMilli(),
				}
				if withToolCall != nil {
					msg.StopReason = ai.StopReasonToolUse
				}

				stream.Push(ai.AssistantMessageEvent{Type: ai.EventStart, Partial: &msg})
				stream.Push(ai.AssistantMessageEvent{
					Type:    ai.EventDone,
					Reason:  msg.StopReason,
					Message: &msg,
				})
			}()
			return stream
		},
	}, "test")
}

var testModel = ai.Model{
	ID:       "test-model",
	Name:     "Test Model",
	Api:      "test-api",
	Provider: "test-provider",
}

func TestAgentLoop_BasicPrompt(t *testing.T) {
	setupMockProvider(t, "Hello!", nil)
	defer ai.ClearApiProviders()

	ctx := context.Background()
	userMsg := ai.NewUserMsg(ai.UserMessage{
		Role:      ai.RoleUser,
		Content:   []ai.ContentBlock{{Type: ai.ContentTypeText, Text: "Hi"}},
		Timestamp: time.Now().UnixMilli(),
	})

	config := LoopConfig{
		Model:        testModel,
		ConvertToLlm: DefaultConvertToLlm,
	}

	agentCtx := AgentContext{
		SystemPrompt: "You are helpful.",
		Messages:     nil,
	}

	stream := AgentLoop(ctx, []Message{userMsg}, agentCtx, config)

	var events []Event
	for event := range stream.Events() {
		events = append(events, event)
	}

	if len(events) == 0 {
		t.Fatal("expected events, got none")
	}

	// Should have: agent_start, turn_start, message_start(user), message_end(user),
	// message_start(assistant), message_end(assistant), turn_end, agent_end
	var foundAgentEnd, foundAssistantEnd bool
	for _, e := range events {
		if e.Type == EventAgentEnd {
			foundAgentEnd = true
			if len(e.Messages) < 2 {
				t.Errorf("expected at least 2 messages in agent_end, got %d", len(e.Messages))
			}
		}
		if e.Type == EventMessageEnd && e.Message != nil && e.Message.Role == ai.RoleAssistant {
			foundAssistantEnd = true
		}
	}
	if !foundAgentEnd {
		t.Error("expected agent_end event")
	}
	if !foundAssistantEnd {
		t.Error("expected assistant message_end event")
	}
}

func TestAgentLoop_WithToolCalls(t *testing.T) {
	callCount := 0
	toolCall := &ai.ContentBlock{
		Type:      ai.ContentTypeToolCall,
		ID:        "call_1",
		Name:      "echo",
		Arguments: map[string]any{"text": "world"},
	}

	// First call returns a tool call, second call returns final text
	ai.ClearApiProviders()
	ai.RegisterApiProvider(ai.ApiProviderImpl{
		Api: "test-api",
		StreamSimple: func(ctx context.Context, model ai.Model, llmCtx ai.Context, options *ai.SimpleStreamOptions) *ai.AssistantMessageEventStream {
			stream := ai.NewAssistantMessageEventStream()
			go func() {
				callCount++
				var content []ai.ContentBlock
				var stopReason ai.StopReason

				if callCount == 1 {
					content = []ai.ContentBlock{*toolCall}
					stopReason = ai.StopReasonToolUse
				} else {
					content = []ai.ContentBlock{{Type: ai.ContentTypeText, Text: "Done!"}}
					stopReason = ai.StopReasonStop
				}

				msg := ai.AssistantMessage{
					Role:       ai.RoleAssistant,
					Content:    content,
					Api:        model.Api,
					Provider:   model.Provider,
					Model:      model.ID,
					StopReason: stopReason,
					Timestamp:  time.Now().UnixMilli(),
				}

				stream.Push(ai.AssistantMessageEvent{Type: ai.EventStart, Partial: &msg})
				stream.Push(ai.AssistantMessageEvent{Type: ai.EventDone, Reason: stopReason, Message: &msg})
			}()
			return stream
		},
	}, "test")
	defer ai.ClearApiProviders()

	echoTool := Tool{
		Tool: ai.Tool{
			Name:        "echo",
			Description: "Echoes text",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text": map[string]any{"type": "string"},
				},
				"required": []any{"text"},
			},
		},
		Execute: func(ctx context.Context, id string, args map[string]any, onUpdate ToolUpdateFunc) (ToolResult, error) {
			return ToolResult{
				Content: []ai.ContentBlock{{Type: ai.ContentTypeText, Text: fmt.Sprintf("Echo: %v", args["text"])}},
			}, nil
		},
	}

	ctx := context.Background()
	agentCtx := AgentContext{
		SystemPrompt: "test",
		Tools:        []Tool{echoTool},
	}

	config := LoopConfig{
		Model:        testModel,
		ConvertToLlm: DefaultConvertToLlm,
	}

	userMsg := ai.NewUserMsg(ai.UserMessage{
		Role:      ai.RoleUser,
		Content:   []ai.ContentBlock{{Type: ai.ContentTypeText, Text: "echo world"}},
		Timestamp: time.Now().UnixMilli(),
	})

	stream := AgentLoop(ctx, []Message{userMsg}, agentCtx, config)

	var toolExecStart, toolExecEnd bool
	for event := range stream.Events() {
		if event.Type == EventToolExecutionStart {
			toolExecStart = true
			if event.ToolName != "echo" {
				t.Errorf("expected tool name 'echo', got %q", event.ToolName)
			}
		}
		if event.Type == EventToolExecutionEnd {
			toolExecEnd = true
			if event.IsError {
				t.Error("expected no error in tool result")
			}
		}
	}

	if !toolExecStart {
		t.Error("expected tool_execution_start event")
	}
	if !toolExecEnd {
		t.Error("expected tool_execution_end event")
	}
	if callCount != 2 {
		t.Errorf("expected 2 LLM calls (one for tool, one for final), got %d", callCount)
	}
}

func TestAgent_BasicPrompt(t *testing.T) {
	setupMockProvider(t, "Hello from agent!", nil)
	defer ai.ClearApiProviders()

	agent := NewAgent(AgentOptions{
		SystemPrompt: "You are helpful.",
		Model:        testModel,
	})

	var events []Event
	agent.Subscribe(func(e Event) {
		events = append(events, e)
	})

	err := agent.Prompt(context.Background(), "Hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msgs := agent.Messages()
	// Should have user + assistant
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(msgs))
	}

	if len(events) == 0 {
		t.Error("expected events from subscription")
	}

	if agent.IsStreaming() {
		t.Error("expected agent to not be streaming after prompt completes")
	}
}

func TestAgent_QueueSteering(t *testing.T) {
	agent := NewAgent(AgentOptions{
		Model: testModel,
	})

	steerMsg := ai.NewUserMsg(ai.UserMessage{
		Role:      ai.RoleUser,
		Content:   []ai.ContentBlock{{Type: ai.ContentTypeText, Text: "stop"}},
		Timestamp: time.Now().UnixMilli(),
	})

	agent.Steer(steerMsg)
	if !agent.HasQueuedMessages() {
		t.Error("expected queued messages")
	}

	agent.ClearAllQueues()
	if agent.HasQueuedMessages() {
		t.Error("expected no queued messages after clear")
	}
}

func TestDefaultConvertToLlm(t *testing.T) {
	messages := []Message{
		ai.NewUserMsg(ai.UserMessage{Role: ai.RoleUser}),
		ai.NewAssistantMsg(ai.AssistantMessage{Role: ai.RoleAssistant}),
		ai.NewToolResultMsg(ai.ToolResultMessage{Role: ai.RoleToolResult}),
	}

	result, err := DefaultConvertToLlm(messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}
}

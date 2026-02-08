package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"uaa/vdnd/internal/llm/llmtypes"
)

func TestAnthropicProvider_GenerateWithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("Expected x-api-key header, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("Expected anthropic-version header, got %s", r.Header.Get("anthropic-version"))
		}

		var req AnthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify system prompt conversion
		systemBlocks, ok := req.System.([]any)
		if !ok || len(systemBlocks) == 0 {
			t.Errorf("Expected system blocks array, got %v", req.System)
		} else {
			firstBlock := systemBlocks[0].(map[string]any)
			if firstBlock["text"] != "System prompt" {
				t.Errorf("Expected system prompt 'System prompt', got %q", firstBlock["text"])
			}
			if firstBlock["cache_control"] == nil {
				t.Error("Expected cache_control in system block")
			}
		}

		if r.Header.Get("anthropic-beta") != "prompt-caching-2024-07-31" {
			t.Errorf("Expected anthropic-beta header, got %s", r.Header.Get("anthropic-beta"))
		}
		if len(req.Tools) != 1 || req.Tools[0].Name != "test_tool" {
			t.Errorf("Expected 1 tool named 'test_tool', got %v", req.Tools)
		}

		resp := AnthropicResponse{
			ID:    "msg_1",
			Type:  "message",
			Role:  "assistant",
			Model: "claude-3",
			Content: []AnthropicContentBlock{
				{
					Type: "text",
					Text: "Thinking...",
				},
				{
					Type: "tool_use",
					ID:   "tool_1",
					Name: "test_tool",
					Input: map[string]any{
						"arg1": "val1",
					},
				},
			},
			StopReason: "tool_use",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p, _ := NewAnthropicProvider("test-key", "claude-3")
	// Override client and URL to use test server
	p.client = server.Client()
	p.baseURL = server.URL

	resp, err := p.GenerateWithTools(context.Background(), []llmtypes.Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "Hello"},
	}, []llmtypes.Tool{
		{Name: "test_tool", Description: "desc", Parameters: map[string]any{"type": "object"}},
	})

	if err != nil {
		t.Fatalf("GenerateWithTools failed: %v", err)
	}

	if resp.FinishReason != "tool_calls" {
		t.Errorf("Expected finish reason 'tool_calls', got %q", resp.FinishReason)
	}

	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "test_tool" {
		t.Errorf("Expected 1 tool call for 'test_tool', got %v", resp.ToolCalls)
	}
}

func TestAnthropicProvider_MessageConversion(t *testing.T) {
	p, _ := NewAnthropicProvider("test-key", "claude-3")

	messages := []llmtypes.Message{
		{Role: "user", Content: "Hello"},
		{Role: "model", Content: "Hi there"},
		{Role: "user", Content: "How are you?"},
	}

	anthropicMsgs, system := p.convertMessages(messages)
	if system != "" {
		t.Errorf("Expected empty system prompt, got %q", system)
	}
	if len(anthropicMsgs) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(anthropicMsgs))
	}
	if anthropicMsgs[0].Role != "user" || anthropicMsgs[0].Content[0].Text != "Hello" {
		t.Errorf("First message mismatch: %+v", anthropicMsgs[0])
	}
	if anthropicMsgs[1].Role != "assistant" || anthropicMsgs[1].Content[0].Text != "Hi there" {
		t.Errorf("Second message mismatch: %+v", anthropicMsgs[1])
	}
}

func TestAnthropicProvider_MergingConsecutiveMessages(t *testing.T) {
	p, _ := NewAnthropicProvider("test-key", "claude-3")

	messages := []llmtypes.Message{
		{Role: "user", Content: "Hello"},
		{Role: "user", Content: "World"},
	}

	anthropicMsgs, _ := p.convertMessages(messages)
	if len(anthropicMsgs) != 1 {
		t.Errorf("Expected 1 merged message, got %d", len(anthropicMsgs))
	}
	if len(anthropicMsgs[0].Content) != 2 {
		t.Errorf("Expected 2 content blocks, got %d", len(anthropicMsgs[0].Content))
	}
}

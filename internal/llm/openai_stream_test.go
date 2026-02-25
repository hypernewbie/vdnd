package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"uaa/vdnd/internal/llm/llmtypes"
)

func TestOpenAIProvider_GenerateStreamText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OpenAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !req.Stream {
			t.Fatalf("expected stream=true request")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello \"},\"finish_reason\":\"\"}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"world\"},\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	p := NewOpenAIProvider(OpenAIProviderConfig{
		Name:          "test",
		BaseURL:       server.URL,
		APIKey:        "x",
		Model:         "test-model",
		SupportsTools: true,
	})
	p.client = server.Client()

	var streamed strings.Builder
	resp, err := p.GenerateStream(context.Background(), []llmtypes.Message{{Role: "user", Content: "hello"}}, nil, func(chunk string) error {
		streamed.WriteString(chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("GenerateStream failed: %v", err)
	}
	if resp.FinishReason != "stop" {
		t.Fatalf("expected stop finish reason, got %q", resp.FinishReason)
	}
	if streamed.String() != "Hello world" {
		t.Fatalf("unexpected streamed output: %q", streamed.String())
	}
	if resp.Content != "Hello world" {
		t.Fatalf("unexpected response content: %q", resp.Content)
	}
}

func TestOpenAIProvider_GenerateStreamToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"do_it\",\"arguments\":\"{\\\"a\\\":\"}}]},\"finish_reason\":\"\"}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"1}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	p := NewOpenAIProvider(OpenAIProviderConfig{
		Name:          "test",
		BaseURL:       server.URL,
		APIKey:        "x",
		Model:         "test-model",
		SupportsTools: true,
	})
	p.client = server.Client()

	resp, err := p.GenerateStream(context.Background(), []llmtypes.Message{{Role: "user", Content: "tool"}}, nil, nil)
	if err != nil {
		t.Fatalf("GenerateStream failed: %v", err)
	}
	if resp.FinishReason != "tool_calls" {
		t.Fatalf("expected tool_calls finish reason, got %q", resp.FinishReason)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "do_it" {
		t.Fatalf("unexpected tool name: %q", resp.ToolCalls[0].Name)
	}
	if resp.ToolCalls[0].Arguments != `{"a":1}` {
		t.Fatalf("unexpected tool args: %q", resp.ToolCalls[0].Arguments)
	}
}

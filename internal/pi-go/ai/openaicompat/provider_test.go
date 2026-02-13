package openaicompat

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"uaa/vdnd/internal/pi-go/ai"
)

func TestDeepSeekStream(t *testing.T) {
	// Mock DeepSeek API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check URL
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Expected path /v1/chat/completions, got %s", r.URL.Path)
		}

		// Check auth
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}

		// Stream response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("Expected http.Flusher")
		}

		// Send chunks
		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"deepseek-chat","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"deepseek-chat","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"deepseek-chat","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"deepseek-chat","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`[DONE]`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
	}))
	defer ts.Close()

	// Configure model to use mock server
	model := ai.Model{
		ID:       "deepseek-chat",
		Provider: "deepseek",
		Api:      ai.ApiOpenAICompletions,
		BaseURL:  ts.URL, // Use mock server URL (which has http:// scheme, provider adds /v1)
	}

	ctx := context.Background()
	llmCtx := ai.Context{
		Messages: []ai.Message{
			{Role: ai.RoleUser, User: &ai.UserMessage{Content: []ai.ContentBlock{{Type: ai.ContentTypeText, Text: "Hi"}}}},
		},
	}
	opts := &ai.SimpleStreamOptions{
		StreamOptions: ai.StreamOptions{
			APIKey: "test-key",
		},
	}

	stream := streamSimpleFn(ctx, model, llmCtx, opts)

	// Collect result
	var fullText string
	for event := range stream.Events() {
		switch event.Type {
		case ai.EventTextDelta:
			fullText += event.Delta
		case ai.EventError:
			t.Fatalf("Stream error: %v", event.Error.ErrorMessage)
		}
	}

	result, err := stream.Result(ctx)
	if err != nil {
		t.Fatalf("Result error: %v", err)
	}

	if fullText != "Hello world" {
		t.Errorf("Expected 'Hello world', got %q", fullText)
	}

	if result.Content[0].Text != "Hello world" {
		t.Errorf("Expected result text 'Hello world', got %q", result.Content[0].Text)
	}
}

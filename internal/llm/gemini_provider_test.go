package llm

import (
	"encoding/json"
	"testing"
)

func TestGeminiProvider_MessageConversion(t *testing.T) {
	p := &GeminiProvider{model: "gemini-2.0-flash-exp"}

	messages := []Message{
		{Role: "user", Content: "Call the tool"},
		{
			Role:     "model",
			Thinking: "Thinking about the tool...",
			ToolCalls: []ToolCall{
				{
					Name:             "test_tool",
					Arguments:        `{"arg1": "val1"}`,
					ThoughtSignature: "sig123",
				},
			},
		},
		{
			Role:       "tool",
			Name:       "test_tool",
			Content:    "result123",
			ToolCallID: "", // Gemini doesn't use IDs
		},
	}

	contents := p.convertMessagesToGeminiContents(messages)

	if len(contents) != 3 {
		t.Fatalf("Expected 3 Gemini contents, got %d", len(contents))
	}

	// Check model turn for thinking part
	modelContent := contents[1]
	if modelContent.Role != "model" {
		t.Errorf("Expected role 'model', got '%s'", modelContent.Role)
	}

	foundThinking := false
	foundToolCall := false
	for _, part := range modelContent.Parts {
		if part.Thought {
			foundThinking = true
			if part.Text != "Thinking about the tool..." {
				t.Errorf("Unexpected thinking text: %s", part.Text)
			}
		}
		if part.FunctionCall != nil {
			foundToolCall = true
			if part.ThoughtSignature != "sig123" {
				t.Errorf("Expected ThoughtSignature (Part) 'sig123', got '%s'", part.ThoughtSignature)
			}
		}
	}
	if !foundThinking {
		t.Error("Did not find thinking part")
	}
	if !foundToolCall {
		t.Error("Did not find tool call part")
	}
}

func TestGeminiProvider_ResponseParsing(t *testing.T) {
	// Mock response with thought_signature at Part level (Correct location)
	rawRespPart := `{
		"candidates": [
			{
				"content": {
					"role": "model",
					"parts": [
						{
							"thought_signature": "sig456",
							"functionCall": {
								"name": "test_tool",
								"args": {"arg1": "val1"}
							}
						}
					]
				},
				"finishReason": "STOP"
			}
		]
	}`
	checkSig(t, rawRespPart, "sig456")

	// Mock response with thoughtSignature (camelCase) at Part level
	rawRespPartCamel := `{
		"candidates": [
			{
				"content": {
					"role": "model",
					"parts": [
						{
							"thoughtSignature": "sig789",
							"functionCall": {
								"name": "test_tool",
								"args": {"arg1": "val1"}
							}
						}
					]
				},
				"finishReason": "STOP"
			}
		]
	}`
	checkSig(t, rawRespPartCamel, "sig789")
}

func checkSig(t *testing.T, jsonStr, expectedSig string) {
	var geminiResp GeminiResponse
	if err := json.Unmarshal([]byte(jsonStr), &geminiResp); err != nil {
		t.Fatalf("Failed to unmarshal mock response: %v", err)
	}

	if len(geminiResp.Candidates) == 0 {
		t.Fatal("No candidates in response")
	}
	candidate := geminiResp.Candidates[0]

	count := 0
	for _, part := range candidate.Content.Parts {
		// Simulate the logic in GenerateWithTools
		if part.FunctionCall != nil {
			count++
			sig := part.ThoughtSignature
			if sig == "" {
				sig = part.ThoughtSignatureCamel
			}

			if sig != expectedSig {
				t.Errorf("Expected ThoughtSignature '%s', got '%s'", expectedSig, sig)
			}
		}
	}
	if count != 1 {
		t.Errorf("Expected 1 function call, got %d", count)
	}
}

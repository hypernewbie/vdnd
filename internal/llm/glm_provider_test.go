package llm

import (
	"testing"
)

func TestNewGLMProvider(t *testing.T) {
	apiKey := "test-api-key"
	model := "glm-4.7"
	p, err := NewGLMProvider(apiKey, model)
	if err != nil {
		t.Fatalf("NewGLMProvider failed: %v", err)
	}

	if p.Name() != "glm" {
		t.Errorf("Expected provider name 'glm', got '%s'", p.Name())
	}

	if p.ModelName() != model {
		t.Errorf("Expected model name '%s', got '%s'", model, p.ModelName())
	}

	if p.config.BaseURL != "https://api.z.ai/api/coding/paas/v4/chat/completions" {
		t.Errorf("Unexpected BaseURL: %s", p.config.BaseURL)
	}

	if !p.SupportsToolCalling() {
		t.Errorf("GLM provider should support tool calling")
	}
}

func TestNewGLMProviderDefaultModel(t *testing.T) {
	apiKey := "test-api-key"
	p, err := NewGLMProvider(apiKey, "")
	if err != nil {
		t.Fatalf("NewGLMProvider failed: %v", err)
	}

	if p.ModelName() != "glm-4.7" {
		t.Errorf("Expected default model name 'glm-4.7', got '%s'", p.ModelName())
	}
}

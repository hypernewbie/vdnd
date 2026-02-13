package ai

import (
	"testing"
)

func TestProviderRegistry(t *testing.T) {
	// Clean state
	ClearApiProviders()
	defer ClearApiProviders()

	// Should return nil for unregistered API
	if p := GetApiProvider("nonexistent"); p != nil {
		t.Fatal("expected nil for unregistered provider")
	}

	// Register a dummy provider
	dummy := ApiProviderImpl{
		Api:      "test-api",
		StreamFn: nil, // not needed for registry tests
	}
	RegisterApiProvider(dummy, "test-source")

	// Should find it
	p := GetApiProvider("test-api")
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Api != "test-api" {
		t.Fatalf("expected api 'test-api', got %q", p.Api)
	}

	// GetApiProviders should return it
	all := GetApiProviders()
	if len(all) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(all))
	}

	// Unregister by source ID
	UnregisterApiProviders("test-source")
	if p := GetApiProvider("test-api"); p != nil {
		t.Fatal("expected nil after unregister")
	}
}

func TestModelRegistry(t *testing.T) {
	m := Model{
		ID:            "test-model-1",
		Name:          "Test Model 1",
		Api:           "test-api",
		Provider:      "test-provider",
		ContextWindow: 128000,
		MaxTokens:     4096,
		Cost: ModelCost{
			Input:  3.0,
			Output: 15.0,
		},
	}

	RegisterModel(m)

	// Find it
	found, ok := GetModel("test-provider", "test-model-1")
	if !ok {
		t.Fatal("expected to find model")
	}
	if found.Name != "Test Model 1" {
		t.Fatalf("expected name 'Test Model 1', got %q", found.Name)
	}

	// Not found
	_, ok = GetModel("test-provider", "nonexistent")
	if ok {
		t.Fatal("expected not found")
	}

	// GetProviders
	providers := GetProviders()
	foundProvider := false
	for _, p := range providers {
		if p == "test-provider" {
			foundProvider = true
		}
	}
	if !foundProvider {
		t.Fatal("expected to find 'test-provider' in providers list")
	}

	// GetModels
	models := GetModels("test-provider")
	if len(models) < 1 {
		t.Fatal("expected at least 1 model for test-provider")
	}
}

func TestCalculateCost(t *testing.T) {
	model := Model{
		Cost: ModelCost{
			Input:      3.0,  // $3/M tokens
			Output:     15.0, // $15/M tokens
			CacheRead:  0.3,
			CacheWrite: 3.75,
		},
	}

	usage := &Usage{
		Input:      1000,
		Output:     500,
		CacheRead:  200,
		CacheWrite: 100,
	}

	cost := CalculateCost(model, usage)

	// input: 3.0/1M * 1000 = 0.003
	if cost.Input < 0.002999 || cost.Input > 0.003001 {
		t.Errorf("expected input cost ~0.003, got %f", cost.Input)
	}
	// output: 15.0/1M * 500 = 0.0075
	if cost.Output < 0.007499 || cost.Output > 0.007501 {
		t.Errorf("expected output cost ~0.0075, got %f", cost.Output)
	}
	// total should be sum of all
	expectedTotal := cost.Input + cost.Output + cost.CacheRead + cost.CacheWrite
	if cost.Total < expectedTotal-0.000001 || cost.Total > expectedTotal+0.000001 {
		t.Errorf("expected total %f, got %f", expectedTotal, cost.Total)
	}
}

func TestModelsAreEqual(t *testing.T) {
	a := &Model{ID: "m1", Provider: "p1"}
	b := &Model{ID: "m1", Provider: "p1"}
	c := &Model{ID: "m2", Provider: "p1"}

	if !ModelsAreEqual(a, b) {
		t.Error("expected a == b")
	}
	if ModelsAreEqual(a, c) {
		t.Error("expected a != c")
	}
	if ModelsAreEqual(a, nil) {
		t.Error("expected a != nil")
	}
	if ModelsAreEqual(nil, nil) {
		t.Error("expected nil != nil")
	}
}

func TestSupportsXhigh(t *testing.T) {
	if !SupportsXhigh(Model{ID: "gpt-5.2-turbo"}) {
		t.Error("expected gpt-5.2 to support xhigh")
	}
	if !SupportsXhigh(Model{ID: "gpt-5.3-codex"}) {
		t.Error("expected gpt-5.3 to support xhigh")
	}
	if !SupportsXhigh(Model{ID: "claude-opus-4.6", Api: ApiAnthropicMessages}) {
		t.Error("expected opus-4.6 to support xhigh")
	}
	if SupportsXhigh(Model{ID: "gpt-4o"}) {
		t.Error("expected gpt-4o to not support xhigh")
	}
}

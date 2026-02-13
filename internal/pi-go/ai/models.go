package ai

import (
	"strings"
	"sync"
)

// modelRegistry is a nested map: provider -> modelID -> Model.
var (
	modelsMu      sync.RWMutex
	modelRegistry = make(map[Provider]map[string]Model)
)

// RegisterModel adds a model to the registry.
func RegisterModel(model Model) {
	modelsMu.Lock()
	defer modelsMu.Unlock()
	pm, ok := modelRegistry[model.Provider]
	if !ok {
		pm = make(map[string]Model)
		modelRegistry[model.Provider] = pm
	}
	pm[model.ID] = model
}

// RegisterModels adds multiple models to the registry.
func RegisterModels(models ...Model) {
	for _, m := range models {
		RegisterModel(m)
	}
}

// GetModel returns a model by provider and model ID, or (Model{}, false) if not found.
func GetModel(provider Provider, modelID string) (Model, bool) {
	modelsMu.RLock()
	defer modelsMu.RUnlock()
	pm, ok := modelRegistry[provider]
	if !ok {
		return Model{}, false
	}
	m, ok := pm[modelID]
	return m, ok
}

// GetProviders returns all providers that have registered models.
func GetProviders() []Provider {
	modelsMu.RLock()
	defer modelsMu.RUnlock()
	providers := make([]Provider, 0, len(modelRegistry))
	for p := range modelRegistry {
		providers = append(providers, p)
	}
	return providers
}

// GetModels returns all models registered for a specific provider.
func GetModels(provider Provider) []Model {
	modelsMu.RLock()
	defer modelsMu.RUnlock()
	pm, ok := modelRegistry[provider]
	if !ok {
		return nil
	}
	models := make([]Model, 0, len(pm))
	for _, m := range pm {
		models = append(models, m)
	}
	return models
}

// CalculateCost computes the cost breakdown for a model and usage, updating usage.Cost in place.
func CalculateCost(model Model, usage *Usage) CostBreakdown {
	usage.Cost.Input = (model.Cost.Input / 1_000_000) * float64(usage.Input)
	usage.Cost.Output = (model.Cost.Output / 1_000_000) * float64(usage.Output)
	usage.Cost.CacheRead = (model.Cost.CacheRead / 1_000_000) * float64(usage.CacheRead)
	usage.Cost.CacheWrite = (model.Cost.CacheWrite / 1_000_000) * float64(usage.CacheWrite)
	usage.Cost.Total = usage.Cost.Input + usage.Cost.Output + usage.Cost.CacheRead + usage.Cost.CacheWrite
	return usage.Cost
}

// SupportsXhigh checks if a model supports the "xhigh" thinking level.
func SupportsXhigh(model Model) bool {
	if strings.Contains(model.ID, "gpt-5.2") || strings.Contains(model.ID, "gpt-5.3") {
		return true
	}
	if model.Api == ApiAnthropicMessages {
		return strings.Contains(model.ID, "opus-4-6") || strings.Contains(model.ID, "opus-4.6")
	}
	return false
}

// ModelsAreEqual checks if two models have the same ID and provider.
func ModelsAreEqual(a, b *Model) bool {
	if a == nil || b == nil {
		return false
	}
	return a.ID == b.ID && a.Provider == b.Provider
}

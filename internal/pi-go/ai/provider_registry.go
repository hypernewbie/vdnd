package ai

import (
	"context"
	"fmt"
	"sync"
)

// StreamFunc is the function signature for streaming an LLM response.
type StreamFunc func(
	ctx context.Context,
	model Model,
	llmCtx Context,
	options *StreamOptions,
) *AssistantMessageEventStream

// SimpleStreamFunc is the function signature for streaming with simplified options.
type SimpleStreamFunc func(
	ctx context.Context,
	model Model,
	llmCtx Context,
	options *SimpleStreamOptions,
) *AssistantMessageEventStream

// ApiProviderImpl implements streaming for a specific API protocol.
type ApiProviderImpl struct {
	Api          Api
	StreamFn     StreamFunc
	StreamSimple SimpleStreamFunc
}

type registeredProvider struct {
	provider ApiProviderImpl
	sourceID string
}

var (
	registryMu sync.RWMutex
	registry   = make(map[Api]*registeredProvider)
)

// RegisterApiProvider registers a provider implementation for an API identifier.
// An optional sourceID can be provided for bulk unregistration.
func RegisterApiProvider(provider ApiProviderImpl, sourceID string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[provider.Api] = &registeredProvider{
		provider: provider,
		sourceID: sourceID,
	}
}

// GetApiProvider returns the provider implementation for the given API, or nil if not registered.
func GetApiProvider(api Api) *ApiProviderImpl {
	registryMu.RLock()
	defer registryMu.RUnlock()
	entry := registry[api]
	if entry == nil {
		return nil
	}
	return &entry.provider
}

// GetApiProviders returns all registered provider implementations.
func GetApiProviders() []ApiProviderImpl {
	registryMu.RLock()
	defer registryMu.RUnlock()
	providers := make([]ApiProviderImpl, 0, len(registry))
	for _, entry := range registry {
		providers = append(providers, entry.provider)
	}
	return providers
}

// UnregisterApiProviders removes all providers registered with the given sourceID.
func UnregisterApiProviders(sourceID string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	for api, entry := range registry {
		if entry.sourceID == sourceID {
			delete(registry, api)
		}
	}
}

// ClearApiProviders removes all registered providers.
func ClearApiProviders() {
	registryMu.Lock()
	defer registryMu.Unlock()
	clear(registry)
}

// resolveApiProvider looks up a provider by API or returns an error.
func resolveApiProvider(api Api) (*ApiProviderImpl, error) {
	p := GetApiProvider(api)
	if p == nil {
		return nil, fmt.Errorf("no API provider registered for api: %s", api)
	}
	return p, nil
}

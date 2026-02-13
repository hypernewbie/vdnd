package ai

import (
	"context"
)

// Stream starts streaming an LLM response for the given model and context.
// Returns an AssistantMessageEventStream that can be iterated for events.
func Stream(
	ctx context.Context,
	model Model,
	llmCtx Context,
	options *StreamOptions,
) (*AssistantMessageEventStream, error) {
	provider, err := resolveApiProvider(model.Api)
	if err != nil {
		return nil, err
	}
	return provider.StreamFn(ctx, model, llmCtx, options), nil
}

// Complete streams an LLM response and returns only the final AssistantMessage.
func Complete(
	ctx context.Context,
	model Model,
	llmCtx Context,
	options *StreamOptions,
) (AssistantMessage, error) {
	stream, err := Stream(ctx, model, llmCtx, options)
	if err != nil {
		return AssistantMessage{}, err
	}
	return stream.Result(ctx)
}

// StreamSimple starts streaming with simplified options (including reasoning level).
func StreamSimple(
	ctx context.Context,
	model Model,
	llmCtx Context,
	options *SimpleStreamOptions,
) (*AssistantMessageEventStream, error) {
	provider, err := resolveApiProvider(model.Api)
	if err != nil {
		return nil, err
	}
	return provider.StreamSimple(ctx, model, llmCtx, options), nil
}

// CompleteSimple streams with simplified options and returns only the final message.
func CompleteSimple(
	ctx context.Context,
	model Model,
	llmCtx Context,
	options *SimpleStreamOptions,
) (AssistantMessage, error) {
	stream, err := StreamSimple(ctx, model, llmCtx, options)
	if err != nil {
		return AssistantMessage{}, err
	}
	return stream.Result(ctx)
}

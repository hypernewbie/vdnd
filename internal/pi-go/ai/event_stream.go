package ai

import (
	"context"
	"sync"
)

// EventStream is a channel-based async event stream.
// It allows a producer to push events and a consumer to iterate over them.
type EventStream[T any, R any] struct {
	events chan T
	result chan R

	mu            sync.Mutex
	done          bool
	isComplete    func(T) bool
	extractResult func(T) R
}

// NewEventStream creates a new EventStream.
// isComplete checks if an event signals stream completion.
// extractResult extracts the final result from a terminal event.
func NewEventStream[T any, R any](
	isComplete func(T) bool,
	extractResult func(T) R,
) *EventStream[T, R] {
	return &EventStream[T, R]{
		events:        make(chan T, 64),
		result:        make(chan R, 1),
		isComplete:    isComplete,
		extractResult: extractResult,
	}
}

// Push sends an event to the stream. If the event is terminal (isComplete returns true),
// the result is extracted and the stream is closed.
func (s *EventStream[T, R]) Push(event T) {
	s.mu.Lock()
	if s.done {
		s.mu.Unlock()
		return
	}

	if s.isComplete(event) {
		s.done = true
		r := s.extractResult(event)
		s.mu.Unlock()
		s.events <- event
		s.result <- r
		close(s.events)
		return
	}
	s.mu.Unlock()

	s.events <- event
}

// End signals the stream is complete with an explicit result.
// Use this when termination happens outside of a normal event flow.
func (s *EventStream[T, R]) End(result R) {
	s.mu.Lock()
	if s.done {
		s.mu.Unlock()
		return
	}
	s.done = true
	s.mu.Unlock()

	s.result <- result
	close(s.events)
}

// Events returns a channel for iterating over events.
// The channel is closed when the stream is done.
func (s *EventStream[T, R]) Events() <-chan T {
	return s.events
}

// Result blocks until the stream completes and returns the final result.
// Respects context cancellation.
func (s *EventStream[T, R]) Result(ctx context.Context) (R, error) {
	select {
	case r := <-s.result:
		return r, nil
	case <-ctx.Done():
		var zero R
		return zero, ctx.Err()
	}
}

// AssistantMessageEventStream is a specialized EventStream for LLM assistant responses.
type AssistantMessageEventStream = EventStream[AssistantMessageEvent, AssistantMessage]

// NewAssistantMessageEventStream creates a new stream for assistant message events.
func NewAssistantMessageEventStream() *AssistantMessageEventStream {
	return NewEventStream[AssistantMessageEvent, AssistantMessage](
		func(event AssistantMessageEvent) bool {
			return event.Type == EventDone || event.Type == EventError
		},
		func(event AssistantMessageEvent) AssistantMessage {
			if event.Type == EventDone && event.Message != nil {
				return *event.Message
			}
			if event.Type == EventError && event.Error != nil {
				return *event.Error
			}
			return AssistantMessage{}
		},
	)
}

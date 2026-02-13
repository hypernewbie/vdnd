package ai

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestEventStream_PushAndIterate(t *testing.T) {
	stream := NewEventStream[int, int](
		func(v int) bool { return v == -1 },
		func(v int) int { return v },
	)

	// Push events in a goroutine
	go func() {
		for i := 0; i < 5; i++ {
			stream.Push(i)
		}
		stream.Push(-1) // terminal event
	}()

	var received []int
	for event := range stream.Events() {
		received = append(received, event)
	}

	if len(received) != 6 {
		t.Fatalf("expected 6 events (0-4 + terminal), got %d: %v", len(received), received)
	}
	for i := 0; i < 5; i++ {
		if received[i] != i {
			t.Errorf("event %d: expected %d, got %d", i, i, received[i])
		}
	}
	if received[5] != -1 {
		t.Errorf("terminal event: expected -1, got %d", received[5])
	}
}

func TestEventStream_Result(t *testing.T) {
	stream := NewEventStream[string, string](
		func(v string) bool { return v == "done" },
		func(v string) string { return v },
	)

	go func() {
		stream.Push("first")
		stream.Push("second")
		stream.Push("done")
	}()

	ctx := context.Background()
	result, err := stream.Result(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "done" {
		t.Fatalf("expected result 'done', got %q", result)
	}
}

func TestEventStream_ResultCancellation(t *testing.T) {
	stream := NewEventStream[int, int](
		func(v int) bool { return v == -1 },
		func(v int) int { return v },
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Don't push any terminal event — should time out
	_, err := stream.Result(ctx)
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
}

func TestEventStream_End(t *testing.T) {
	stream := NewEventStream[int, int](
		func(v int) bool { return false }, // never completes via push
		func(v int) int { return v },
	)

	go func() {
		stream.Push(1)
		stream.Push(2)
		stream.End(42)
	}()

	ctx := context.Background()
	result, err := stream.Result(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Fatalf("expected result 42, got %d", result)
	}
}

func TestEventStream_ConcurrentPush(t *testing.T) {
	stream := NewEventStream[int, int](
		func(v int) bool { return v == 999 },
		func(v int) int { return v },
	)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				stream.Push(start*10 + j)
			}
		}(i)
	}

	// Drain in parallel to avoid blocking
	go func() {
		for range stream.Events() {
		}
	}()

	wg.Wait()
	stream.Push(999) // terminal

	ctx := context.Background()
	result, err := stream.Result(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 999 {
		t.Fatalf("expected result 999, got %d", result)
	}
}

func TestAssistantMessageEventStream(t *testing.T) {
	stream := NewAssistantMessageEventStream()

	finalMsg := AssistantMessage{
		Role:       RoleAssistant,
		Model:      "test-model",
		StopReason: StopReasonStop,
	}

	go func() {
		stream.Push(AssistantMessageEvent{
			Type:    EventStart,
			Partial: &AssistantMessage{Role: RoleAssistant},
		})
		stream.Push(AssistantMessageEvent{
			Type:    EventTextDelta,
			Delta:   "Hello",
			Partial: &AssistantMessage{Role: RoleAssistant},
		})
		stream.Push(AssistantMessageEvent{
			Type:    EventDone,
			Reason:  StopReasonStop,
			Message: &finalMsg,
		})
	}()

	ctx := context.Background()
	result, err := stream.Result(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Model != "test-model" {
		t.Fatalf("expected model 'test-model', got %q", result.Model)
	}
	if result.StopReason != StopReasonStop {
		t.Fatalf("expected stop reason 'stop', got %q", result.StopReason)
	}
}

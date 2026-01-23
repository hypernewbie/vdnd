package encounter

import (
	"uaa/vdnd/pkg/rules/ability"
)

type EventHandler func(event ability.Event, encounter *Encounter) bool // Returns true if handled

type EventQueue struct {
	events   []ability.Event
	handlers map[ability.EventType][]EventHandler
}

func NewEventQueue() *EventQueue {
	return &EventQueue{
		events:   make([]ability.Event, 0),
		handlers: make(map[ability.EventType][]EventHandler),
	}
}

// Emit adds an event to the queue
func (q *EventQueue) Emit(event ability.Event) {
	q.events = append(q.events, event)
}

// Process handles all queued events, checking for reactions
func (q *EventQueue) Process(encounter *Encounter) {
	// Snapshot current events and clear queue so new events (re-entrance) aren't lost/wiped
	processing := q.events
	q.events = make([]ability.Event, 0)

	for _, event := range processing {
		if handlers, ok := q.handlers[event.Type]; ok {
			for _, handler := range handlers {
				if handler(event, encounter) {
					// Event handled
				}
			}
		}
	}
}

// RegisterHandler adds a handler for an event type (for reactions)
func (q *EventQueue) RegisterHandler(eventType ability.EventType, handler EventHandler) {
	q.handlers[eventType] = append(q.handlers[eventType], handler)
}
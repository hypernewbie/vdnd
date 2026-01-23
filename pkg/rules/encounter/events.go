package encounter

import (
	"uaa/vdnd/pkg/rules/entity"
)

type EventType int

const (
	EventMove        EventType = iota // Entity moved
	EventManipulate                   // Manipulate action used
	EventConcentrate                  // Concentrate action used
	EventStrike                       // Strike made
	EventCast                         // Spell cast
	EventDamaged                      // Entity took damage
)

type Event struct {
	Type     EventType
	Actor    *entity.Entity
	Target   *entity.Entity
	Position string // For movement events
	Details  map[string]interface{}
}

type EventHandler func(event Event, encounter *Encounter) bool // Returns true if handled

type EventQueue struct {
	events   []Event
	handlers map[EventType][]EventHandler
}

func NewEventQueue() *EventQueue {
	return &EventQueue{
		events:   make([]Event, 0),
		handlers: make(map[EventType][]EventHandler),
	}
}

// Emit adds an event to the queue
func (q *EventQueue) Emit(event Event) {
	q.events = append(q.events, event)
}

// Process handles all queued events, checking for reactions
func (q *EventQueue) Process(encounter *Encounter) {
	// Snapshot current events and clear queue so new events (re-entrance) aren't lost/wiped
	processing := q.events
	q.events = make([]Event, 0)

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
func (q *EventQueue) RegisterHandler(eventType EventType, handler EventHandler) {
	q.handlers[eventType] = append(q.handlers[eventType], handler)
}

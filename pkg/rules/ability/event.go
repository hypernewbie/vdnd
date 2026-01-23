package ability

type EventType int

const (
	EventMove        EventType = iota // Entity moved
	EventManipulate                   // Manipulate action used
	EventConcentrate                  // Concentrate action used
	EventStrike                       // Strike made
	EventCast                         // Spell cast
	EventDamaged                      // Entity took damage
)

// EventActor is a minimal interface for entities in events
type EventActor interface {
	GetID() string
	GetName() string
}

type Event struct {
	Type     EventType
	Actor    EventActor
	Target   EventActor
	Position string // For movement events
	Details  map[string]interface{}
}

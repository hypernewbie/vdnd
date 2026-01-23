package encounter

import (
	"errors"
	"uaa/vdnd/pkg/rules/combat"
	"uaa/vdnd/pkg/rules/entity"
)

type EncounterState int

const (
	StateNotStarted EncounterState = iota
	StateRollingInitiative
	StateInProgress
	StateEnded
)

type Participant struct {
	Entity     *entity.Entity
	Initiative int
	HasActed   bool // This round
	IsDelaying bool
	TurnState  *combat.TurnState
}

type Encounter struct {
	ID           string
	State        EncounterState
	Participants []*Participant
	CurrentRound int
	CurrentTurn  int // Index into sorted initiative order

	// Event system for reactions
	EventQueue *EventQueue
}

func NewEncounter(id string) *Encounter {
	return &Encounter{
		ID:           id,
		State:        StateNotStarted,
		Participants: make([]*Participant, 0),
		EventQueue:   NewEventQueue(),
		CurrentRound: 0,
		CurrentTurn:  -1,
	}
}

// AddParticipant adds an entity to the encounter
func (e *Encounter) AddParticipant(ent *entity.Entity) {
	e.Participants = append(e.Participants, &Participant{
		Entity: ent,
	})
}

// RemoveParticipant removes an entity (fled, died, etc.)
func (e *Encounter) RemoveParticipant(entityID string) {
	for i, p := range e.Participants {
		if p.Entity.ID == entityID {
			// Remove from slice
			e.Participants = append(e.Participants[:i], e.Participants[i+1:]...)

			// Adjust CurrentTurn if necessary
			if e.CurrentTurn >= i {
				e.CurrentTurn--
			}
			break
		}
	}
}

// GetCurrentParticipant returns whose turn it is
func (e *Encounter) GetCurrentParticipant() *Participant {
	if e.CurrentTurn < 0 || e.CurrentTurn >= len(e.Participants) {
		return nil
	}
	return e.Participants[e.CurrentTurn]
}

// EndEncounter marks the encounter as finished
func (e *Encounter) EndEncounter() {
	e.State = StateEnded
}

// IsOver returns true if encounter has ended
func (e *Encounter) IsOver() bool {
	return e.State == StateEnded
}

// Start begins the encounter (assuming initiative is already rolled or should be rolled)
func (e *Encounter) Start() error {
	if len(e.Participants) == 0 {
		return errors.New("cannot start encounter with no participants")
	}
	if e.State != StateNotStarted && e.State != StateRollingInitiative {
		return errors.New("encounter already started or ended")
	}

	e.SortByInitiative()
	e.State = StateInProgress
	e.CurrentRound = 1
	e.CurrentTurn = 0
	return nil
}

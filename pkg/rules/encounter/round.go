package encounter

import (
	"errors"
	"fmt"
	"uaa/vdnd/pkg/rules/combat"
)

// StartTurn begins the current participant's turn
func (e *Encounter) StartTurn() (*combat.TurnState, error) {
	p := e.GetCurrentParticipant()
	if p == nil {
		return nil, errors.New("no current participant")
	}

	if p.IsDelaying {
		return nil, fmt.Errorf("participant %s is delaying", p.Entity.ID)
	}

	// Create turn state
	turn := combat.NewTurn(p.Entity)
	p.TurnState = turn

	// Process start-of-turn effects
	p.Entity.Conditions.StartTurn()

	return turn, nil
}

// EndTurn ends the current participant's turn
func (e *Encounter) EndTurn() error {
	p := e.GetCurrentParticipant()
	if p == nil {
		return errors.New("no current participant")
	}

	// Process end-of-turn effects
	p.Entity.Conditions.EndTurn(p.Entity)

	// Persistent damage etc would go here
	// e.ProcessPersistentDamage(p.Entity)

	p.HasActed = true
	p.TurnState = nil

	// Advance to next turn
	e.CurrentTurn++

	// Check if round is over
	if e.CurrentTurn >= len(e.Participants) {
		e.EndRound()
	}

	return nil
}

// EndRound resets states for a new round
func (e *Encounter) EndRound() {
	for _, p := range e.Participants {
		p.HasActed = false
	}
	e.CurrentRound++
	e.CurrentTurn = 0
}

// Delay causes current participant to delay their turn
func (e *Encounter) Delay() error {
	p := e.GetCurrentParticipant()
	if p == nil {
		return errors.New("no current participant")
	}

	p.IsDelaying = true

	// Effectively they end their turn now without acting
	e.CurrentTurn++
	if e.CurrentTurn >= len(e.Participants) {
		e.EndRound()
	}

	return nil
}

// ResumeFromDelay lets a delaying participant take their turn
func (e *Encounter) ResumeFromDelay(entityID string) error {
	var targetIdx = -1
	for i, p := range e.Participants {
		if p.Entity.ID == entityID && p.IsDelaying {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		return fmt.Errorf("participant %s not found or not delaying", entityID)
	}

	p := e.Participants[targetIdx]
	p.IsDelaying = false

	// Move participant to CurrentTurn position in the slice
	// Rules: They go immediately after the creature that just acted.
	// In our simple model, we'll insert them at e.CurrentTurn.

	participant := e.Participants[targetIdx]
	// Remove from old position
	e.Participants = append(e.Participants[:targetIdx], e.Participants[targetIdx+1:]...)

	// Insert at CurrentTurn
	// Adjust CurrentTurn if we removed someone before it
	if targetIdx < e.CurrentTurn {
		e.CurrentTurn--
	}

	// Insert
	if e.CurrentTurn >= len(e.Participants) {
		e.Participants = append(e.Participants, participant)
		e.CurrentTurn = len(e.Participants) - 1
	} else {
		e.Participants = append(e.Participants[:e.CurrentTurn], append([]*Participant{participant}, e.Participants[e.CurrentTurn:]...)...)
	}

	return nil
}

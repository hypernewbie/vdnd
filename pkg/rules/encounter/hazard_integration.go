package encounter

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/hazard"
)

// AddHazard adds a hazard to the encounter
func (e *Encounter) AddHazard(h *hazard.Hazard) {
	if h.Complexity != hazard.ComplexityComplex {
		// Simple hazards don't join initiative
		return
	}

	e.Participants = append(e.Participants, NewHazardParticipant(h))
}

// RollHazardInitiative rolls initiative for all hazard participants
func (e *Encounter) RollHazardInitiative() {
	for i := range e.Participants {
		if e.Participants[i].Type == ParticipantHazard && e.Participants[i].Hazard != nil {
			h := e.Participants[i].Hazard
			// Hazards use their Initiative modifier
			roll := dice.DieRoll{Count: 1, Sides: 20}.Roll()
			e.Participants[i].Initiative = roll + h.Initiative
		}
	}
}

// ExecuteHazardTurn runs a hazard's routine
func (e *Encounter) ExecuteHazardTurn(hazardID string) hazard.TurnResult {
	p := e.GetParticipantByID(hazardID)
	if p == nil || p.Type != ParticipantHazard {
		return hazard.TurnResult{}
	}

	// Gather potential targets (all entities at hazard's position)
	targets := e.GetEntitiesAtPosition(p.Hazard.Position)

	// Execute turn
	result := p.Hazard.TakeTurn(targets)
	p.HasActed = true

	return result
}

// GetEntitiesAtPosition returns all entities at a given position
func (e *Encounter) GetEntitiesAtPosition(position string) []*entity.Entity {
	entities := make([]*entity.Entity, 0)
	for _, p := range e.Participants {
		if p.Type == ParticipantEntity && p.Entity != nil {
			if p.Entity.Position == position {
				entities = append(entities, p.Entity)
			}
		}
	}
	return entities
}

// GetParticipantByID finds a participant by their ID
func (e *Encounter) GetParticipantByID(id string) *Participant {
	for i := range e.Participants {
		if e.Participants[i].GetID() == id {
			return e.Participants[i]
		}
	}
	return nil
}

// ProcessHazardTriggers checks if any hazards should trigger
func (e *Encounter) ProcessHazardTriggers(event ability.Event) []hazard.TriggerResult {
	results := make([]hazard.TriggerResult, 0)

	for _, p := range e.Participants {
		if p.Type != ParticipantHazard || p.Hazard == nil {
			continue
		}

		h := p.Hazard
		if h.IsDisabled {
			continue
		}

		if h.CheckTrigger(event) {
			// Hazard triggered, execute immediate effect
			targets := e.GetEntitiesAtPosition(h.Position)
			hazardResults := h.Activate(targets)

			results = append(results, hazard.TriggerResult{
				HazardID:   h.ID,
				HazardName: h.Name,
				Triggered:  true,
				Results:    hazardResults,
			})
		}
	}

	return results
}

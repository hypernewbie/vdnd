package encounter

import (
	"uaa/vdnd/pkg/rules/combat"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/hazard"
)

// ParticipantType distinguishes entities from hazards
type ParticipantType int

const (
	ParticipantEntity ParticipantType = iota
	ParticipantHazard
)

// Participant represents anyone/anything in initiative order
type Participant struct {
	Type       ParticipantType
	Entity     *entity.Entity // If Type == ParticipantEntity
	Hazard     *hazard.Hazard // If Type == ParticipantHazard

	// Common fields
	Initiative int
	HasActed   bool // This round
	IsDelaying bool
	TurnState  *combat.TurnState // Only for entities
}

// GetID returns the identifier for this participant
func (p *Participant) GetID() string {
	if p.Type == ParticipantEntity && p.Entity != nil {
		return p.Entity.ID
	}
	if p.Type == ParticipantHazard && p.Hazard != nil {
		return p.Hazard.ID
	}
	return ""
}

// GetName returns display name
func (p *Participant) GetName() string {
	if p.Type == ParticipantEntity && p.Entity != nil {
		return p.Entity.Name
	}
	if p.Type == ParticipantHazard && p.Hazard != nil {
		return p.Hazard.Name
	}
	return "Unknown"
}

// IsActive returns true if participant can still act
func (p *Participant) IsActive() bool {
	if p.Type == ParticipantEntity && p.Entity != nil {
		return p.Entity.CurrentHP > 0
	}
	if p.Type == ParticipantHazard && p.Hazard != nil {
		return !p.Hazard.IsDisabled
	}
	return false
}

// NewEntityParticipant creates a participant from an entity
func NewEntityParticipant(e *entity.Entity) *Participant {
	return &Participant{
		Type:      ParticipantEntity,
		Entity:    e,
		TurnState: combat.NewTurn(e),
	}
}

// NewHazardParticipant creates a participant from a hazard
func NewHazardParticipant(h *hazard.Hazard) *Participant {
	return &Participant{
		Type:   ParticipantHazard,
		Hazard: h,
	}
}
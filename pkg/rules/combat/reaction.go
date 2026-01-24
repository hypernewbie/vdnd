package combat

import (
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
)

type ReactionTrigger int

const (
	TriggerOnDamageTaken ReactionTrigger = iota
	TriggerOnMovementInReach
	TriggerOnManipulateInReach
	TriggerOnAllyDamaged
)

type ReactionEvent struct {
	Trigger    ReactionTrigger
	Source     *entity.Entity // Who caused the trigger
	Target     *entity.Entity // Who is affected
	Damage     int            // For damage triggers
	DamageType item.DamageType
	Position   string // For movement triggers
}

type Reaction interface {
	Name() string
	TriggerType() ReactionTrigger
	CanUse(actor *entity.Entity, event ReactionEvent) bool
}

// ReactionQueue holds pending reactions for an event
type ReactionQueue struct {
	Event     ReactionEvent
	Available []AvailableReaction
	Resolved  bool
}

type AvailableReaction struct {
	Actor    *entity.Entity
	Reaction Reaction
}

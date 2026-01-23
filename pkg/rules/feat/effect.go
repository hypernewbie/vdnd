package feat

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/trait"
)

// Minimal interfaces for combat and encounter to avoid circular dependencies
type TurnState interface {
	SpendActions(cost ability.ActionCost) error
	SpendReaction() error
	RecordAttack()
	CanAct() bool
}

type Encounter interface {
	GetID() string
}

// FeatActor is a minimal interface for the entity receiving the effect
type FeatActor interface {
	GetID() string
	GetName() string
}

// ActionGrant represents an action granted by a feat
type ActionGrant struct {
	Name    string
	Cost    ability.ActionCost
	Traits  trait.TraitSet
	Execute ActionFunc
}

type ActionFunc func(actor FeatActor, target FeatActor, turn TurnState) ability.ActionResult

// ReactionGrant represents a reaction granted by a feat
type ReactionGrant struct {
	Name      string
	Trigger   ability.EventType
	Condition ReactionCondition
	Execute   ReactionFunc
}

type ReactionCondition func(event ability.Event, reactor FeatActor) bool
type ReactionFunc func(event ability.Event, reactor FeatActor, encounter Encounter) ability.ActionResult

// PassiveType represents what a passive effect modifies
type PassiveType int

const (
	PassiveHP PassiveType = iota
	PassiveAC
	PassiveSpeed
	PassiveSave
	PassiveSkill
	PassiveProficiency
)

// PassiveEffect represents an always-on bonus
type PassiveEffect struct {
	Type      PassiveType
	Value     int
	Condition string // Optional condition description
	// TODO: WARNING - PASSIVE EFFECTS ARE CURRENTLY STUBBED!
	// The FeatActor interface only exposes GetID/GetName, so Apply() functions cannot
	// actually modify the Entity's stats (HP, Speed, etc).
	// To fix this, FeatActor needs methods like AddHPModifier(int) or ModifySpeed(int),
	// and the Entity implementation needs to support these modifiers.
	Apply PassiveFunc
}

type PassiveFunc func(e FeatActor)

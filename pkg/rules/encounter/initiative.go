package encounter

import (
	"sort"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
)

type InitiativeType int

const (
	InitPerception InitiativeType = iota
	InitStealth
	InitDeception
)

// RollInitiative rolls initiative for all participants using a default type
func (e *Encounter) RollInitiative(initType InitiativeType) {
	for _, p := range e.Participants {
		p.Initiative = RollInitiativeFor(p.Entity, initType)
	}
	e.State = StateRollingInitiative
}

// RollInitiativeFor rolls initiative for a single entity
func RollInitiativeFor(ent *entity.Entity, initType InitiativeType) int {
	mod := 0
	switch initType {
	case InitPerception:
		mod = ent.GetPerception()
	case InitStealth:
		mod = ent.GetSkillModifier(ability.SkillStealth)
	case InitDeception:
		mod = ent.GetSkillModifier(ability.SkillDeception)
	}

	roll := dice.DieRoll{Count: 1, Sides: 20, Modifier: 0}.Roll()
	return roll + mod
}

// SortByInitiative orders participants from highest to lowest
func (e *Encounter) SortByInitiative() {
	sort.SliceStable(e.Participants, func(i, j int) bool {
		if e.Participants[i].Initiative != e.Participants[j].Initiative {
			return e.Participants[i].Initiative > e.Participants[j].Initiative
		}
		// Tie-breaker: higher modifier goes first
		// We'll use perception modifier as a default tie-breaker if same initiative
		return ResolveTie(e.Participants[i], e.Participants[j]) < 0
	})
}

// ResolveTie determines order when two participants have same initiative
// Returns -1 if a first, 1 if b first
func ResolveTie(a, b *Participant) int {
	modA := a.Entity.GetPerception()
	modB := b.Entity.GetPerception()

	if modA > modB {
		return -1
	}
	if modB > modA {
		return 1
	}

	// Still tied? Use ID for determinism
	if a.Entity.ID < b.Entity.ID {
		return -1
	}
	return 1
}

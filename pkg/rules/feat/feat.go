package feat

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/trait"
)

type FeatType int

const (
	FeatTypeAncestry FeatType = iota
	FeatTypeClass
	FeatTypeSkill
	FeatTypeGeneral
	FeatTypeArchetype
)

type Prerequisite struct {
	MinLevel        int
	MinAbilityScore map[ability.Ability]int
	RequiredFeat    string
	RequiredSkill   *SkillRequirement
	RequiredTrait   trait.TraitID
}

type SkillRequirement struct {
	Skill ability.SkillID
	Rank  ability.ProficiencyRank
}

type Feat struct {
	ID            string
	Name          string
	Type          FeatType
	Level         int // Minimum level to take
	Prerequisites []Prerequisite
	Traits        trait.TraitSet
	Description   string

	// Effects
	GrantsAction   *ActionGrant
	GrantsReaction *ReactionGrant
	Passives       []PassiveEffect
}

// FeatEntity is an interface representing an entity that can have feats.
// This allows the feat package to check prerequisites without importing entity.
type FeatEntity interface {
	GetLevel() int
	GetAbilityScore(ability.Ability) int
	HasFeat(featID string) bool
	HasSkillRank(skillID ability.SkillID, rank ability.ProficiencyRank) bool
	// TODO: WARNING - TRAIT PREREQUISITES REQUIRE SYNC!
	// Feats often require specific traits (e.g. 'human', 'fighter').
	// Ensure that when creating an Entity, its Ancestry and Class are explicitly
	// added to its Traits list. If they are only stored in the Ancestry/Class string fields,
	// this HasTrait check will fail and prerequisites won't work.
	HasTrait(traitID trait.TraitID) bool
}

// MeetsPrerequisites checks if an entity qualifies
func (f *Feat) MeetsPrerequisites(e FeatEntity) bool {
	if e.GetLevel() < f.Level {
		return false
	}

	for _, p := range f.Prerequisites {
		if p.MinLevel > e.GetLevel() {
			return false
		}

		for ab, min := range p.MinAbilityScore {
			if e.GetAbilityScore(ab) < min {
				return false
			}
		}

		if p.RequiredFeat != "" {
			if !e.HasFeat(p.RequiredFeat) {
				return false
			}
		}

		if p.RequiredSkill != nil {
			if !e.HasSkillRank(p.RequiredSkill.Skill, p.RequiredSkill.Rank) {
				return false
			}
		}

		if p.RequiredTrait != "" {
			if !e.HasTrait(p.RequiredTrait) {
				return false
			}
		}
	}

	return true
}

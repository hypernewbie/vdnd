package spell

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/trait"
)

// StandardRituals contains common rituals
var StandardRituals = map[string]*Ritual{}

func init() {
	// Resurrect - Rank 5
	// src: rules/compendium/spells/rituals/resurrect.md
	resurrect := NewRitual("resurrect", "Resurrect", 5, ability.SkillReligion, 2).
		WithCastingTime(1, DurationDays).
		WithCost(7500). // 75 gp in diamonds
		WithSecondaryCheck(ability.SkillMedicine, ability.Expert, "Prepare the body").
		WithSecondaryCheck(ability.SkillReligion, ability.Trained, "Call the soul")
	resurrect.RequiredRank = ability.Expert
	resurrect.Effect = &GenericRitualEffect{
		CritSuccessDesc: "Target returns to life at full HP with no negative conditions",
		SuccessDesc:     "Target returns to life at 1 HP, wounded 1, and fatigued",
		FailureDesc:     "Ritual fails, materials are not consumed",
		CritFailureDesc: "Ritual fails catastrophically",
		BacklashDesc:    "Primary caster is doomed 1",
		RefundOnFailure:  true,
	}
	StandardRituals["resurrect"] = resurrect

	// Commune - Rank 6
	// src: rules/compendium/spells/rituals/commune.md
	commune := NewRitual("commune", "Commune", 6, ability.SkillReligion, 0).
		WithCastingTime(1, DurationHours).
		WithCost(15000) // 150 gp
	commune.RequiredRank = ability.Master
	commune.Effect = &GenericRitualEffect{
		CritSuccessDesc: "Deity answers 7 questions clearly",
		SuccessDesc:     "Deity answers 5 questions with single words",
		FailureDesc:     "Deity does not respond",
		CritFailureDesc: "Deity is angered, caster cannot attempt again for a week",
		RefundOnFailure:  true,
	}
	StandardRituals["commune"] = commune

	// Plane Shift - Rank 7
	planeShift := NewRitual("plane_shift", "Plane Shift", 7, ability.SkillOccultism, 3).
		WithCastingTime(1, DurationHours).
		WithCost(35000). // 350 gp tuning fork
		WithSecondaryCheck(ability.SkillArcana, ability.Trained, "Stabilise the portal").
		WithSecondaryCheck(ability.SkillOccultism, ability.Trained, "Navigate the planes").
		WithSecondaryCheck(ability.SkillSurvival, ability.Trained, "Chart safe arrival")
	planeShift.RequiredRank = ability.Master
	planeShift.Effect = &GenericRitualEffect{
		CritSuccessDesc: "All targets arrive precisely where intended",
		SuccessDesc:     "Targets arrive on correct plane, 1d20 miles from destination",
		FailureDesc:     "Portal fails to open, materials consumed",
		CritFailureDesc: "Targets scattered across the plane or arrive on wrong plane",
		BacklashDesc:    "All casters take 10d6 force damage",
		RefundOnFailure:  false,
	}
	StandardRituals["plane_shift"] = planeShift

	// Atone - Rank 4
	atone := NewRitual("atone", "Atone", 4, ability.SkillReligion, 0).
		WithCastingTime(1, DurationDays).
		WithCost(2000) // 20 gp
	atone.RequiredRank = ability.Expert
	atone.Effect = &GenericRitualEffect{
		CritSuccessDesc: "Target's anathema violation is forgiven, regains all class features",
		SuccessDesc:     "Target must perform a quest, then features are restored",
		FailureDesc:     "Deity does not grant forgiveness at this time",
		CritFailureDesc: "Target is further punished, loses additional powers",
		RefundOnFailure:  true,
	}
	StandardRituals["atone"] = atone

	// Create Undead - Rank 2
	createUndead := NewRitual("create_undead", "Create Undead", 2, ability.SkillReligion, 1).
		WithCastingTime(1, DurationDays).
		WithCost(2500). // 25 gp onyx
		WithSecondaryCheck(ability.SkillArcana, ability.Trained, "Bind the negative energy")
	createUndead.Traits = trait.TraitSet{trait.TraitEvil, trait.TraitNecromancy}
	createUndead.RequiredRank = ability.Expert
	createUndead.Effect = &GenericRitualEffect{
		CritSuccessDesc: "Creature is raised and permanently under your control",
		SuccessDesc:     "Creature is raised, obeys for 1 week unless renewed",
		FailureDesc:     "Ritual fails, corpse is damaged beyond use",
		CritFailureDesc: "Creature rises but is uncontrolled and hostile",
		BacklashDesc:    "Undead immediately attacks all living creatures",
		RefundOnFailure:  true,
	}
	StandardRituals["create_undead"] = createUndead
}

// GetRitual retrieves a ritual by ID
func GetRitual(id string) *Ritual {
	return StandardRituals[id]
}

// ListRituals returns all available rituals
func ListRituals() []*Ritual {
	list := make([]*Ritual, 0, len(StandardRituals))
	for _, r := range StandardRituals {
		list = append(list, r)
	}
	return list
}

package skill

import (
	"uaa/vdnd/pkg/rules/ability"
)

type Skill struct {
	ID         ability.SkillID
	Name       string
	KeyAbility ability.Ability
}

func GetKeyAbility(id ability.SkillID) ability.Ability {
	return ability.GetKeyAbility(id)
}

func AllSkills() []ability.SkillID {
	return []ability.SkillID{
		ability.SkillAcrobatics, ability.SkillArcana, ability.SkillAthletics, ability.SkillCrafting,
		ability.SkillDeception, ability.SkillDiplomacy, ability.SkillIntimidation, ability.SkillMedicine,
		ability.SkillNature, ability.SkillOccultism, ability.SkillPerformance, ability.SkillReligion,
		ability.SkillSociety, ability.SkillStealth, ability.SkillSurvival, ability.SkillThievery,
	}
}

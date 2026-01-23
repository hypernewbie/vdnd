package ability

type SkillID string

const (
	SkillAcrobatics   SkillID = "acrobatics"
	SkillArcana       SkillID = "arcana"
	SkillAthletics    SkillID = "athletics"
	SkillCrafting     SkillID = "crafting"
	SkillDeception    SkillID = "deception"
	SkillDiplomacy    SkillID = "diplomacy"
	SkillIntimidation SkillID = "intimidation"
	SkillMedicine     SkillID = "medicine"
	SkillNature       SkillID = "nature"
	SkillOccultism    SkillID = "occultism"
	SkillPerformance  SkillID = "performance"
	SkillReligion     SkillID = "religion"
	SkillSociety      SkillID = "society"
	SkillStealth      SkillID = "stealth"
	SkillSurvival     SkillID = "survival"
	SkillThievery     SkillID = "thievery"
)

func GetKeyAbility(id SkillID) Ability {
	switch id {
	case SkillAthletics:
		return Strength
	case SkillAcrobatics, SkillStealth, SkillThievery:
		return Dexterity
	case SkillArcana, SkillCrafting, SkillOccultism, SkillSociety:
		return Intelligence
	case SkillMedicine, SkillNature, SkillReligion, SkillSurvival:
		return Wisdom
	case SkillDeception, SkillDiplomacy, SkillIntimidation, SkillPerformance:
		return Charisma
	default:
		return Strength // Default
	}
}

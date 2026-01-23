package spell

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/combat"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
)

type SpellRank int // 1-10

type SpellTradition int

const (
	TraditionArcane SpellTradition = iota
	TraditionDivine
	TraditionOccult
	TraditionPrimal
)

type SpellComponent int

const (
	ComponentSomatic SpellComponent = iota
	ComponentVerbal
	ComponentMaterial
	ComponentFocus
)

type SpellAreaType int

const (
	AreaNone SpellAreaType = iota
	AreaBurst
	AreaCone
	AreaLine
	AreaEmanation
)

type Spell struct {
	ID                 string
	Name               string
	Rank               SpellRank
	Traditions         []SpellTradition
	ActionCost         combat.ActionCost
	Components         []SpellComponent
	Range              int // Feet, 0 = touch, -1 = self
	Area               SpellAreaType
	AreaSize           int // Radius for burst, length for line/cone
	Targets            int // 0 = area effect, 1+ = targeted
	Duration           int // Rounds, -1 = instantaneous, -2 = sustained
	Save               ability.SaveType
	IsBasicSave        bool // True for "basic X save"
	RequiresAttackRoll bool

	// Effect is implemented per-spell
	Effect SpellEffect
}

type EffectRoll struct {
	Damage int
	Healed int
}

type SpellEffect interface {
	Roll(caster *entity.Entity) EffectRoll
	Apply(caster, target *entity.Entity, degree check.DegreeOfSuccess, roll EffectRoll) EffectResult
}

type EffectResult struct {
	Damage      int
	DamageType  item.DamageType
	Conditions  []condition.ConditionInstance
	Healed      int
	Description string
}

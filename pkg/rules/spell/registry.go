package spell

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/item"
)

var (
	// Electric Arc - 2-action, up to 2 targets, basic Reflex
	ElectricArc = Spell{
		ID:          "electric-arc",
		Name:        "Electric Arc",
		Rank:        1,
		Traditions:  []SpellTradition{TraditionArcane, TraditionPrimal},
		ActionCost:  ability.CostTwo,
		Components:  []SpellComponent{ComponentSomatic, ComponentVerbal},
		Range:       30,
		Targets:     2,
		Save:        ability.SaveReflex,
		IsBasicSave: true,
		Effect: &DamageEffect{
			DamageDice: dice.DieRoll{Groups: []dice.DiceGroup{{Count: 1, Sides: 4}}, Modifier: 0},
			DamageType: item.Electricity,
		},
	}

	// Fireball - 2-action, 20ft burst, basic Reflex
	Fireball = Spell{
		ID:          "fireball",
		Name:        "Fireball",
		Rank:        3,
		Traditions:  []SpellTradition{TraditionArcane, TraditionPrimal},
		ActionCost:  ability.CostTwo,
		Components:  []SpellComponent{ComponentSomatic, ComponentVerbal},
		Range:       500,
		Area:        AreaBurst,
		AreaSize:    20,
		Save:        ability.SaveReflex,
		IsBasicSave: true,
		Effect: &DamageEffect{
			DamageDice: dice.DieRoll{Groups: []dice.DiceGroup{{Count: 6, Sides: 6}}, Modifier: 0},
			DamageType: item.Fire,
		},
	}

	// Fear - 2-action, single target, Will save
	Fear = Spell{
		ID:          "fear",
		Name:        "Fear",
		Rank:        1,
		Traditions:  []SpellTradition{TraditionArcane, TraditionDivine, TraditionOccult, TraditionPrimal},
		ActionCost:  ability.CostTwo,
		Range:       30,
		Targets:     1,
		Save:        ability.SaveWill,
		IsBasicSave: false,
		Effect:      &FearEffect{},
	}
)

var spells = map[string]*Spell{
	"electric-arc": &ElectricArc,
	"fireball":     &Fireball,
	"fear":         &Fear,
}

func GetSpell(id string) (*Spell, bool) {
	s, ok := spells[id]
	return s, ok
}

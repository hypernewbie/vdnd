package hazard

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/affliction"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/item"
	"uaa/vdnd/pkg/rules/trait"
)

var (
	// Pit Trap - Simple, fall damage
	PitTrap = Hazard{
		ID:         "pit-trap",
		Name:       "Pit Trap",
		Level:      1,
		Type:       HazardTrap,
		Complexity: ComplexitySimple,
		Traits:     trait.TraitSet{trait.TraitMechanical, trait.TraitTrap},
		StealthDC:  18,
		DisableOptions: []DisableOption{
			{Skill: ability.SkillThievery, DC: 15, Description: "Jam the trapdoor"},
		},
		Trigger: TriggerCondition{
			Type:      TriggerPressure,
			MinWeight: 50,
		},
		Effect: &DamageEffect{
			Damage:      dice.DieRoll{Count: 2, Sides: 6, Modifier: 0}, // 2d6 fall damage
			DamageType:  item.Bludgeoning,
			SaveType:    ability.SaveReflex,
			SaveDC:      17,
			IsBasicSave: false, // Reflex to grab edge
		},
	}

	// Poison Dart Trap - Simple, ranged attack + poison
	PoisonDartTrap = Hazard{
		ID:         "poison-dart-trap",
		Name:       "Poison Dart Trap",
		Level:      2,
		Type:       HazardTrap,
		Complexity: ComplexitySimple,
		Traits:     trait.TraitSet{trait.TraitMechanical, trait.TraitTrap},
		StealthDC:  20,
		AC:         18,
		HP:         30,
		Hardness:   8,
		DisableOptions: []DisableOption{
			{Skill: ability.SkillThievery, DC: 18, Description: "Disable firing mechanism"},
		},
		Trigger: TriggerCondition{Type: TriggerTouch},
		Effect: &MultiEffect{
			Effects: []HazardEffect{
				&AttackEffect{
					AttackBonus: 12,
					Damage:      dice.DieRoll{Count: 1, Sides: 6, Modifier: 0},
					DamageType:  item.Piercing,
				},
				&AfflictionEffect{
					Affliction: affliction.GiantCentipedeVenom,
					OnHit:      true,
				},
			},
		},
	}

	// Blade Barrier - Complex, acts each round
	BladeBarrier = Hazard{
		ID:         "blade-barrier",
		Name:       "Spinning Blade Trap",
		Level:      5,
		Type:       HazardTrap,
		Complexity: ComplexityComplex,
		Traits:     trait.TraitSet{trait.TraitMechanical, trait.TraitTrap},
		StealthDC:  24,
		AC:         22,
		HP:         60,
		Hardness:   12,
		Initiative: 8, // Modifier for initiative roll
		DisableOptions: []DisableOption{
			{Skill: ability.SkillThievery, DC: 22, Description: "Jam the mechanism"},
			{Skill: ability.SkillAthletics, DC: 24, Description: "Force blades apart"},
		},
		Trigger: TriggerCondition{Type: TriggerEnter},
		Effect: &DamageEffect{
			Damage:      dice.DieRoll{Count: 3, Sides: 8, Modifier: 0}, // 3d8 per round
			DamageType:  item.Slashing,
			SaveType:    ability.SaveReflex,
			SaveDC:      22,
			IsBasicSave: true,
		},
	}
)

var hazards = map[string]*Hazard{
	"pit-trap":          &PitTrap,
	"poison-dart-trap":  &PoisonDartTrap,
	"blade-barrier":     &BladeBarrier,
}

func GetHazard(id string) (*Hazard, bool) {
	h, ok := hazards[id]
	return h, ok
}

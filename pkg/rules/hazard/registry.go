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
			Damage:      dice.DieRoll{Groups: []dice.DiceGroup{{Count: 2, Sides: 6}}, Modifier: 0}, // 2d6 fall damage
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
					Damage:      dice.DieRoll{Groups: []dice.DiceGroup{{Count: 1, Sides: 6}}, Modifier: 0},
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
			Damage:      dice.DieRoll{Groups: []dice.DiceGroup{{Count: 3, Sides: 8}}, Modifier: 0}, // 3d8 per round
			DamageType:  item.Slashing,
			SaveType:    ability.SaveReflex,
			SaveDC:      22,
			IsBasicSave: true,
		},
	}
)

var hazards = map[string]*Hazard{
	"pit-trap":         &PitTrap,
	"poison-dart-trap": &PoisonDartTrap,
	"blade-barrier":    &BladeBarrier,
}

// StandardComplexHazards contains predefined complex hazards
var StandardComplexHazards = map[string]func() *Hazard{}

func init() {
	// Spinning Blade Pillar - Level 4 Complex Trap
	// src: rules/rules/gm/hazards/spinning-blade-pillar.md
	StandardComplexHazards["spinning_blade_pillar"] = func() *Hazard {
		h := NewHazard("spinning_blade_pillar", "Spinning Blade Pillar", 4)
		h.Type = HazardTrap
		h.Complexity = ComplexityComplex
		h.StealthDC = 23
		h.AC = 21
		h.Fortitude = 14
		h.Reflex = 10
		h.HP = 50
		h.Hardness = 10
		h.Initiative = 8

		h.DisableOptions = []DisableOption{
			{Skill: ability.SkillThievery, DC: 21, Description: "Jam the mechanism"},
		}

		h.Routine = NewRoutine(2).
			AddAttack("Blade Slash", 1, 15, dice.DieRoll{Groups: []dice.DiceGroup{{Count: 2, Sides: 8}}, Modifier: 5}, item.Slashing, 1).
			AddAttack("Blade Slash", 1, 15, dice.DieRoll{Groups: []dice.DiceGroup{{Count: 2, Sides: 8}}, Modifier: 5}, item.Slashing, 1)

		return h
	}

	// Poisoned Dart Gallery - Level 6 Complex Trap
	StandardComplexHazards["poisoned_dart_gallery"] = func() *Hazard {
		h := NewHazard("poisoned_dart_gallery", "Poisoned Dart Gallery", 6)
		h.Type = HazardTrap
		h.Complexity = ComplexityComplex
		h.StealthDC = 26
		h.AC = 24
		h.Fortitude = 15
		h.Reflex = 12
		h.HP = 60
		h.Hardness = 12
		h.Initiative = 10

		h.DisableOptions = []DisableOption{
			{Skill: ability.SkillThievery, DC: 24, Description: "Block the dart holes"},
			{Skill: ability.SkillAthletics, DC: 26, Description: "Smash the pressure plates"},
		}

		h.Routine = NewRoutine(3).
			AddAttack("Poison Dart", 1, 17, dice.DieRoll{Groups: []dice.DiceGroup{{Count: 1, Sides: 6}}, Modifier: 3}, item.Piercing, 1).
			AddSaveEffect("Poison", 0, ability.SaveFortitude, 22,
				"No effect",
				"Sickened 1 and 1d6 poison damage",
				"Sickened 2 and 2d6 poison damage").
			AddAttack("Poison Dart", 1, 17, dice.DieRoll{Groups: []dice.DiceGroup{{Count: 1, Sides: 6}}, Modifier: 3}, item.Piercing, 1)

		return h
	}

	// Flooding Room - Level 8 Complex Trap
	StandardComplexHazards["flooding_room"] = func() *Hazard {
		h := NewHazard("flooding_room", "Flooding Room", 8)
		h.Type = HazardTrap
		h.Complexity = ComplexityComplex
		h.StealthDC = 28
		h.AC = 26
		h.Fortitude = 18
		h.Reflex = 14
		h.HP = 80
		h.Hardness = 15
		h.Initiative = 12

		h.DisableOptions = []DisableOption{
			{Skill: ability.SkillThievery, DC: 28, Description: "Open the drainage grate"},
			{Skill: ability.SkillAthletics, DC: 30, Description: "Force open the door"},
		}

		h.Routine = NewRoutine(1).
			AddSaveEffect("Rising Waters", 1, ability.SaveReflex, 26,
				"Avoid worst of current, take half damage",
				"Swept off feet, 2d6 bludgeoning and prone",
				"Pulled under, 4d6 bludgeoning, prone, and grabbed by water")

		return h
	}

	// Haunted Stage - Level 5 Complex Haunt
	StandardComplexHazards["haunted_stage"] = func() *Hazard {
		h := NewHazard("haunted_stage", "Haunted Stage", 5)
		h.Type = HazardHaunt
		h.Complexity = ComplexityComplex
		h.StealthDC = 24
		h.Initiative = 9

		h.DisableOptions = []DisableOption{
			{Skill: ability.SkillReligion, DC: 22, Description: "Perform last rites"},
			{Skill: ability.SkillPerformance, DC: 24, Description: "Complete the unfinished play"},
		}

		h.Routine = NewRoutine(2).
			AddSaveEffect("Terrifying Visage", 1, ability.SaveWill, 22,
				"Unaffected",
				"Frightened 1",
				"Frightened 2 and fleeing").
			AddSaveEffect("Ghostly Props", 1, ability.SaveReflex, 20,
				"Dodge the flying objects",
				"2d6 bludgeoning from hurled props",
				"4d6 bludgeoning and knocked prone")

		return h
	}
}

// GetComplexHazard retrieves a hazard template by ID
func GetComplexHazard(id string) *Hazard {
	if factory, ok := StandardComplexHazards[id]; ok {
		return factory()
	}
	return nil
}

// ListComplexHazards returns all available complex hazard IDs
func ListComplexHazards() []string {
	ids := make([]string, 0, len(StandardComplexHazards))
	for id := range StandardComplexHazards {
		ids = append(ids, id)
	}
	return ids
}

func GetHazard(id string) (*Hazard, bool) {
	h, ok := hazards[id]
	return h, ok
}

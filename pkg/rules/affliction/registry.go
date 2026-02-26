package affliction

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/item"
)

var (
	GiantCentipedeVenom = Affliction{
		ID:           "giant-centipede-venom",
		Name:         "Giant Centipede Venom",
		Type:         TypePoison,
		DC:           17,
		Save:         ability.SaveFortitude,
		MaxStage:     3,
		Interval:     1,
		IntervalUnit: ability.IntervalRounds,
		Stages: []Stage{
			{
				Number:     1,
				Damage:     dice.DieRoll{Groups: []dice.DiceGroup{{Count: 1, Sides: 6}}, Modifier: 0},
				DamageType: item.Poison,
				Conditions: []ConditionEffect{{ID: condition.FlatFooted, Value: 0}},
			},
			{
				Number:     2,
				Damage:     dice.DieRoll{Groups: []dice.DiceGroup{{Count: 1, Sides: 8}}, Modifier: 0},
				DamageType: item.Poison,
				Conditions: []ConditionEffect{{ID: condition.FlatFooted, Value: 0}, {ID: condition.Clumsy, Value: 1}},
			},
			{
				Number:     3,
				Damage:     dice.DieRoll{Groups: []dice.DiceGroup{{Count: 1, Sides: 12}}, Modifier: 0},
				DamageType: item.Poison,
				Conditions: []ConditionEffect{{ID: condition.FlatFooted, Value: 0}, {ID: condition.Clumsy, Value: 2}},
			},
		},
	}

	ZombieRot = Affliction{
		ID:           "zombie-rot",
		Name:         "Zombie Rot",
		Type:         TypeDisease,
		DC:           14,
		Save:         ability.SaveFortitude,
		OnsetDelay:   1,
		OnsetUnit:    ability.IntervalDays,
		MaxStage:     4,
		Interval:     1,
		IntervalUnit: ability.IntervalDays,
		Stages: []Stage{
			{
				Number: 1,
				// Carrier - no symptoms
			},
			{
				Number:     2,
				Damage:     dice.DieRoll{Groups: []dice.DiceGroup{{Count: 1, Sides: 6}}, Modifier: 0},
				DamageType: item.Negative,
				Conditions: []ConditionEffect{{ID: condition.Slowed, Value: 1}},
			},
			{
				Number:     3,
				Damage:     dice.DieRoll{Groups: []dice.DiceGroup{{Count: 1, Sides: 6}}, Modifier: 0},
				DamageType: item.Negative,
				Conditions: []ConditionEffect{{ID: condition.Slowed, Value: 2}},
			},
			{
				Number:  4,
				IsFatal: true,
			},
		},
	}
)

var afflictions = map[string]*Affliction{
	"giant-centipede-venom": &GiantCentipedeVenom,
	"zombie-rot":            &ZombieRot,
}

func GetAffliction(id string) (*Affliction, bool) {
	a, ok := afflictions[id]
	return a, ok
}

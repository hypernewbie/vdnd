package skill

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
)

type mockRoller struct {
	results []int
	index   int
}

func (m *mockRoller) Roll(count, sides int) []int {
	res := m.results[m.index : m.index+count]
	m.index += count
	return res
}

func TestRecallKnowledge(t *testing.T) {
	actor := entity.NewEntity("pc", "PC", 1)
	actor.Abilities = ability.AbilityScores{Strength: 10, Dexterity: 10, Constitution: 10, Intelligence: 10, Wisdom: 10, Charisma: 10}
	actor.SkillProficiencies[ability.SkillArcana] = ability.Trained
	dc := 15

	tests := []struct {
		name    string
		roll    int
		learned bool
		degree  check.DegreeOfSuccess
	}{
		{"Success", 15, true, check.Success},
		{"Critical Success", 25, true, check.CriticalSuccess},
		{"Failure", 10, false, check.Failure},
		{"Critical Failure", 1, false, check.CriticalFailure},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			learned, res := RecallKnowledgeWithRoll(actor, ability.SkillArcana, dc, tt.roll)
			if learned != tt.learned {
				t.Errorf("%s: learned = %v, want %v", tt.name, learned, tt.learned)
			}
			if res.Degree != tt.degree {
				t.Errorf("%s: degree = %v, want %v", tt.name, res.Degree, tt.degree)
			}
		})
	}
}

func TestTreatWounds(t *testing.T) {
	healer := entity.NewEntity("healer", "Healer", 1)
	healer.Abilities = ability.AbilityScores{Strength: 10, Dexterity: 10, Constitution: 10, Intelligence: 10, Wisdom: 10, Charisma: 10}
	healer.SkillProficiencies[ability.SkillMedicine] = ability.Trained
	patient := entity.NewEntity("patient", "Patient", 1)
	patient.Abilities = ability.AbilityScores{Strength: 10, Dexterity: 10, Constitution: 10, Intelligence: 10, Wisdom: 10, Charisma: 10}
	patient.MaxHP = 100
	patient.CurrentHP = 50

	t.Run("Success DC 15", func(t *testing.T) {
		patient.CurrentHP = 50
		patient.Conditions.Remove(condition.TreatWoundsImmunity)
		roller := &mockRoller{results: []int{4, 5}} // 2d8 -> 9
		res := TreatWoundsWithRoll(healer, patient, 15, 15, roller)

		if res.Degree != check.Success {
			t.Errorf("Expected Success, got %v", res.Degree)
		}
		if res.HealingAmount != 9 {
			t.Errorf("Expected 9 healing, got %d", res.HealingAmount)
		}
		if patient.CurrentHP != 59 {
			t.Errorf("Expected 59 HP, got %d", patient.CurrentHP)
		}
		if !patient.Conditions.Has(condition.TreatWoundsImmunity) {
			t.Error("Expected TreatWoundsImmunity")
		}
	})

	t.Run("Critical Success DC 15", func(t *testing.T) {
		patient.CurrentHP = 50
		patient.Conditions.Remove(condition.TreatWoundsImmunity)
		roller := &mockRoller{results: []int{4, 5, 6, 7}} // 4d8 -> 22
		res := TreatWoundsWithRoll(healer, patient, 15, 25, roller)

		if res.Degree != check.CriticalSuccess {
			t.Errorf("Expected CriticalSuccess, got %v", res.Degree)
		}
		if res.HealingAmount != 22 {
			t.Errorf("Expected 22 healing, got %d", res.HealingAmount)
		}
		if patient.CurrentHP != 72 {
			t.Errorf("Expected 72 HP, got %d", patient.CurrentHP)
		}
	})

	t.Run("Success DC 30", func(t *testing.T) {
		patient.CurrentHP = 50
		patient.Conditions.Remove(condition.TreatWoundsImmunity)
		roller := &mockRoller{results: []int{4, 5}} // 2d8 -> 9
		res := TreatWoundsWithRoll(healer, patient, 30, 30, roller)

		// 9 + 10 (bonus) = 19
		if res.HealingAmount != 19 {
			t.Errorf("Expected 19 healing, got %d", res.HealingAmount)
		}
	})

	t.Run("Crit Success DC 40", func(t *testing.T) {
		patient.CurrentHP = 50
		patient.Conditions.Remove(condition.TreatWoundsImmunity)
		roller := &mockRoller{results: []int{4, 5, 6, 7}} // 4d8 -> 22
		res := TreatWoundsWithRoll(healer, patient, 40, 50, roller)

		// 22 + 30 (bonus) = 52
		if res.HealingAmount != 52 {
			t.Errorf("Expected 52 healing, got %d", res.HealingAmount)
		}
	})

	t.Run("Critical Failure", func(t *testing.T) {
		patient.CurrentHP = 50
		patient.Conditions.Remove(condition.TreatWoundsImmunity)
		roller := &mockRoller{results: []int{6}} // 1d8 damage
		res := TreatWoundsWithRoll(healer, patient, 15, 1, roller)

		if res.Degree != check.CriticalFailure {
			t.Errorf("Expected CriticalFailure, got %v", res.Degree)
		}
		if res.HealingAmount != -6 {
			t.Errorf("Expected -6 healing (damage), got %d", res.HealingAmount)
		}
		if patient.CurrentHP != 44 {
			t.Errorf("Expected 44 HP, got %d", patient.CurrentHP)
		}
		if !patient.Conditions.Has(condition.TreatWoundsImmunity) {
			t.Error("Expected immunity even on failure")
		}
	})

	t.Run("Immune patient", func(t *testing.T) {
		patient.Conditions.Apply(condition.NewCondition(condition.TreatWoundsImmunity, "test"))
		res := TreatWoundsWithRoll(healer, patient, 15, 15, nil)

		if res.Applied {
			t.Error("Expected no application on immune patient")
		}
		if res.Degree != check.Failure {
			t.Errorf("Expected Failure for immune patient, got %v", res.Degree)
		}
	})

	t.Run("Untrained healer", func(t *testing.T) {
		healer.SkillProficiencies[ability.SkillMedicine] = ability.Untrained
		patient.Conditions.Remove(condition.TreatWoundsImmunity)
		res := TreatWoundsWithRoll(healer, patient, 15, 15, nil)

		if res.Applied {
			t.Error("Expected no application for untrained healer")
		}
		if res.Degree != check.Failure {
			t.Errorf("Expected Failure for untrained healer, got %v", res.Degree)
		}
	})
}

package skill

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
)

func setupTestEntities() (*entity.Entity, *entity.Entity) {
	actor := entity.NewEntity("actor", "Actor", 1)
	actor.Abilities = ability.AbilityScores{10, 10, 10, 10, 10, 10}
	target := entity.NewEntity("target", "Target", 1)
	target.Abilities = ability.AbilityScores{10, 10, 10, 10, 10, 10}
	return actor, target
}

type DegreeTest struct {
	Name string
	Roll int
	Want check.DegreeOfSuccess
}

// Actor Level 1, Untrained = +0 bonus. Mod 0 (Score 10).
// DC 10.
// 20 -> Total 20 -> Crit Success (Success + Nat 20)
// 10 -> Total 10 -> Success
// 9  -> Total 9  -> Failure
// 1  -> Total 1  -> Critical Failure (Failure + Nat 1)
var standardDegrees = []DegreeTest{
	{"Critical Success", 20, check.CriticalSuccess},
	{"Success", 10, check.Success},
	{"Failure", 9, check.Failure},
	{"Critical Failure", 1, check.CriticalFailure},
}

func TestExhaustiveSkills(t *testing.T) {
	actor, target := setupTestEntities()
	dc := 10 

	// --- Acrobatics: Balance ---
	t.Run("Balance", func(t *testing.T) {
		for _, tt := range standardDegrees {
			actor.Conditions.Remove(condition.Prone)
			res := Balance(actor, dc, tt.Roll)
			if res.Degree != tt.Want { t.Errorf("%s: got %v, want %v (Total: %d vs DC %d)", tt.Name, res.Degree, tt.Want, res.Total, dc) }
			if tt.Want == check.CriticalFailure && !actor.Conditions.Has(condition.Prone) {
				t.Error("Crit Failure should apply Prone")
			}
		}
	})

	// --- Acrobatics: Tumble Through ---
	t.Run("TumbleThrough", func(t *testing.T) {
		for _, tt := range standardDegrees {
			res := TumbleThrough(actor, target, tt.Roll)
			if res.Degree != tt.Want { t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want) }
		}
	})

	// --- Athletics: Climb ---
	t.Run("Climb", func(t *testing.T) {
		for _, tt := range standardDegrees {
			actor.Conditions.Remove(condition.Prone)
			move, res := Climb(actor, dc, tt.Roll)
			if res.Degree != tt.Want { t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want) }
			if tt.Want == check.CriticalSuccess && move.Speed != 10 { t.Error("Crit Success speed should be 10") }
			if tt.Want == check.Success && move.Speed != 5 { t.Error("Success speed should be 5") }
			if tt.Want == check.CriticalFailure && !actor.Conditions.Has(condition.Prone) { t.Error("Crit Fail should be Prone") }
		}
	})

	// --- Athletics: Disarm ---
	t.Run("Disarm", func(t *testing.T) {
		for _, tt := range standardDegrees {
			actor.Conditions.Remove(condition.FlatFooted)
			res := Disarm(actor, target, tt.Roll)
			if res.Degree != tt.Want { t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want) }
			if tt.Want == check.Success && !target.IsTemporarilyImmune("disarm-bonus", actor.ID) {
				t.Error("Success should apply disarm bonus immunity")
			}
			if tt.Want == check.CriticalFailure && !actor.Conditions.Has(condition.FlatFooted) {
				t.Error("Crit Fail should apply Flat-Footed to actor")
			}
		}
	})

	// --- Deception: Feint ---
	t.Run("Feint", func(t *testing.T) {
		for _, tt := range standardDegrees {
			target.Conditions.Remove(condition.FlatFooted)
			actor.Conditions.Remove(condition.FlatFooted)
			res := Feint(actor, target, tt.Roll)
			if res.Degree != tt.Want { t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want) }
			if (tt.Want == check.CriticalSuccess || tt.Want == check.Success) && !target.Conditions.HasRelative(condition.FlatFooted, actor.ID) {
				t.Error("Success/CritSuccess should apply Flat-Footed to target")
			}
			if tt.Want == check.CriticalFailure && !actor.Conditions.HasRelative(condition.FlatFooted, target.ID) {
				t.Error("Crit Fail should apply Flat-Footed to actor")
			}
		}
	})

	// --- Intimidation: Demoralize ---
	t.Run("Demoralize", func(t *testing.T) {
		for _, tt := range standardDegrees {
			target.Conditions.Remove(condition.Frightened)
			target.TemporaryImmunities = make(map[string]int) 
			res := Demoralize(actor, target, tt.Roll)
			if res.Degree != tt.Want { t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want) }
			if tt.Want == check.CriticalSuccess && target.Conditions.Value(condition.Frightened) != 2 { t.Error("Crit Success: Frightened 2") }
			if tt.Want == check.Success && target.Conditions.Value(condition.Frightened) != 1 { t.Error("Success: Frightened 1") }
		}
	})

	// --- Medicine: Treat Wounds ---
	t.Run("TreatWounds", func(t *testing.T) {
		actor.SkillProficiencies[ability.SkillMedicine] = ability.Trained
		// Trained Actor at Level 1, mod 0 -> +3 bonus (2 + 1).
		// DC 10.
		// 20 -> 23 -> Crit Success
		// 10 -> 13 -> Success
		// 6  -> 9  -> Failure
		// 1  -> 4  -> Critical Failure
		
		for _, tt := range []DegreeTest{
			{"Critical Success", 20, check.CriticalSuccess},
			{"Success", 10, check.Success},
			{"Failure", 6, check.Failure},
			{"Critical Failure", 1, check.CriticalFailure},
		} {
			patient := entity.NewEntity("p", "P", 1)
			patient.Abilities = ability.AbilityScores{10, 10, 10, 10, 10, 10}
			patient.CurrentHP = 50
			healing, res := TreatWounds(actor, patient, dc, tt.Roll)
			if res.Degree != tt.Want { t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want) }
			if tt.Want == check.CriticalSuccess && healing == 0 { t.Error("Crit Success: should heal") }
			if tt.Want == check.CriticalFailure && patient.CurrentHP >= 50 { t.Error("Crit Fail: should deal damage") }
		}
	})

	// --- Stealth: Sneak ---
	t.Run("Sneak", func(t *testing.T) {
		for _, tt := range standardDegrees {
			actor.Conditions.Remove(condition.Hidden)
			res := Sneak(actor, target, tt.Roll)
			if res.Degree != tt.Want { t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want) }
			if tt.Want >= check.Success && !actor.Conditions.HasRelative(condition.Hidden, target.ID) {
				t.Error("Success: should be Hidden")
			}
		}
	})

	// --- Thievery: Pick Lock ---
	t.Run("PickLock", func(t *testing.T) {
		for _, tt := range standardDegrees {
			prog, res := PickLock(actor, dc, 3, tt.Roll)
			if res.Degree != tt.Want { t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want) }
			if tt.Want == check.CriticalSuccess && prog != 2 { t.Error("Crit Success: 2 progress") }
			if tt.Want == check.Success && prog != 1 { t.Error("Success: 1 progress") }
		}
	})

	// --- General: Recall Knowledge ---
	t.Run("RecallKnowledge", func(t *testing.T) {
		for _, tt := range standardDegrees {
			info, res := RecallKnowledge(actor, ability.SkillArcana, dc, tt.Roll)
			if res.Degree != tt.Want { t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want) }
			if tt.Want >= check.Success && info == "" { t.Error("Success: should give info") }
		}
	})
}
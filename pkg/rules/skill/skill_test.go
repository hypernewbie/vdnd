package skill

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
)

func setupTestEntities() (*entity.Entity, *entity.Entity) {
	actor := entity.NewEntity("actor", "Actor", 1)
	actor.Abilities = ability.AbilityScores{Strength: 10, Dexterity: 10, Constitution: 10, Intelligence: 10, Wisdom: 10, Charisma: 10}

	target := entity.NewEntity("target", "Target", 1)
	target.Abilities = ability.AbilityScores{Strength: 10, Dexterity: 10, Constitution: 10, Intelligence: 10, Wisdom: 10, Charisma: 10}

	return actor, target
}

type DegreeTest struct {
	Name string
	Roll int
	Want check.DegreeOfSuccess
}

// Baseline: DC 10, Mod 0.
var standardDegrees = []DegreeTest{
	{"Critical Success", 20, check.CriticalSuccess},
	{"Success", 10, check.Success},
	{"Failure", 5, check.Failure},
	{"Critical Failure", 1, check.CriticalFailure},
}

func TestExhaustiveSkills(t *testing.T) {
	actor, target := setupTestEntities()
	dc := 10

	// --- Acrobatics ---
	t.Run("Balance", func(t *testing.T) {
		for _, tt := range standardDegrees {
			actor.Conditions.Remove(condition.Prone)
			res := Balance(actor, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want == check.CriticalFailure && !actor.Conditions.Has(condition.Prone) {
				t.Error("Crit Fail should apply Prone")
			}
		}
	})

	t.Run("TumbleThrough", func(t *testing.T) {
		for _, tt := range standardDegrees {
			res, checkRes := TumbleThrough(actor, target, tt.Roll)
			if checkRes.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, checkRes.Degree, tt.Want)
			}
			if tt.Want >= check.Success && !res.Success {
				t.Error("Success should return success=true")
			}
		}
	})

	t.Run("ManeuverInFlight", func(t *testing.T) {
		for _, tt := range standardDegrees {
			dist, res := ManeuverInFlight(actor, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want >= check.Success && dist != actor.BaseSpeed {
				t.Errorf("Success should return %d, got %d", actor.BaseSpeed, dist)
			}
		}
	})

	t.Run("Squeeze", func(t *testing.T) {
		for _, tt := range standardDegrees {
			dist, res := Squeeze(actor, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want == check.CriticalSuccess && dist != 10 {
				t.Errorf("Crit Success should return 10, got %d", dist)
			}
			if tt.Want == check.Success && dist != 5 {
				t.Errorf("Success should return 5, got %d", dist)
			}
		}
	})

	// --- Athletics ---
	t.Run("Climb", func(t *testing.T) {
		for _, tt := range standardDegrees {
			actor.Conditions.Remove(condition.Prone)
			move, res := Climb(actor, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want == check.CriticalSuccess && move.Speed != 8 {
				t.Error("Crit Success speed should be 8")
			}
			if tt.Want == check.Success && move.Speed != 5 {
				t.Error("Success speed should be 5")
			}
		}
	})

	t.Run("Swim", func(t *testing.T) {
		for _, tt := range standardDegrees {
			move, res := Swim(actor, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want == check.CriticalSuccess && move.Speed != 10 {
				t.Error("Crit Success speed 10")
			}
			if tt.Want == check.Success && move.Speed != 5 {
				t.Error("Success speed 5")
			}
		}
	})

	t.Run("HighJump", func(t *testing.T) {
		for _, tt := range standardDegrees {
			dist, res := HighJump(actor, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want == check.Success && dist != 5 {
				t.Error("Success dist 5")
			}
		}
	})

	t.Run("LongJump", func(t *testing.T) {
		for _, tt := range standardDegrees {
			dist, res := LongJump(actor, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want == check.Success && dist != 10 {
				t.Error("Success dist Total (10)")
			}
		}
	})

	t.Run("Disarm", func(t *testing.T) {
		for _, tt := range standardDegrees {
			target.Conditions.Remove("DisarmWeakness")
			res := Disarm(actor, target, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want == check.Success && !target.Conditions.Has("DisarmWeakness") {
				t.Error("Success should apply DisarmWeakness")
			}
		}
	})

	t.Run("ForceOpen", func(t *testing.T) {
		for _, tt := range standardDegrees {
			res := ForceOpen(actor, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	t.Run("Grapple", func(t *testing.T) {
		for _, tt := range standardDegrees {
			target.Conditions.Remove(condition.Grabbed)
			target.Conditions.Remove(condition.Restrained)
			res := Grapple(actor, target, nil, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want == check.Success && !target.Conditions.Has(condition.Grabbed) {
				t.Error("Success: Grabbed")
			}
			if tt.Want == check.CriticalSuccess && !target.Conditions.Has(condition.Restrained) {
				t.Error("Crit Success: Restrained")
			}
		}
	})

	t.Run("Trip", func(t *testing.T) {
		for _, tt := range standardDegrees {
			target.Conditions.Remove(condition.Prone)
			actor.Conditions.Remove(condition.Prone)
			res := Trip(actor, target, nil, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want >= check.Success && !target.Conditions.Has(condition.Prone) {
				t.Error("Success: target Prone")
			}
		}
	})

	t.Run("Shove", func(t *testing.T) {
		for _, tt := range standardDegrees {
			actor.Conditions.Remove(condition.Prone)
			res := Shove(actor, target, nil, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	// --- Deception ---
	t.Run("CreateDiversion", func(t *testing.T) {
		for _, tt := range standardDegrees {
			actor.Conditions.RemoveRelative(condition.Hidden, target.ID)
			res := CreateADiversion(actor, []*entity.Entity{target}, tt.Roll)
			if res[0].Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res[0].Degree, tt.Want)
			}
			if tt.Want >= check.Success && !actor.Conditions.HasRelative(condition.Hidden, target.ID) {
				t.Error("Success: Hidden")
			}
		}
	})

	t.Run("Feint", func(t *testing.T) {
		for _, tt := range standardDegrees {
			target.Conditions.Remove(condition.FlatFooted)
			res := Feint(actor, target, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want >= check.Success && !target.Conditions.HasRelative(condition.FlatFooted, actor.ID) {
				t.Error("Success: FlatFooted relative")
			}
		}
	})

	t.Run("Impersonate", func(t *testing.T) {
		for _, tt := range standardDegrees {
			res := Impersonate(actor, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	t.Run("Lie", func(t *testing.T) {
		for _, tt := range standardDegrees {
			res := Lie(actor, target, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	// --- Diplomacy ---
	t.Run("GatherInfo", func(t *testing.T) {
		for _, tt := range standardDegrees {
			res := GatherInformation(actor, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	t.Run("MakeImpression", func(t *testing.T) {
		for _, tt := range standardDegrees {
			res := MakeAnImpression(actor, target, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	t.Run("Request", func(t *testing.T) {
		for _, tt := range standardDegrees {
			res := Request(actor, target, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	// --- Intimidation ---
	t.Run("Coerce", func(t *testing.T) {
		for _, tt := range standardDegrees {
			res := Coerce(actor, target, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	t.Run("Demoralize", func(t *testing.T) {
		for _, tt := range standardDegrees {
			target.TemporaryImmunities = make(map[string]int)
			target.Conditions.Remove(condition.Frightened)
			res := Demoralize(actor, target, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want == check.CriticalSuccess && target.Conditions.Value(condition.Frightened) != 2 {
				t.Error("Crit Success: Frightened 2")
			}
			if tt.Want == check.Success && target.Conditions.Value(condition.Frightened) != 1 {
				t.Error("Success: Frightened 1")
			}
		}
	})

	// --- Medicine ---
	t.Run("TreatWounds", func(t *testing.T) {
		actor.SkillProficiencies[ability.SkillMedicine] = ability.Trained
		for _, tt := range []DegreeTest{
			{"Critical Success", 20, check.CriticalSuccess},
			{"Success", 10, check.Success},
			{"Failure", 2, check.Failure},
			{"Critical Failure", 1, check.CriticalFailure},
		} {
			target.Conditions.Remove(condition.TreatWoundsImmunity)
			// Using dummy roller since we only check degree here
			res := TreatWoundsWithRoll(actor, target, dc, tt.Roll, &dice.SimpleRoller{})
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	t.Run("FirstAid", func(t *testing.T) {
		for _, tt := range standardDegrees {
			target.Conditions.Apply(condition.NewCondition(condition.Dying, "Injured"))
			res := AdministerFirstAid(actor, target, dc, true, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want >= check.Success && target.Conditions.Has(condition.Dying) {
				t.Error("Success should remove Dying")
			}
		}
	})

	t.Run("TreatPoison", func(t *testing.T) {
		for _, tt := range standardDegrees {
			target.Conditions.Remove("Next Poison Save")
			res := TreatPoison(actor, target, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want >= check.Success && !target.Conditions.Has("Next Poison Save") {
				t.Error("Success should apply bonus condition")
			}
		}
	})

	// --- Stealth ---
	t.Run("Sneak", func(t *testing.T) {
		for _, tt := range standardDegrees {
			actor.Conditions.RemoveRelative(condition.Hidden, target.ID)
			actor.Conditions.RemoveRelative(condition.Undetected, target.ID)
			actor.Conditions.RemoveRelative(condition.Observed, target.ID)
			res := Sneak(actor, target, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	t.Run("ConcealObject", func(t *testing.T) {
		for _, tt := range standardDegrees {
			res := ConcealObject(actor, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	// --- Thievery ---
	t.Run("PickLock", func(t *testing.T) {
		for _, tt := range standardDegrees {
			prog, res := PickLock(actor, dc, 3, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want == check.CriticalSuccess && prog != 2 {
				t.Error("Crit Success: 2 successes")
			}
			if tt.Want == check.Success && prog != 1 {
				t.Error("Success: 1 success")
			}
		}
	})

	t.Run("DisableDevice", func(t *testing.T) {
		for _, tt := range standardDegrees {
			res := DisableDevice(actor, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	t.Run("PalmObject", func(t *testing.T) {
		for _, tt := range standardDegrees {
			res := PalmObject(actor, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	t.Run("Steal", func(t *testing.T) {
		for _, tt := range standardDegrees {
			res := Steal(actor, target, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
		}
	})

	// --- General ---
	t.Run("RecallKnowledge", func(t *testing.T) {
		for _, tt := range standardDegrees {
			learned, res := RecallKnowledgeWithRoll(actor, ability.SkillArcana, dc, tt.Roll)
			if res.Degree != tt.Want {
				t.Errorf("%s: got %v, want %v", tt.Name, res.Degree, tt.Want)
			}
			if tt.Want >= check.Success && !learned {
				t.Error("Success should be learned")
			}
		}
	})
}

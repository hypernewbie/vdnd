package subsystem_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/subsystem"
)

func TestVictoryPointsBasic(t *testing.T) {
	sub := subsystem.NewSubsystem("test", "Test Challenge", subsystem.SubsystemCustom, 10, -5, 5)

	if err := sub.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Contribute some VP
	result := sub.Contribute("player1", check.Success, 2, 3)

	if result.VPEarned != 2 {
		t.Errorf("Expected 2 VP, got %d", result.VPEarned)
	}
	if sub.CurrentVP != 2 {
		t.Errorf("Expected current VP 2, got %d", sub.CurrentVP)
	}

	// Crit success
	result = sub.Contribute("player2", check.CriticalSuccess, 2, 3)
	if result.VPEarned != 3 {
		t.Errorf("Expected 3 VP on crit, got %d", result.VPEarned)
	}
}

func TestVictoryPointsCompletion(t *testing.T) {
	sub := subsystem.NewSubsystem("test", "Test", subsystem.SubsystemCustom, 5, -3, 10)
	sub.Start()

	// Reach target
	sub.Contribute("p1", check.CriticalSuccess, 2, 5)

	if !sub.IsComplete() {
		t.Error("Should be complete at target VP")
	}
	if !sub.IsSuccess() {
		t.Error("Should be successful")
	}
}

func TestVictoryPointsFailure(t *testing.T) {
	sub := subsystem.NewSubsystem("test", "Test", subsystem.SubsystemCustom, 10, -3, 10)
	sub.Start()

	// Three crit failures = -3 VP
	sub.Contribute("p1", check.CriticalFailure, 1, 2)
	sub.Contribute("p1", check.CriticalFailure, 1, 2)
	sub.Contribute("p1", check.CriticalFailure, 1, 2)

	if !sub.IsComplete() {
		t.Error("Should be complete at failure threshold")
	}
	if sub.IsSuccess() {
		t.Error("Should not be successful")
	}
}

func TestChase(t *testing.T) {
	chase := subsystem.NewChase("chase1", "Rooftop Chase", 10, 3, 8)

	pursuer := entity.NewEntity("guard", "City Guard", 5)
	pursuer.SkillProficiencies[ability.SkillAthletics] = ability.Trained

	quarry := entity.NewEntity("thief", "Nimble Thief", 4)
	quarry.SkillProficiencies[ability.SkillAcrobatics] = ability.Expert

	chase.AddPursuer(pursuer)
	chase.AddQuarry(quarry)
	chase.Start()

	if chase.GetGap() != 3 {
		t.Errorf("Initial gap should be 3, got %d", chase.GetGap())
	}

	// Quarry strides
	result := chase.TakeChaseAction("thief", subsystem.ChaseActionStride, "", 0)
	if result.PositionDelta != 1 {
		t.Errorf("Stride should move 1, got %d", result.PositionDelta)
	}

	if chase.GetGap() != 4 {
		t.Errorf("Gap should be 4 after quarry stride, got %d", chase.GetGap())
	}
}

func TestResearch(t *testing.T) {
	research := subsystem.NewResearch("library", "Ancient Library", 10)
	research.AddTopic("dragons", "Dragon Origins", "What created dragons?", "Dragons were born from primordial chaos", 3)
	research.AddTopic("weakness", "Dragon Weakness", "How to defeat dragons?", "Cold iron disrupts their magic", 5)
	research.Start()

	scholar := entity.NewEntity("scholar", "Scholar", 6)
	scholar.SkillProficiencies[ability.SkillArcana] = ability.Expert

	// First research check
	result := research.Research(scholar, ability.SkillArcana, 15, 18)

	t.Logf("Research result: %d VP earned", result.VPEarned)

	if !research.CanResearch(scholar.ID) {
		// Should still be able to research (max 2 per day)
		if research.ChecksToday[scholar.ID] >= research.MaxChecksPerDay {
			t.Log("At max checks, expected")
		}
	}
}

func TestInfluence(t *testing.T) {
	influence := subsystem.NewInfluence("council", "City Council Meeting", 5)

	mayor := entity.NewEntity("mayor", "Mayor Thornwood", 8)
	influence.AddTarget(mayor, 4,
		[]ability.SkillID{ability.SkillIntimidation}, // Resists intimidation
		[]ability.SkillID{ability.SkillDiplomacy})    // Weak to diplomacy

	influence.Start()

	diplomat := entity.NewEntity("bard", "Silver-Tongued Bard", 6)
	diplomat.SkillProficiencies[ability.SkillDiplomacy] = ability.Expert

	// Discover first
	discResult := influence.Discover(diplomat, "mayor", ability.SkillSociety, 15)
	t.Logf("Discovery: %s", discResult.Learned)

	// Try to influence with weakness skill
	infResult := influence.InfluenceTarget(diplomat, "mayor", ability.SkillDiplomacy, 18)
	t.Logf("Influence: %d/%d VP", infResult.TargetVP, infResult.TargetMax)
}

package subsystem_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/subsystem"
)

func TestImpossibleChase(t *testing.T) {
	// Quarry escapes if gap >= 5. Starting gap 2.
	// Pursuer pos 0, Quarry pos 2.
	chase := subsystem.NewChase("impossible", "The Impossible Chase", 10, 2, 5)
	
	pursuer := entity.NewEntity("p", "Pursuer", 1)
	quarry := entity.NewEntity("q", "Quarry", 1)
	
	chase.AddPursuer(pursuer)
	chase.AddQuarry(quarry)
	chase.Start()

	// Round 1: Pursuer fails to move (Dash Fail), Quarry Strides (+1)
	chase.TakeChaseAction("p", subsystem.ChaseActionDash, "", 10) // Failure, pos 0
	chase.TakeChaseAction("q", subsystem.ChaseActionStride, "", 0) // pos 2 -> 3
	
	// Gap = 3 - 0 = 3.
	if chase.GetGap() != 3 {
		t.Errorf("Expected gap 3, got %d", chase.GetGap())
	}

	// Round 2: Pursuer fails to move, Quarry Strides (+1)
	chase.AdvanceChaseRound()
	chase.TakeChaseAction("p", subsystem.ChaseActionDash, "", 10) // pos 0
	chase.TakeChaseAction("q", subsystem.ChaseActionStride, "", 0) // pos 3 -> 4
	
	// Gap = 4 - 0 = 4.
	if chase.GetGap() != 4 {
		t.Errorf("Expected gap 4, got %d", chase.GetGap())
	}

	// Round 3: Pursuer fails to move, Quarry Strides (+1)
	chase.AdvanceChaseRound()
	chase.TakeChaseAction("p", subsystem.ChaseActionDash, "", 10) // pos 0
	chase.TakeChaseAction("q", subsystem.ChaseActionStride, "", 0) // pos 4 -> 5
	
	// Gap = 5. Escape!
	if chase.GetGap() != 5 {
		t.Errorf("Expected gap 5, got %d", chase.GetGap())
	}
	
	if !chase.IsComplete() {
		t.Fatal("Chase should be complete")
	}
	if !chase.IsSuccess() {
		t.Error("Quarry should have successfully escaped")
	}
}

func TestInfluenceStalemate(t *testing.T) {
	// Target 10 VP. Round limit 5.
	inf := subsystem.NewSubsystem("inf", "Influence Target", subsystem.SubsystemInfluence, 10, -5, 5)
	inf.Start()
	
	actor := entity.NewEntity("pc", "Player", 1)
	
	// Round 1-5: Earn 1 VP each round
	for r := 1; r <= 5; r++ {
		inf.Contribute(actor.ID, 2, 1, 2) // Success (2) -> 1 VP
		if r < 5 {
			inf.AdvanceRound()
		}
	}
	
	if inf.CurrentVP != 5 {
		t.Errorf("Expected 5 VP, got %d", inf.CurrentVP)
	}
	
	// Try to advance beyond limit
	err := inf.AdvanceRound()
	if err == nil {
		t.Error("Should have failed to advance past round 5")
	}
	
	if inf.State != subsystem.StateFailure {
		t.Errorf("Expected Failure state, got %s", inf.State)
	}
}

func TestResearchDeadEnd(t *testing.T) {
	// Research topic requires Arcana
	res := subsystem.NewSubsystem("res", "Secret Lore", subsystem.SubsystemResearch, 5, 0, 10)
	res.Start()
	
	actor := entity.NewEntity("fighter", "Fighter", 1)
	// Fighter has no arcana proficiency (Untrained)
	
	// If the UI/CLI layer enforces "must be trained", we simulate it here
	canContribute := false
	if prof, ok := actor.SkillProficiencies[ability.SkillArcana]; ok && prof >= ability.Trained {
		canContribute = true
	}
	
	if canContribute {
		t.Error("Fighter should not be able to contribute to Arcana research")
	}
}

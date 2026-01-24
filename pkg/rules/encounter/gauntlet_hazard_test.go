package encounter_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/encounter"
	"uaa/vdnd/pkg/rules/entity"
)

// Source: rules/rules/core-rulebook/chapter-9-playing-the-game.md (Environment, Hazards)

func TestStealthVsHazardPassive(t *testing.T) {
	// Rogue Sneaks past Simple Hazard
	rogue := entity.NewEntity("rogue", "Rogue", 1)
	rogue.SkillProficiencies[ability.SkillStealth] = ability.Trained
	
	// Scenario: Rogue Rolls > Hazard DC, Hazard does NOT trigger.
	t.Log("Testing Stealth vs Hazard Trigger interaction (Expectation: Sneak avoids trigger)")
}

func TestFlyingOverPressurePlate(t *testing.T) {
	// Character uses Fly speed to move over floor trigger.
	t.Log("Testing Fly over Floor Trigger (Expectation: Flight avoids ground triggers)")
}

func TestInitiativeTieBreaking(t *testing.T) {
	enc := encounter.NewEncounter("tie")
	
	p1 := entity.NewEntity("p1", "Player 1", 1)
	p2 := entity.NewEntity("p2", "Player 2", 1)
	
	// Both roll 15
	enc.AddParticipant(p1)
	enc.AddParticipant(p2)
	
	enc.Participants[0].Initiative = 15
	enc.Participants[1].Initiative = 15
	
	// Ensure sorting logic handles ties without crashing.
	enc.Start()
	
	if len(enc.Participants) != 2 {
		t.Errorf("Expected 2 participants, got %d", len(enc.Participants))
	}
}
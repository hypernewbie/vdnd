package encounter_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/encounter"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/hazard"
	"uaa/vdnd/pkg/rules/item"
)

func TestHazardInterleavedTurns(t *testing.T) {
	e := encounter.NewEncounter("enc1")
	
	pc1 := entity.NewEntity("pc1", "PC1", 1)
	pc1.Position = "A"
	pc2 := entity.NewEntity("pc2", "PC2", 1)
	pc2.Position = "B"
	
	// Create a complex hazard
	h := hazard.NewHazard("h1", "Blades", 1)
	h.Position = "A"
	h.Routine = hazard.NewRoutine(3)
	h.Routine.AddAttack("Spin", 1, 10, dice.DieRoll{Count: 1, Sides: 8, Modifier: 4}, item.Slashing, 1)
	
	p1 := encounter.NewEntityParticipant(pc1)
	p1.Initiative = 20
	e.Participants = append(e.Participants, p1)

	ph := encounter.NewHazardParticipant(h)
	ph.Initiative = 15
	e.Participants = append(e.Participants, ph)

	p2 := encounter.NewEntityParticipant(pc2)
	p2.Initiative = 10
	e.Participants = append(e.Participants, p2)
	
	e.Start()
	
	// Current: PC1 (20)
	if e.GetCurrentParticipant().Entity.ID != "pc1" {
		t.Errorf("Expected PC1, got %s", e.GetCurrentParticipant().Entity.ID)
	}
	e.EndTurn()
	
	// Current: Hazard (15)
	p := e.GetCurrentParticipant()
	if p.Type != encounter.ParticipantHazard {
		t.Fatal("Expected Hazard participant")
	}
	
	// Run hazard turn
	// It should only target PC1 because PC1 is at position "A"
	res := h.TakeTurn([]*entity.Entity{pc1, pc2})
	
	foundPC1 := false
	foundPC2 := false
	for _, ar := range res.ActionResults {
		for _, tr := range ar.Targets {
			if tr.EntityID == "pc1" { foundPC1 = true }
			if tr.EntityID == "pc2" { foundPC2 = true }
		}
	}
	
	if !foundPC1 { t.Error("Hazard should have targeted PC1") }
	if foundPC2 { t.Error("Hazard should NOT have targeted PC2 (wrong position)") }
	
	e.EndTurn()
	
	// Current: PC2 (10)
	if e.GetCurrentParticipant().Entity.ID != "pc2" {
		t.Errorf("Expected PC2, got %s", e.GetCurrentParticipant().Entity.ID)
	}
}

func TestHazardResetAndRetrigger(t *testing.T) {
	h := hazard.NewHazard("h1", "Trap", 1)
	h.Routine = hazard.NewRoutine(1)
	h.Routine.AddReset("Reset") // Corrected signature
	
	h.IsTriggered = true
	
	// Turn 1
	res := h.TakeTurn(nil)
	if !res.WasReset {
		t.Error("Hazard should have reset")
	}
	if h.IsTriggered {
		t.Error("Hazard should no longer be triggered after reset")
	}
}
package entity_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/entity"
)

func TestDeriveFamiliarStats(t *testing.T) {
	master := entity.NewPC("wiz", "Wizard", 5, "Elf", "Wizard", "Scholar")
	master.Abilities.Constitution = 14

	fam := entity.NewEntity("cat", "Cat", 1)
	fam.Minion = &entity.MinionSettings{
		Type:     entity.MinionFamiliar,
		MasterID: master.ID,
	}

	// Master HP: say 40
	master.MaxHP = 40
	master.CurrentHP = 40

	// Derive
	err := fam.DeriveFamiliarStats(master)
	if err != nil {
		t.Fatalf("Derive failed: %v", err)
	}

	// Check Level
	if fam.Level != 5 {
		t.Errorf("Familiar should take master level 5, got %d", fam.Level)
	}

	// Check HP (5 * Level = 25)
	if fam.MaxHP != 25 {
		t.Errorf("Familiar HP should be 25, got %d", fam.MaxHP)
	}
}

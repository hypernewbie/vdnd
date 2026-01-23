package item

import (
	"testing"
	"uaa/vdnd/pkg/rules/trait"
)

func TestDamageType(t *testing.T) {
	if !Bludgeoning.IsPhysical() {
		t.Error("Bludgeoning should be physical")
	}
	if Fire.IsPhysical() {
		t.Error("Fire should not be physical")
	}
	if !Fire.IsEnergy() {
		t.Error("Fire should be energy")
	}
	if Slashing.IsEnergy() {
		t.Error("Slashing should not be energy")
	}
	if Mental.IsPhysical() || Mental.IsEnergy() {
		t.Error("Mental should be neither physical nor energy")
	}
}

func TestWeaponTraits(t *testing.T) {
	if !Dagger.IsAgile() {
		t.Error("Dagger should be agile")
	}
	if Longsword.IsAgile() {
		t.Error("Longsword should not be agile")
	}
	if !Rapier.IsFinesse() {
		t.Error("Rapier should be finesse")
	}
	if Greatsword.Hands != 2 {
		t.Errorf("Greatsword should need 2 hands, got %d", Greatsword.Hands)
	}
	if Longsword.GetReach() != 5 {
		t.Errorf("Longsword reach should be 5, got %d", Longsword.GetReach())
	}

	// Test reach trait
	glaive := NewWeapon("glaive", "Glaive", CategoryMartial, GroupPolearm, Longsword.Damage, Slashing, 2, trait.TraitReach)
	if glaive.GetReach() != 10 {
		t.Errorf("Glaive reach should be 10, got %d", glaive.GetReach())
	}
}

func TestArmor(t *testing.T) {
	if LeatherArmor.ACBonus != 1 {
		t.Errorf("Leather AC bonus should be 1, got %d", LeatherArmor.ACBonus)
	}
	if PlateArmor.ACBonus != 6 {
		t.Errorf("Plate AC bonus should be 6, got %d", PlateArmor.ACBonus)
	}

	// AppliedDexBonus
	if val := LeatherArmor.AppliedDexBonus(5); val != 4 {
		t.Errorf("Leather (DexCap 4) with +5 Dex should give +4, got %d", val)
	}
	if val := LeatherArmor.AppliedDexBonus(2); val != 2 {
		t.Errorf("Leather (DexCap 4) with +2 Dex should give +2, got %d", val)
	}
	if val := NoArmor.AppliedDexBonus(10); val != 10 {
		t.Errorf("Unarmored (No Cap) with +10 Dex should give +10, got %d", val)
	}
}

func TestArmorPenalties(t *testing.T) {
	// Chain Mail: STR 16, Check -2, Speed -5
	if pen := ChainMail.EffectiveCheckPenalty(12); pen != -2 {
		t.Errorf("Chain Mail with STR 12 should have -2 penalty, got %d", pen)
	}
	if pen := ChainMail.EffectiveCheckPenalty(16); pen != 0 {
		t.Errorf("Chain Mail with STR 16 should have 0 penalty, got %d", pen)
	}

	// Plate: STR 18, Speed -10
	if pen := PlateArmor.EffectiveSpeedPenalty(14); pen != -10 {
		t.Errorf("Plate with STR 14 should have -10 speed penalty, got %d", pen)
	}
	if pen := PlateArmor.EffectiveSpeedPenalty(18); pen != 0 {
		t.Errorf("Plate with STR 18 should have 0 speed penalty, got %d", pen)
	}
}

func TestRegistry(t *testing.T) {
	if _, ok := GetWeapon("longsword"); !ok {
		t.Error("Registry should contain longsword")
	}
	if _, ok := GetWeapon("lightsaber"); ok {
		t.Error("Registry should not contain lightsaber")
	}
	if _, ok := GetArmor("plate"); !ok {
		t.Error("Registry should contain plate")
	}
}

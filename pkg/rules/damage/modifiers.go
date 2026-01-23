package damage

import (
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

// CheckImmunity returns true if target is immune to damage type or traits
func CheckImmunity(target *entity.Entity, damageType string, isPrecision bool, traits []trait.TraitID) bool {
	if target.IsImmuneTo(damageType) {
		return true
	}
	if isPrecision && target.IsImmuneTo("precision") {
		return true
	}
	for _, tr := range traits {
		if target.IsImmuneTo(string(tr)) {
			return true
		}
	}
	return false
}

// CalculateWeakness returns total weakness value for damage type
func CalculateWeakness(target *entity.Entity, damageType string, traits []trait.TraitID) int {
	weakness := target.GetWeakness(damageType)

	// In PF2E, you usually take the highest applicable weakness,
	// but multiple weaknesses to different things can apply if they are distinct.
	// For simplicity, we'll follow the guideline: "highest that applies"
	// for a single damage instance of a single type.

	// Check traits too (e.g. weakness to silver)
	for _, tr := range traits {
		trW := target.GetWeakness(string(tr))
		if trW > weakness {
			weakness = trW
		}
	}

	return weakness
}

// CalculateResistance returns total resistance value for damage type
func CalculateResistance(target *entity.Entity, damageType string, traits []trait.TraitID) int {
	resEntry := target.GetResistanceEntry(damageType)

	// Check exceptions
	for _, ex := range resEntry.Except {
		for _, tr := range traits {
			if string(tr) == ex {
				return 0 // Resistance bypassed
			}
		}
		// Also check damage type if exception is a type (rare)
		if damageType == ex {
			return 0
		}
	}

	resistance := resEntry.Amount

	// Check trait-based resistances (e.g. resistance to physical)
	// This would require checking if damageType is physical etc.
	// For now we assume the map in Entity handles the mapping.

	return resistance
}

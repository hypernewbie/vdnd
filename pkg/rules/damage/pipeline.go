package damage

import (
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
)

// PipelineResult contains the outcome of processing damage
type PipelineResult struct {
	OriginalDamage    int
	FinalDamage       int
	WasImmune         bool
	WeaknessApplied   int
	ResistanceApplied int
	PreviousHP        int
	CurrentHP         int
	BecameDying       bool
	Died              bool
}

// ProcessDamage applies damage to an entity through the full pipeline
func ProcessDamage(target *entity.Entity, damage DamageInstance, isCritical bool) PipelineResult {
	result := PipelineResult{
		OriginalDamage: damage.Amount,
		PreviousHP:     target.CurrentHP,
	}

	// 1. Determine base damage (already in damage.Amount)
	// 2. Critical doubling (already handled in DamageRoll.RollCritical usually)

	// Calculate raw damage (immunity, weakness, resistance)
	rawResult := CalculateRawDamage(target, damage)
	
	result.WasImmune = rawResult.WasImmune
	result.WeaknessApplied = rawResult.WeaknessApplied
	result.ResistanceApplied = rawResult.ResistanceApplied
	result.FinalDamage = rawResult.FinalDamage

	if result.WasImmune {
		result.FinalDamage = 0
		result.CurrentHP = target.CurrentHP
		return result
	}

	amount := result.FinalDamage

	// 6. Apply to HP
	target.ApplyDamage(amount)
	result.CurrentHP = target.CurrentHP

	// 7. Check dying/death
	if target.CurrentHP <= 0 {
		if !target.Conditions.Has(condition.Dying) {
			target.CheckDying(isCritical)
			result.BecameDying = true
		} else {
			// Already dying, taking damage increases it
			target.CheckDying(isCritical)
		}
	}

	if target.IsDead() {
		result.Died = true
	}

	return result
}

// RawDamageResult contains the damage calculation before application
type RawDamageResult struct {
	FinalDamage       int
	WasImmune         bool
	WeaknessApplied   int
	ResistanceApplied int
}

// CalculateRawDamage calculates damage after immunity, weakness, and resistance
func CalculateRawDamage(target *entity.Entity, damage DamageInstance) RawDamageResult {
	result := RawDamageResult{}
	amount := damage.Amount

	// 3. Check immunity
	if CheckImmunity(target, string(damage.Type), damage.IsPrecision, damage.Traits) {
		result.WasImmune = true
		result.FinalDamage = 0
		return result
	}

	// 4. Apply weakness
	weakness := CalculateWeakness(target, string(damage.Type), damage.Traits)
	amount += weakness
	result.WeaknessApplied = weakness

	// 5. Apply resistance
	resistance := CalculateResistance(target, string(damage.Type), damage.Traits)
	amount -= resistance
	if amount < 0 {
		amount = 0
	}
	result.ResistanceApplied = resistance
	result.FinalDamage = amount

	return result
}

// ProcessMultipleDamage handles multiple damage types at once (e.g., sword + flaming)
func ProcessMultipleDamage(target *entity.Entity, damages []DamageInstance, isCritical bool) []PipelineResult {
	results := make([]PipelineResult, len(damages))
	for i, dmg := range damages {
		results[i] = ProcessDamage(target, dmg, isCritical)
	}
	return results
}

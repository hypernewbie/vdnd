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
	amount := damage.Amount

	// 2. Critical doubling (already handled in DamageRoll.RollCritical usually)
	// If it wasn't doubled yet, we might want to double it here,
	// but the standard is to pass in the rolled instance.

	// 3. Check immunity
	if CheckImmunity(target, string(damage.Type), damage.IsPrecision, damage.Traits) {
		result.WasImmune = true
		result.FinalDamage = 0
		result.CurrentHP = target.CurrentHP
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

// ProcessMultipleDamage handles multiple damage types at once (e.g., sword + flaming)
func ProcessMultipleDamage(target *entity.Entity, damages []DamageInstance, isCritical bool) []PipelineResult {
	results := make([]PipelineResult, len(damages))
	for i, dmg := range damages {
		results[i] = ProcessDamage(target, dmg, isCritical)
	}
	return results
}

package encounter

import (
	"uaa/vdnd/pkg/rules/combat"
	"uaa/vdnd/pkg/rules/damage"
	"uaa/vdnd/pkg/rules/entity"
)

type DamageResult struct {
	FinalDamage      int
	DamageBlocked    int
	WasLethal        bool
	PendingReactions *combat.ReactionQueue // nil if no reactions available
}

func ProcessDamageWithReactions(target *entity.Entity, dmg damage.DamageInstance, isCrit bool, encounter *Encounter) DamageResult {
	// Step 1: Calculate raw damage (resistances, weaknesses, immunities)
	rawResult := damage.CalculateRawDamage(target, dmg)

	// Step 2: Check for available reactions (Shield Block, etc.)
	event := combat.ReactionEvent{
		Trigger:    combat.TriggerOnDamageTaken,
		Target:     target,
		Damage:     rawResult.FinalDamage,
		DamageType: dmg.Type,
	}

	available := findAvailableReactions(target, event, encounter)
	if len(available) > 0 {
		// Return without applying damage - caller must resolve reactions first
		return DamageResult{
			FinalDamage:      rawResult.FinalDamage,
			PendingReactions: &combat.ReactionQueue{Event: event, Available: available},
		}
	}

	// Step 3: Apply damage directly if no reactions
	// Since we already calculated raw damage, we can apply it.
	// But damage.ProcessDamage does calculation + application.
	// We should just use target.ApplyDamage since we have the final amount.
	
	target.ApplyDamage(rawResult.FinalDamage)
	
	// Handle dying check manually since we bypassed ProcessDamage
	if target.CurrentHP <= 0 {
		target.CheckDying(isCrit)
	}

	return DamageResult{FinalDamage: rawResult.FinalDamage, WasLethal: target.CurrentHP <= 0}
}

func findAvailableReactions(target *entity.Entity, event combat.ReactionEvent, enc *Encounter) []combat.AvailableReaction {
	available := make([]combat.AvailableReaction, 0)

	// Check target's own reactions (Shield Block)
	shieldBlock := &combat.ShieldBlockReaction{}
	if shieldBlock.CanUse(target, event) {
		available = append(available, combat.AvailableReaction{Actor: target, Reaction: shieldBlock})
	}

	// TODO: Check allies for reactions like Liberating Step

	return available
}

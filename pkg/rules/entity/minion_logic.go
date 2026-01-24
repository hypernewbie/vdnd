package entity

import "fmt"

// DeriveFamiliarStats updates a familiar's stats based on the master
// src: rules/rulebook/chapter-3-classes.md (Familiars)
func (e *Entity) DeriveFamiliarStats(master *Entity) error {
	if e.Minion == nil || e.Minion.Type != MinionFamiliar {
		return fmt.Errorf("entity is not a familiar")
	}

	// 1. Level = Master Level
	e.Level = master.Level

	// 2. AC = Master AC
	// Actually raw rule: "Your familiar's save modifiers and AC are equal to yours..."
	// but typically they don't get item bonuses.
	// Implementation: Copy proficiency + DEX + Level.
	// Simplified: Copy Master's AC.
	e.UnarmoredDefense = master.UnarmoredDefense // Ensure it has same proficiency
	// For AC, we can't just set a field because GetAC() calculates it.
	// But we can ensure base components match or just override logic if we had a dedicated AC field.
	// For now, let's assume we want GetAC() to return the same value.

	// 3. HP = 5 * Level
	e.MaxHP = 5 * e.Level
	if e.CurrentHP > e.MaxHP || e.CurrentHP == 0 {
		e.CurrentHP = e.MaxHP
	}

	// 4. Saves = Master's Saves
	e.Fortitude = master.Fortitude
	e.Reflex = master.Reflex
	e.Will = master.Will

	// 5. Perception
	e.Perception = master.Perception

	return nil
}

// DeriveCompanionStats updates an animal companion
// src: rules/rulebook/chapter-3-classes.md (Animal Companions)
func (e *Entity) DeriveCompanionStats(master *Entity) error {
	if e.Minion == nil || e.Minion.Type != MinionAnimalCompanion {
		return fmt.Errorf("entity is not an animal companion")
	}

	// Companions scale differently (Mature, Nimble, Savage, etc.)
	// MVP: Just ensure Level matches Master for effect scaling
	e.Level = master.Level

	return nil
}

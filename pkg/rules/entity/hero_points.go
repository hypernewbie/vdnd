package entity

import (
	"fmt"

	"uaa/vdnd/pkg/rules/condition"
)

const MaxHeroPoints = 3

// GainHeroPoint adds a hero point, capped at 3.
func (e *Entity) GainHeroPoint() {
	e.heroPointsMu.Lock()
	defer e.heroPointsMu.Unlock()

	e.HeroPoints++
	if e.HeroPoints > MaxHeroPoints {
		e.HeroPoints = MaxHeroPoints
	}
}

// SpendHeroPoint removes one hero point.
// Returns an error if no hero points are available.
func (e *Entity) SpendHeroPoint() error {
	e.heroPointsMu.Lock()
	defer e.heroPointsMu.Unlock()

	if e.HeroPoints <= 0 {
		return fmt.Errorf("no hero points available (have %d)", e.HeroPoints)
	}
	e.HeroPoints--
	return nil
}

// HeroPointStabilize spends ALL hero points to stabilize when dying.
// Returns false if no hero points to spend or not dying.
// On success: sets HP to exactly 0, removes Dying condition, consumes all hero points.
func (e *Entity) HeroPointStabilize() bool {
	e.heroPointsMu.Lock()
	defer e.heroPointsMu.Unlock()

	if e.HeroPoints == 0 {
		return false
	}
	if !e.Conditions.Has(condition.Dying) {
		return false
	}

	e.HeroPoints = 0
	e.Conditions.Remove(condition.Dying)
	e.CurrentHP = 0 // Stabilize at exactly 0 HP
	return true
}

// CanUseHeroPoints returns true if the entity has hero points.
func (e *Entity) CanUseHeroPoints() bool {
	e.heroPointsMu.Lock()
	defer e.heroPointsMu.Unlock()

	return e.HeroPoints > 0
}
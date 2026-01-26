package entity

import (
	"errors"

	"uaa/vdnd/pkg/rules/condition"
)

const MaxHeroPoints = 3

// GainHeroPoint adds a hero point, capped at 3.
func (e *Entity) GainHeroPoint() {
	e.HeroPoints++
	if e.HeroPoints > MaxHeroPoints {
		e.HeroPoints = MaxHeroPoints
	}
}

// SpendHeroPoint removes one hero point.
// Returns an error if no hero points are available.
func (e *Entity) SpendHeroPoint() error {
	if e.HeroPoints <= 0 {
		return errors.New("no hero points available")
	}
	e.HeroPoints--
	return nil
}

// HeroPointStabilise spends ALL hero points to stabilise when dying.
// Returns false if no hero points to spend or not dying.
func (e *Entity) HeroPointStabilise() bool {
	if e.HeroPoints == 0 {
		return false
	}
	if !e.Conditions.Has(condition.Dying) {
		return false
	}

	e.HeroPoints = 0
	e.Conditions.Remove(condition.Dying)
	e.CurrentHP = 0 // Stabilise at exactly 0 HP
	return true
}

// CanUseHeroPoints returns true if the entity has hero points.
func (e *Entity) CanUseHeroPoints() bool {
	return e.HeroPoints > 0
}

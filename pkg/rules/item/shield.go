package item

import "uaa/vdnd/pkg/rules/trait"

type Shield struct {
	ID           string
	Name         string
	ACBonus      int // Circumstance bonus when raised (+1 or +2)
	SpeedPenalty int // Negative number, 0 for most shields
	Hardness     int // Damage absorbed before shield takes damage
	MaxHP        int // Total HP
	CurrentHP    int // Runtime HP tracking
	BT           int // Broken Threshold (typically MaxHP/2)
	Bulk         int
	Traits       trait.TraitSet

	// Runtime state
	IsRaised bool
}

func NewShield(id, name string, acBonus, hardness, maxHP, bulk int, traits ...trait.TraitID) *Shield {
	return &Shield{
		ID:        id,
		Name:      name,
		ACBonus:   acBonus,
		Hardness:  hardness,
		MaxHP:     maxHP,
		CurrentHP: maxHP,
		BT:        maxHP / 2,
		Bulk:      bulk,
		Traits:    traits,
	}
}

// IsBroken returns true if shield HP is at or below Broken Threshold
func (s *Shield) IsBroken() bool {
	return s.CurrentHP <= s.BT
}

// IsDestroyed returns true if shield HP is 0 or less
func (s *Shield) IsDestroyed() bool {
	return s.CurrentHP <= 0
}

// TakeDamage applies damage to the shield, returns actual damage taken
func (s *Shield) TakeDamage(amount int) int {
	if s.IsDestroyed() {
		return 0
	}
	actual := amount
	if actual > s.CurrentHP {
		actual = s.CurrentHP
	}
	s.CurrentHP -= actual
	return actual
}

// Repair restores HP up to max
func (s *Shield) Repair(amount int) {
	s.CurrentHP += amount
	if s.CurrentHP > s.MaxHP {
		s.CurrentHP = s.MaxHP
	}
}

// Reset clears runtime state (for new encounter)
func (s *Shield) Reset() {
	s.IsRaised = false
}

package trait

import (
	"strings"
)

// Registry holds all known traits
type Registry struct {
	traits map[TraitID]Trait
}

// NewRegistry creates a new empty registry
func NewRegistry() *Registry {
	return &Registry{
		traits: make(map[TraitID]Trait),
	}
}

// DefaultRegistry returns a registry pre-populated with core PF2E traits
func DefaultRegistry() *Registry {
	r := NewRegistry()

	// Weapon traits
	weaponTraits := map[TraitID]string{
		TraitAgile:       "Agile",
		TraitFinesse:     "Finesse",
		TraitReach:       "Reach",
		TraitDeadly:      "Deadly",
		TraitFatal:       "Fatal",
		TraitTwoHand:     "Two-Hand",
		TraitThrown:      "Thrown",
		TraitVersatile:   "Versatile",
		TraitBackstabber: "Backstabber",
		TraitForceful:    "Forceful",
		TraitSweep:       "Sweep",
	}
	for id, name := range weaponTraits {
		r.Register(NewTrait(id, name, CategoryWeapon))
	}

	// Damage types
	damageTypes := []TraitID{
		TraitBludgeoning, TraitPiercing, TraitSlashing,
		TraitFire, TraitCold, TraitElectricity, TraitAcid, TraitSonic,
		TraitForce, TraitMental, TraitPoison, TraitPositive, TraitNegative,
		"chaotic", "evil", "good", "lawful", "bleed", "precision",
	}
	for _, id := range damageTypes {
		name := strings.Title(string(id))
		r.Register(NewTrait(id, name, CategoryDamage))
	}

	// Action traits
	actionTraits := map[TraitID]string{
		TraitAttack:      "Attack",
		TraitMove:        "Move",
		TraitManipulate:  "Manipulate",
		TraitConcentrate: "Concentrate",
		TraitAuditory:    "Auditory",
		TraitVisual:      "Visual",
		TraitLinguistic:  "Linguistic",
		TraitMental:      "Mental",
		TraitEmotion:     "Emotion",
		TraitFear:        "Fear",
	}
	for id, name := range actionTraits {
		r.Register(NewTrait(id, name, CategoryAction))
	}

	// Creature traits
	creatureTraits := []string{
		"humanoid", "beast", "construct", "undead", "fiend", "celestial",
		"dragon", "elemental", "fey", "giant", "ooze", "plant", "mindless", "incorporeal",
	}
	for _, s := range creatureTraits {
		id := TraitID(s)
		name := strings.Title(s)
		r.Register(NewTrait(id, name, CategoryCreature))
	}

	return r
}

// Get retrieves a trait by ID, returns (trait, found)
func (r *Registry) Get(id TraitID) (Trait, bool) {
	t, ok := r.traits[id]
	return t, ok
}

// Register adds a new trait to the registry
func (r *Registry) Register(t Trait) {
	r.traits[t.ID] = t
}

// Has checks if a trait ID is known
func (r *Registry) Has(id TraitID) bool {
	_, ok := r.traits[id]
	return ok
}

// AllInCategory returns all traits of a given category
func (r *Registry) AllInCategory(cat TraitCategory) []Trait {
	var result []Trait
	for _, t := range r.traits {
		if t.Category == cat {
			result = append(result, t)
		}
	}
	return result
}

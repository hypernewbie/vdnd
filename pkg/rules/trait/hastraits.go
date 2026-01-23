package trait

// HasTraits is implemented by anything with traits
type HasTraits interface {
	Traits() []TraitID
	HasTrait(id TraitID) bool
}

// TraitSet is a helper type for storing traits on a struct
type TraitSet []TraitID

func (ts TraitSet) Traits() []TraitID {
	return ts
}

func (ts TraitSet) HasTrait(id TraitID) bool {
	for _, t := range ts {
		if t == id {
			return true
		}
	}
	return false
}

// HasAnyTrait checks if the thing has any of the given traits
func HasAnyTrait(h HasTraits, ids ...TraitID) bool {
	for _, id := range ids {
		if h.HasTrait(id) {
			return true
		}
	}
	return false
}

// HasAllTraits checks if the thing has all of the given traits
func HasAllTraits(h HasTraits, ids ...TraitID) bool {
	for _, id := range ids {
		if !h.HasTrait(id) {
			return false
		}
	}
	return true
}

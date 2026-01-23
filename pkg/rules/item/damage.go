package item

// DamageType represents the type of damage dealt
type DamageType string

const (
	Bludgeoning DamageType = "bludgeoning"
	Piercing    DamageType = "piercing"
	Slashing    DamageType = "slashing"
	Fire        DamageType = "fire"
	Cold        DamageType = "cold"
	Electricity DamageType = "electricity"
	Acid        DamageType = "acid"
	Sonic       DamageType = "sonic"
	Force       DamageType = "force"
	Mental      DamageType = "mental"
	Poison      DamageType = "poison"
	Positive    DamageType = "positive"
	Negative    DamageType = "negative"
	Bleed       DamageType = "bleed"
	Precision   DamageType = "precision"
)

// IsPhysical returns true for bludgeoning, piercing, slashing
func (d DamageType) IsPhysical() bool {
	return d == Bludgeoning || d == Piercing || d == Slashing
}

// IsEnergy returns true for fire, cold, electricity, acid, sonic
func (d DamageType) IsEnergy() bool {
	return d == Fire || d == Cold || d == Electricity || d == Acid || d == Sonic
}

package trait

// TraitID is a unique identifier for a trait (lowercase, hyphenated)
type TraitID string

// Common trait IDs as constants for type safety
const (
	// Weapon traits
	TraitAgile       TraitID = "agile"
	TraitFinesse     TraitID = "finesse"
	TraitReach       TraitID = "reach"
	TraitDeadly      TraitID = "deadly"
	TraitFatal       TraitID = "fatal"
	TraitTwoHand     TraitID = "two-hand"
	TraitThrown      TraitID = "thrown"
	TraitVersatile   TraitID = "versatile"
	TraitBackstabber TraitID = "backstabber"
	TraitForceful    TraitID = "forceful"
	TraitSweep       TraitID = "sweep"
	TraitBackswing   TraitID = "backswing"

	// Action traits
	TraitAttack      TraitID = "attack"
	TraitMove        TraitID = "move"
	TraitManipulate  TraitID = "manipulate"
	TraitConcentrate TraitID = "concentrate"
	TraitAuditory    TraitID = "auditory"
	TraitVisual      TraitID = "visual"
	TraitLinguistic  TraitID = "linguistic"
	TraitMental      TraitID = "mental"
	TraitEmotion     TraitID = "emotion"
	TraitFear        TraitID = "fear"
	TraitSecret      TraitID = "secret"

	// Damage types (some are also traits)
	TraitBludgeoning TraitID = "bludgeoning"
	TraitPiercing    TraitID = "piercing"
	TraitSlashing    TraitID = "slashing"
	TraitFire        TraitID = "fire"
	TraitCold        TraitID = "cold"
	TraitElectricity TraitID = "electricity"
	TraitAcid        TraitID = "acid"
	TraitSonic       TraitID = "sonic"
	TraitForce       TraitID = "force"
	TraitPoison      TraitID = "poison"
	TraitPositive    TraitID = "positive"
	TraitNegative    TraitID = "negative"

	// Class traits
	TraitFighter TraitID = "fighter"
	TraitWizard  TraitID = "wizard"
	TraitRogue   TraitID = "rogue"
	TraitCleric  TraitID = "cleric"

	// Ancestry traits
	TraitHuman TraitID = "human"
	TraitElf   TraitID = "elf"
	TraitDwarf TraitID = "dwarf"

	// Misc
	TraitHealing TraitID = "healing"

	// Hazard traits
	TraitMechanical    TraitID = "mechanical"
	TraitTrap          TraitID = "trap"
	TraitHaunt         TraitID = "haunt"
	TraitEnvironmental TraitID = "environmental"
)

type TraitCategory int

const (
	CategoryWeapon TraitCategory = iota
	CategoryArmor
	CategoryAction
	CategorySpell
	CategoryDamage
	CategoryCreature
	CategoryCondition
	CategoryGeneral
	CategoryRarity
	CategoryTradition
	CategorySchool
)

const (
	// Rarity traits
	TraitCommon   TraitID = "common"
	TraitUncommon TraitID = "uncommon"
	TraitRare     TraitID = "rare"
	TraitUnique   TraitID = "unique"

	// Magic Traditions
	TraitArcane   TraitID = "arcane"
	TraitDivine   TraitID = "divine"
	TraitOccult   TraitID = "occult"
	TraitPrimal   TraitID = "primal"

	// Magic Schools
	TraitAbjuration   TraitID = "abjuration"
	TraitConjuration  TraitID = "conjuration"
	TraitDivination   TraitID = "divination"
	TraitEnchantment  TraitID = "enchantment"
	TraitEvocation    TraitID = "evocation"
	TraitIllusion     TraitID = "illusion"
	TraitNecromancy   TraitID = "necromancy"
	TraitTransmutation TraitID = "transmutation"
)

type Trait struct {
	ID          TraitID
	Name        string // Display name: "Agile"
	Description string // Full rules text
	Category    TraitCategory
	Parameter   string // "d10", "20", "slashing", etc.
}

// NewTrait creates a trait with basic info
func NewTrait(id TraitID, name string, category TraitCategory) Trait {
	return Trait{
		ID:       id,
		Name:     name,
		Category: category,
	}
}

// NewParameterizedTrait creates a trait with a parameter
func NewParameterizedTrait(id TraitID, name string, category TraitCategory, param string) Trait {
	return Trait{
		ID:        id,
		Name:      name,
		Category:  category,
		Parameter: param,
	}
}

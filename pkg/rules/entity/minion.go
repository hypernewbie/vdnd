package entity

type MinionType int

const (
	MinionFamiliar MinionType = iota
	MinionAnimalCompanion
	MinionSummon
)

// MinionSettings defines the configurables for a minion
type MinionSettings struct {
	Type           MinionType
	MasterID       string
	IsCommanded    bool // Reset every round
	ActionsPerTurn int  // Usually 2 if commanded
}

// MinionAbility tracks specific minion powers (e.g. "Fly Speed", "Manual Dexterity")
type MinionAbility string

const (
	MinionAbilityFly        MinionAbility = "fly"
	MinionAbilityDarkvision MinionAbility = "darkvision"
	MinionAbilitySpeech     MinionAbility = "speech"
	MinionAbilityTouch      MinionAbility = "touch_telepathy"
)

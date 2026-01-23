package entity

import "fmt"

type Size int

const (
	Tiny Size = iota
	Small
	Medium
	Large
	Huge
	Gargantuan
)

// Space returns the space occupied in feet
func (s Size) Space() int {
	switch s {
	case Tiny:
		return 2 // PF2E says < 5ft, often 2.5ft, using 2 for grid simplicity if needed or 0
	case Small, Medium:
		return 5
	case Large:
		return 10
	case Huge:
		return 15
	case Gargantuan:
		return 20
	default:
		return 5
	}
}

// Reach returns base reach for a tall creature
func (s Size) Reach() int {
	switch s {
	case Tiny:
		return 0
	case Small, Medium:
		return 5
	case Large:
		return 10
	case Huge:
		return 15
	case Gargantuan:
		return 20
	default:
		return 5
	}
}

func (s Size) String() string {
	switch s {
	case Tiny:
		return "Tiny"
	case Small:
		return "Small"
	case Medium:
		return "Medium"
	case Large:
		return "Large"
	case Huge:
		return "Huge"
	case Gargantuan:
		return "Gargantuan"
	default:
		return fmt.Sprintf("Size(%d)", s)
	}
}

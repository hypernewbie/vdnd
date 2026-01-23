package ability

type IntervalUnit int

const (
	IntervalRounds IntervalUnit = iota
	IntervalMinutes
	IntervalHours
	IntervalDays
)

func (u IntervalUnit) String() string {
	return [...]string{"Rounds", "Minutes", "Hours", "Days"}[u]
}

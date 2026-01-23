package combat

// CalculateMAP returns the penalty for the Nth attack
func CalculateMAP(attackNumber int, isAgile bool) int {
	if attackNumber <= 1 {
		return 0
	}
	if isAgile {
		if attackNumber == 2 {
			return -4
		}
		return -8
	}
	if attackNumber == 2 {
		return -5
	}
	return -10
}

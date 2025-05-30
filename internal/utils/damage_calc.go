package utils

// CalculateDamage tính toán lượng damage gây ra dựa trên ATK, DEF và CRIT%
func CalculateDamage(atk int, def int, useCrit bool) int {
	if useCrit {
		atk = int(float64(atk) * 1.2)
	}
	dmg := atk - def
	if dmg < 0 {
		dmg = 0
	}
	return dmg
}

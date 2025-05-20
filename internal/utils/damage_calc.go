package utils

import (
	"math/rand"
	"time"
)

// Initialize random seed once
func init() {
	rand.Seed(time.Now().UnixNano())
}

// CalculateDamage tính toán lượng damage gây ra dựa trên ATK, DEF và CRIT%
func CalculateDamage(atk int, def int, critChance float64) int {
	// CRIT: nếu xảy ra, tăng sát thương 20%
	actualATK := atk
	if rand.Float64() < critChance {
		actualATK = int(float64(atk) * 1.2)
	}

	dmg := actualATK - def
	if dmg < 0 {
		dmg = 0
	}
	return dmg
}

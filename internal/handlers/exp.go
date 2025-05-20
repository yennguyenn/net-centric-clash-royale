package handlers

import (
	"fmt"
	"tcr_project/models"
)

func AddExp(player *models.Player, expGain int) {
	player.EXP += expGain
	nextLevelExp := 100 + (player.Level-1)*10

	for player.EXP >= nextLevelExp {
		player.EXP -= nextLevelExp
		player.Level++
		fmt.Printf("🌟 %s leveled up! Now level %d\n", player.Username, player.Level)

		// Tăng chỉ số Tower theo cấp độ (10%)
		for i := range player.Towers {
			player.Towers[i].HP = int(float64(player.Towers[i].HP) * 1.1)
			player.Towers[i].ATK = int(float64(player.Towers[i].ATK) * 1.1)
			player.Towers[i].DEF = int(float64(player.Towers[i].DEF) * 1.1)
		}
	}
}

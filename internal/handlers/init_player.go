package handlers

import (
	"fmt"
	"net-centric-clash-royale/internal/models"
	"net-centric-clash-royale/internal/utils"
)

// InitNewPlayer sets up default game state for a newly registered player
func InitNewPlayer(player *models.Player) error {
	towers, err := utils.LoadPlayerTowers()
	if err != nil {
		return fmt.Errorf("failed to load towers: %w", err)
	}
	player.Towers = towers
	player.Troops = []models.Troop{}
	player.Mana = 10
	player.Level = 1
	player.EXP = 0

	return nil
}

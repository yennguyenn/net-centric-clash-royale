package models

// import (
// 	"log"
// 	// "net-centric-clash-royale/internal/utils"
// )

type Player struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	EXP      int     `json:"exp"`
	Level    int     `json:"level"`
	Mana     int     `json:"mana"`
	Towers   []Tower `json:"towers"`
	Troops   []Troop `json:"troops"`
}

// Hàm tạo Player mới
// func NewPlayer(username, password string) *Player {
// 	// towers, err := utils.LoadPlayerTowers()
// 	// if err != nil {
// 	// 	log.Fatalf("❌ Failed to load towers for player %s: %v", username, err)
// 	// }
// 	return &Player{
// 		Username: username,
// 		Password: password,
// 		// Towers:   towers,
// 		Troops: []Troop{},
// 		Mana:   10,
// 		Level:  1,
// 		EXP:    0,
// 	}
// }

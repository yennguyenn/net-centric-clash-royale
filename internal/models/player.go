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

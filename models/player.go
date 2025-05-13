package models

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
func NewPlayer(username, password string) *Player {
	return &Player{
		Username: username,
		Password: password,
		EXP:      0,
		Level:    1,
		Mana:     5,
		Towers:   []Tower{},
		Troops:   []Troop{},
	}
}

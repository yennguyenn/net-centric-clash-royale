package models

type Player struct {
	Username      string    `json:"username"`
	Password      string    `json:"password"`
	EXP           int       `json:"exp"`
	Level         int       `json:"level"`
	Mana          int       `json:"mana"`
	Towers        []Tower   `json:"towers"`
	Troops        []Troop   `json:"troops"`
	GameModeTimed bool      `json:"-"` // Added for game mode selection, not persisted
	WaitChannel   chan bool `json:"-"` // Channel for signaling match found (true) or timeout (false), not persisted
}

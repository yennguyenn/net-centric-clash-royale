package models

type Troop struct {
	Name    string `json:"name"`
	HP      int    `json:"hp"`
	ATK     int    `json:"atk"`
	DEF     int    `json:"def"`
	Mana    int    `json:"mana"`
	EXP     int    `json:"exp"`
	Special string `json:"special,omitempty"`
}

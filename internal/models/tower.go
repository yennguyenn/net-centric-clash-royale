package models

type Tower struct {
	Type string  `json:"type"`
	HP   int     `json:"hp"`
	ATK  int     `json:"atk"`
	DEF  int     `json:"def"`
	CRIT float64 `json:"crit"`
	EXP  int     `json:"exp"`
}

package utils

import (
	"encoding/json"
	"fmt"
	"net-centric-clash-royale/internal/models"
	"os"
	"path/filepath"
)

// LoadTowersFromFile loads tower definitions from data/tower.json
func LoadTowersFromFile() ([]models.Tower, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(cwd, "data", "tower.json")
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open tower.json: %w", err)
	}
	defer file.Close()

	var towers []models.Tower
	if err := json.NewDecoder(file).Decode(&towers); err != nil {
		return nil, fmt.Errorf("failed to decode tower.json: %w", err)
	}
	return towers, nil
}

// LoadPlayerTowers constructs a slice of 3 towers (2 Guard Tower clones + 1 King Tower)
func LoadPlayerTowers() ([]models.Tower, error) {
	towers, err := LoadTowersFromFile()
	if err != nil {
		return nil, err
	}
	var guards []models.Tower
	var king models.Tower
	for _, t := range towers {
		switch t.Type {
		case "Guard Tower":
			guards = append(guards, t)
		case "King Tower":
			king = t
		}
	}

	if len(guards) == 0 || king.Type == "" {
		return nil, fmt.Errorf("⚠️ tower.json missing Guard Tower or King Tower")
	}

	// Clone the same Guard Tower twice to avoid shared pointer issue
	return []models.Tower{
		cloneTower(guards[0]),
		cloneTower(guards[0]),
		king,
	}, nil
}

// cloneTower creates a deep copy of a tower
func cloneTower(t models.Tower) models.Tower {
	return models.Tower{
		Type: t.Type,
		HP:   t.HP,
		ATK:  t.ATK,
		DEF:  t.DEF,
		CRIT: t.CRIT,
		EXP:  t.EXP,
	}
}

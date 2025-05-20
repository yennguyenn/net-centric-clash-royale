package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"tcr_project/internal/models"
)

func LoadTroopsFromFile(path string) ([]models.Troop, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open troop file: %w", err)
	}
	defer file.Close()

	var troops []models.Troop
	if err := json.NewDecoder(file).Decode(&troops); err != nil {
		return nil, fmt.Errorf("failed to decode troops: %w", err)
	}
	return troops, nil
}

func GetTroopByName(troops []models.Troop, name string) *models.Troop {
	for _, t := range troops {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

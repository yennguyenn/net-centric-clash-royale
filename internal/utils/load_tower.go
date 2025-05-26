package utils

import (
	"encoding/json"
	"fmt"
	"net-centric-clash-royale/internal/models"
	"os"
	"path/filepath"
)

//	func LoadTowersFromFile() ([]models.Tower, error) {
//		cwd, err := os.Getwd()
//		if err != nil {
//			return nil, err
//		}
//		path := filepath.Join(cwd, "data", "tower.json")
//		file, err := os.Open(path)
//		if err != nil {
//			return nil, fmt.Errorf("failed to open tower.json: %w", err)
//		}
//		defer file.Close()
//		var towers []models.Tower
//		if err := json.NewDecoder(file).Decode(&towers); err != nil {
//			return nil, fmt.Errorf("failed to decode tower.json: %w", err)
//		}
//		return towers, nil
//	}
//
//	func LoadPlayerTowers() ([]models.Tower, error) {
//		towers, err := LoadTowersFromFile()
//		if err != nil {
//			return nil, err
//		}
//		var guards []models.Tower
//		var king models.Tower
//		for _, t := range towers {
//			switch t.Type {
//			case "Guard Tower":
//				guards = append(guards, t)
//			case "King Tower":
//				king = t
//			}
//		}
//		if len(guards) == 0 || king.Type == "" {
//			return nil, fmt.Errorf("⚠ tower.json missing Guard Tower or King Tower")
//		}
//		return []models.Tower{guards[0], guards[0], king}, nil
//	}
func LoadPlayerTowers() ([]models.Tower, error) {
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

	return []models.Tower{guards[0], guards[0], king}, nil
}

package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"net-centric-clash-royale/internal/models"
)

func LoadTroopsFromFile(relPath string) ([]models.Troop, error) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("❌ os.Getwd failed:", err)
		return nil, err
	}

	absPath := filepath.Join(cwd, relPath)
	fmt.Println("🛠️ Attempting to load troop.json from:", absPath)

	file, err := os.Open(absPath)
	if err != nil {
		fmt.Println("❌ File open failed:", err)
		return nil, err
	}
	defer file.Close()

	var troops []models.Troop
	if err := json.NewDecoder(file).Decode(&troops); err != nil {
		fmt.Println("❌ JSON decode failed:", err)
		return nil, err
	}

	fmt.Println("✅ Loaded", len(troops), "troops")
	return troops, nil
}

// func GetTroopByName(troops []models.Troop, name string) *models.Troop {
// 	for _, t := range troops {
// 		if t.Name == name {
// 			return &t
// 		}
// 	}
// 	return nil
// }

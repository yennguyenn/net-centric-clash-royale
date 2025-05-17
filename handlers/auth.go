package handlers

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tcr_project/models"
	"tcr_project/network"
)

var userDataFile = filepath.Join("data", "players.json")

func Authenticate(conn net.Conn, players *map[string]*models.Player, mutex *sync.Mutex) *models.Player {
	for {
		network.SendPDU(conn, "menu", "📋 Do you want to (1) Register or (2) Login? Enter 1 or 2:")
		pdu, err := network.ReadPDU(conn)
		if err != nil {
			fmt.Println("❌ Failed to read PDU:", err)
			return nil
		}
		choice := strings.TrimSpace(pdu.Payload)

		switch choice {
		case "1":
			player := register(conn, players, mutex)
			if player != nil {
				return player
			}
		case "2":
			player := login(conn, players, mutex)
			if player != nil {
				return player
			}
		default:
			network.SendPDU(conn, "error", "❗ Invalid option. Please enter 1 or 2.")
		}
	}
}

func register(conn net.Conn, players *map[string]*models.Player, mutex *sync.Mutex) *models.Player {
	network.SendPDU(conn, "input", "🆕 Enter a new username:")
	usernamePDU, err := network.ReadPDU(conn)
	if err != nil {
		return nil
	}
	username := strings.TrimSpace(usernamePDU.Payload)
	network.SendPDU(conn, "input", "🔒 Enter a password:")
	passwordPDU, err := network.ReadPDU(conn)
	if err != nil {
		return nil
	}
	password := strings.TrimSpace(passwordPDU.Payload)

	mutex.Lock()
	defer mutex.Unlock()

	if _, exists := (*players)[username]; exists {
		network.SendPDU(conn, "error", "❌ Username already exists.")
		return nil
	}

	player := models.NewPlayer(username, password)
	(*players)[username] = player
	savePlayers(*players)

	network.SendPDU(conn, "success", "✅ Registration successful!")
	return player
}

func login(conn net.Conn, players *map[string]*models.Player, mutex *sync.Mutex) *models.Player {
	network.SendPDU(conn, "input", "👤 Enter username:")
	usernamePDU, err := network.ReadPDU(conn)
	if err != nil {
		return nil
	}
	username := strings.TrimSpace(usernamePDU.Payload)
	network.SendPDU(conn, "input", "🔑 Enter password:")
	passwordPDU, err := network.ReadPDU(conn)
	if err != nil {
		return nil
	}
	password := strings.TrimSpace(passwordPDU.Payload)
	mutex.Lock()
	defer mutex.Unlock()

	if player, exists := (*players)[username]; exists && player.Password == password {
		network.SendPDU(conn, "success", "✅ Login successful!")
		return player
	}

	network.SendPDU(conn, "error", "❌ Invalid username or password.")
	return nil
}

func savePlayers(players map[string]*models.Player) {
	file, err := os.Create(userDataFile)
	if err != nil {
		fmt.Printf("❌ Failed to save player data: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(players); err != nil {
		fmt.Printf("❌ Failed to encode player data: %v\n", err)
	}
}

func LoadPlayers() (map[string]*models.Player, error) {
	file, err := os.Open(userDataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*models.Player), nil
		}
		return nil, err
	}
	defer file.Close()

	var players map[string]*models.Player
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&players); err != nil {
		return nil, err
	}
	return players, nil
}

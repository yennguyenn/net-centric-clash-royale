package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tcr_project/models"
)

var userDataFile = filepath.Join("data", "players.json")

func Authenticate(conn net.Conn, players *map[string]*models.Player, mutex *sync.Mutex) *models.Player {
	reader := bufio.NewReader(conn)

	for {
		fmt.Fprintln(conn, "📋 Do you want to (1) Register or (2) Login? Enter 1 or 2:")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

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
			fmt.Fprintln(conn, "❗ Invalid option. Please enter 1 or 2.")
		}
	}
}

func register(conn net.Conn, players *map[string]*models.Player, mutex *sync.Mutex) *models.Player {
	reader := bufio.NewReader(conn)
	fmt.Fprintln(conn, "🆕 Enter a new username:")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Fprintln(conn, "🔒 Enter a password:")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	mutex.Lock()
	defer mutex.Unlock()

	if _, exists := (*players)[username]; exists {
		fmt.Fprintln(conn, "❌ Username already exists.")
		return nil
	}

	player := models.NewPlayer(username, password)
	(*players)[username] = player
	savePlayers(*players)

	fmt.Fprintln(conn, "✅ Registration successful!")
	return player
}

func login(conn net.Conn, players *map[string]*models.Player, mutex *sync.Mutex) *models.Player {
	reader := bufio.NewReader(conn)
	fmt.Fprintln(conn, "👤 Enter username:")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Fprintln(conn, "🔑 Enter password:")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	mutex.Lock()
	defer mutex.Unlock()

	if player, exists := (*players)[username]; exists && player.Password == password {
		fmt.Fprintln(conn, "✅ Login successful!")
		return player
	}

	fmt.Fprintln(conn, "❌ Invalid username or password.")
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
		// Nếu file chưa tồn tại, trả về map rỗng
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

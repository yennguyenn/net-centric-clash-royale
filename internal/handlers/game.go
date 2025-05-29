package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"net-centric-clash-royale/internal/models"
	"net-centric-clash-royale/internal/network"
	"net-centric-clash-royale/internal/utils"
)

type GameSession struct {
	Player1      *models.Player
	Player2      *models.Player
	Conn1        net.Conn
	Conn2        net.Conn
	GameOver     bool
	TurnOwner    *models.Player
	Mutex        *sync.Mutex
	GameTimer    *GameTimer // Add GameTimer to the session
	IsTimedGame  bool       // New field to indicate if the game is timed
	gameOverChan chan bool  // Channel to signal game completion to the caller
}

// StartGameSession now accepts an isTimedGame boolean and returns a channel
// that will be closed or sent a signal when the game is over.
func StartGameSession(p1, p2 *models.Player, conn1, conn2 net.Conn, isTimedGame bool) chan bool {
	session := &GameSession{
		Player1:      p1,
		Player2:      p2,
		Conn1:        conn1,
		Conn2:        conn2,
		GameOver:     false,
		TurnOwner:    p1,
		Mutex:        &sync.Mutex{},
		IsTimedGame:  isTimedGame,     // Set the game mode
		gameOverChan: make(chan bool), // Initialize the game over channel
	}

	troops, err := utils.LoadTroopsFromFile("data/troop.json")
	if err != nil {
		fmt.Printf("❌ Failed to load troops.json: %v\n", err)
		network.SendPDU(conn1, "error", "❌ Server error: cannot load troop data.")
		network.SendPDU(conn2, "error", "❌ Server error: cannot load troop data.")
		conn1.Close() // Close connections on error
		conn2.Close()
		close(session.gameOverChan) // Signal immediate game over
		return session.gameOverChan
	}
	if len(troops) < 3 {
		fmt.Printf("⚠️ Not enough troops in data file. Only %d troops found\n", len(troops))
		network.SendPDU(conn1, "error", "⚠️ Not enough troop data on server.")
		network.SendPDU(conn2, "error", "⚠️ Not enough troop data on server.")
		conn1.Close() // Close connections on error
		conn2.Close()
		close(session.gameOverChan) // Signal immediate game over
		return session.gameOverChan
	}

	p1.Troops = getRandomTroops(troops, 3)
	p2.Troops = getRandomTroops(troops, 3)

	fmt.Printf("DEBUG: %s got %d troops\n", p1.Username, len(p1.Troops))
	fmt.Printf("DEBUG: %s got %d troops\n", p2.Username, len(p2.Troops))

	session.Broadcast("🔥 Match found! " + p1.Username + " vs " + p2.Username)
	session.Broadcast("� " + p1.Username + " will go first!")

	if session.IsTimedGame {
		session.GameTimer = NewGameTimer() // Initialize the game timer only if timed
		session.GameTimer.Start()          // Start the game timer

		// Goroutine to check for game end by time
		go func() {
			ticker := time.NewTicker(1 * time.Second) // Check every second for more precise time-out
			defer ticker.Stop()
			for range ticker.C {
				if session.GameOver {
					return // Game already over, stop checking
				}

				if session.GameTimer.IsTimeUp() {
					session.Mutex.Lock()
					if !session.GameOver { // Double check to avoid race condition
						session.GameOver = true
						session.endGameByTime() // Determine winner by destroyed towers
						session.Conn1.Close()   // Close connections when game ends by time
						session.Conn2.Close()
						close(session.gameOverChan) // Signal game over
					}
					session.Mutex.Unlock()
					return
				}
			}
		}()
	} else {
		session.Broadcast("This is an untimed game.")
	}

	// Main game loop in a goroutine so StartGameSession can return
	go func() {
		defer func() {
			// Ensure gameOverChan is closed if the game loop exits for any reason
			// other than explicit closure (e.g., King Tower destroyed or disconnection).
			// Also ensure connections are closed if the game ends unexpectedly.
			if !session.GameOver {
				session.Conn1.Close()
				session.Conn2.Close()
				close(session.gameOverChan)
			}
		}()

		for !session.GameOver {
			session.TakeTurn()
		}
	}()

	return session.gameOverChan // Return the channel immediately
}

func (gs *GameSession) TakeTurn() {
	// Check if game is already over by time before taking turn (only for timed games)
	if gs.IsTimedGame && gs.GameTimer.IsTimeUp() && !gs.GameOver {
		gs.Mutex.Lock()
		if !gs.GameOver { // Double check
			gs.GameOver = true
			gs.endGameByTime()
			gs.Conn1.Close() // Close connections
			gs.Conn2.Close()
			close(gs.gameOverChan) // Signal game over
		}
		gs.Mutex.Unlock()
		return
	}

	var conn net.Conn
	var active, opponent *models.Player
	if gs.TurnOwner == gs.Player1 {
		conn = gs.Conn1
		active = gs.Player1
		opponent = gs.Player2
	} else {
		conn = gs.Conn2
		active = gs.Player2
		opponent = gs.Player1
	}

	fmt.Printf("DEBUG: %s has %d troops at start of turn\n", active.Username, len(active.Troops))

	// Include remaining time in the menu message only if it's a timed game
	menu := fmt.Sprintf("🎯 Your turn, %s", active.Username)
	if gs.IsTimedGame {
		menu += fmt.Sprintf(" (Time Left: %s)", gs.GameTimer.FormattedTimeRemaining())
	}
	menu += "\n1. Attack Tower\n2. Show Status"
	network.SendPDU(conn, "menu", menu)

	pdu, err := network.ReadPDU(conn)
	if err != nil {
		// Handle connection error, set GameOver and signal
		gs.Mutex.Lock()
		if !gs.GameOver {
			gs.GameOver = true
			gs.Broadcast(fmt.Sprintf("🚫 %s disconnected. %s wins!", active.Username, opponent.Username))
			gs.Conn1.Close() // Close connections
			gs.Conn2.Close()
			close(gs.gameOverChan) // Signal game over
		}
		gs.Mutex.Unlock()
		return
	}
	choice := strings.TrimSpace(pdu.Payload)

	switch choice {
	case "1":
		gs.HandleAttack(active, opponent, conn)
	case "2":
		showStatus(conn, active)
	default:
		network.SendPDU(conn, "error", "❗ Invalid choice.")
	}

	if active.Mana >= 10 && len(active.Troops) < 3 {
		troops, _ := utils.LoadTroopsFromFile("data/troop.json")
		newTroop := getRandomTroops(troops, 1)[0]
		active.Troops = append(active.Troops, newTroop)
		fmt.Printf("DEBUG: %s restored troop %s\n", active.Username, newTroop.Name)
		network.SendPDU(conn, "event", fmt.Sprintf("✨ %s is restored to your hand!", newTroop.Name))
	}

	// Only switch turn if game is not over
	if !gs.GameOver {
		if gs.TurnOwner == gs.Player1 {
			gs.TurnOwner = gs.Player2
		} else {
			gs.TurnOwner = gs.Player1
		}
	}
}

func (gs *GameSession) Broadcast(msg string) {
	network.SendPDU(gs.Conn1, "broadcast", msg)
	network.SendPDU(gs.Conn2, "broadcast", msg)
}

func (gs *GameSession) HandleAttack(attacker, defender *models.Player,
	conn net.Conn) {
	fmt.Printf("DEBUG: %s troop count before attack: %d\n",
		attacker.Username, len(attacker.Troops))
	if len(attacker.Troops) == 0 {
		network.SendPDU(conn, "error", "❌ You have no troops to attack with.")
		return
	}
	// Select troop
	troopList := "Choose a troop to attack with:\n"
	for i, t := range attacker.Troops {
		troopList += fmt.Sprintf("%d. %s (ATK: %d, DEF: %d, Mana: %d)\n", i+1, t.Name, t.ATK, t.DEF, t.Mana)
	}
	network.SendPDU(conn, "select", troopList)
	pdu, err := network.ReadPDU(conn)
	if err != nil {
		// Handle connection error, set GameOver and signal
		gs.Mutex.Lock()
		if !gs.GameOver {
			gs.GameOver = true
			gs.Broadcast(fmt.Sprintf("🚫 %s disconnected. %s wins!", attacker.Username, defender.Username))
			gs.Conn1.Close() // Close connections
			gs.Conn2.Close()
			close(gs.gameOverChan) // Signal game over
		}
		gs.Mutex.Unlock()
		return
	}
	troopIndex := parseIndex(pdu.Payload) - 1
	if troopIndex < 0 || troopIndex >= len(attacker.Troops) {
		network.SendPDU(conn, "error", "❌ Invalid troop selection.")
		return
	}
	troop := attacker.Troops[troopIndex]
	if attacker.Mana < troop.Mana {
		network.SendPDU(conn, "error", fmt.Sprintf("❌ Not enough mana. You have %d, need %d. Turn skipped.", attacker.Mana, troop.Mana))
		return
	}

	// Determine if King Tower is allowed to be attacked
	guardsDown := true
	for _, t := range defender.Towers {
		if t.Type == "Guard Tower" && t.HP > 0 {
			guardsDown = false
			break
		}
	}
	targetList := "Choose tower to attack:\n"
	validIndices := []int{}
	for i, t := range defender.Towers {
		if t.HP <= 0 {
			continue
		}
		if t.Type == "King Tower" && !guardsDown {
			continue
		}
		targetList += fmt.Sprintf("%d. %s (HP: %d)\n", i+1, t.Type, t.HP)
		validIndices = append(validIndices, i)
	}
	if len(validIndices) == 0 {
		network.SendPDU(conn, "error", "❌ No valid towers to attack.")
		return
	}
	network.SendPDU(conn, "select", targetList)
	pdu, err = network.ReadPDU(conn)
	if err != nil {
		// Handle connection error, set GameOver and signal
		gs.Mutex.Lock()
		if !gs.GameOver {
			gs.GameOver = true
			gs.Broadcast(fmt.Sprintf("🚫 %s disconnected. %s wins!", attacker.Username, defender.Username))
			gs.Conn1.Close() // Close connections
			gs.Conn2.Close()
			close(gs.gameOverChan) // Signal game over
		}
		gs.Mutex.Unlock()
		return
	}
	targetIndex := parseIndex(pdu.Payload) - 1
	if targetIndex < 0 || targetIndex >= len(defender.Towers) {
		network.SendPDU(conn, "error", "❌ Invalid tower selection.")
		return
	}
	tower := &defender.Towers[targetIndex]
	damage := utils.CalculateDamage(troop.ATK, tower.DEF, tower.CRIT)
	tower.HP -= damage

	attacker.Mana -= troop.Mana
	attacker.Troops = append(attacker.Troops[:troopIndex], attacker.Troops[troopIndex+1:]...)

	network.SendPDU(conn, "result", fmt.Sprintf("💥 %s dealt %d damage to %s", troop.Name, damage, tower.Type))

	if tower.HP <= 0 {
		network.SendPDU(conn, "event", fmt.Sprintf("🏰 %s destroyed!", tower.Type))
		if tower.Type == "King Tower" {
			gs.Mutex.Lock() // Lock before modifying GameOver
			if !gs.GameOver {
				gs.GameOver = true
				gs.Broadcast(fmt.Sprintf("🎉 %s wins by destroying the King Tower!", attacker.Username))
				AddExp(attacker, 30)
				AddExp(defender, 10)
				gs.Conn1.Close() // Close connections
				gs.Conn2.Close()
				close(gs.gameOverChan) // Signal game over
			}
			gs.Mutex.Unlock()
		}
	}
}

// endGameByTime determines the winner when the game timer runs out.
// Winner is the player who destroyed more opponent towers.
func (gs *GameSession) endGameByTime() {
	// This function should only be called if IsTimedGame is true
	if !gs.IsTimedGame {
		fmt.Println("WARNING: endGameByTime called for untimed game.")
		return
	}

	// Count destroyed towers for each player
	p1DestroyedTowers := countDestroyedTowers(gs.Player2) // Player1 destroyed Player2's towers
	p2DestroyedTowers := countDestroyedTowers(gs.Player1) // Player2 destroyed Player1's towers

	gs.Broadcast("⏰ Time is up! Calculating results...")

	if p1DestroyedTowers > p2DestroyedTowers {
		gs.Broadcast(fmt.Sprintf("🎉 %s wins by destroying more towers (%d vs %d)!", gs.Player1.Username, p1DestroyedTowers, p2DestroyedTowers))
		AddExp(gs.Player1, 20)
		AddExp(gs.Player2, 5)
	} else if p2DestroyedTowers > p1DestroyedTowers {
		gs.Broadcast(fmt.Sprintf("🎉 %s wins by destroying more towers (%d vs %d)!", gs.Player2.Username, p2DestroyedTowers, p1DestroyedTowers))
		AddExp(gs.Player2, 20)
		AddExp(gs.Player1, 5)
	} else {
		gs.Broadcast("🤝 It's a draw! Both players destroyed the same number of towers.")
		AddExp(gs.Player1, 10)
		AddExp(gs.Player2, 10)
	}
	// Connections are NOT closed here. They are closed by the GameSession's goroutine or explicit calls.
}

// countDestroyedTowers counts the number of towers with HP <= 0 for a given player.
func countDestroyedTowers(player *models.Player) int {
	count := 0
	for _, t := range player.Towers {
		if t.HP <= 0 { // Tower is considered destroyed if HP is 0 or less
			count++
		}
	}
	return count
}

func showStatus(conn net.Conn, player *models.Player) {
	status := map[string]interface{}{
		"username": player.Username,
		"level":    player.Level,
		"exp":      player.EXP,
		"mana":     player.Mana,
		"towers":   player.Towers,
		"troops":   player.Troops,
	}
	jsonData, _ := json.MarshalIndent(status, "", " ")
	network.SendPDU(conn, "status", string(jsonData))
}

func parseIndex(input string) int {
	var idx int
	fmt.Sscanf(input, "%d", &idx)
	return idx
}

func getRandomTroops(all []models.Troop, count int) []models.Troop {
	rand.Seed(time.Now().UnixNano())
	var selected []models.Troop
	used := make(map[int]bool)
	for len(selected) < count && len(used) < len(all) {
		i := rand.Intn(len(all))
		if !used[i] {
			selected = append(selected, all[i])
			used[i] = true
		}
	}
	return selected
}

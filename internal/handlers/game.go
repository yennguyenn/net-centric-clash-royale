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

const (
	QueenHealAmount = 200
	QueenMaxHealHP  = 1000
	MaxCritsPerGame = 5
)

type GameSession struct {
	Player1      *models.Player
	Player2      *models.Player
	Conn1        net.Conn
	Conn2        net.Conn
	GameOver     bool
	TurnOwner    *models.Player
	Mutex        *sync.Mutex
	GameTimer    *GameTimer
	IsTimedGame  bool
	gameOverChan chan bool
}

// StartGameSession initializes a game between two players
func StartGameSession(p1, p2 *models.Player, conn1, conn2 net.Conn, isTimedGame bool) chan bool {

	session := &GameSession{
		Player1:      p1,
		Player2:      p2,
		Conn1:        conn1,
		Conn2:        conn2,
		GameOver:     false,
		TurnOwner:    p1,
		Mutex:        &sync.Mutex{},
		IsTimedGame:  isTimedGame,
		gameOverChan: make(chan bool),
	}

	troops, err := utils.LoadTroopsFromFile("data/troop.json")
	if err != nil || len(troops) < 3 {
		errMsg := "❌ Server error: cannot load or insufficient troop data."
		network.SendPDU(conn1, "error", errMsg)
		network.SendPDU(conn2, "error", errMsg)
		conn1.Close()
		conn2.Close()
		close(session.gameOverChan)
		return session.gameOverChan
	}

	p1.Troops = getRandomTroops(troops, 3)
	p2.Troops = getRandomTroops(troops, 3)
	p1.Towers, _ = utils.LoadPlayerTowers()
	p2.Towers, _ = utils.LoadPlayerTowers()
	p1.CritsLeft = MaxCritsPerGame
	p2.CritsLeft = MaxCritsPerGame

	session.Broadcast("🔥 Match found! " + p1.Username + " vs " + p2.Username)
	session.Broadcast("🎯 " + p1.Username + " will go first!")
	if session.IsTimedGame {
		session.GameTimer = NewGameTimer()
		session.GameTimer.Start()
		go session.watchTimer()
	} else {
		session.Broadcast("This is an untimed game.")
	}

	go func() {
		for !session.GameOver {
			session.TakeTurn()
		}
		// Wait briefly to ensure all PDUs are sent
		time.Sleep(500 * time.Millisecond)
		session.askRematch()
	}()
	return session.gameOverChan
}

func (gs *GameSession) watchTimer() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if gs.GameOver {
			return
		}
		if gs.GameTimer.IsTimeUp() {
			gs.Mutex.Lock()
			if !gs.GameOver {
				gs.GameOver = true
				gs.endGameByTime()
				gs.signalGameOver()
			}
			gs.Mutex.Unlock()
			return
		}
	}
}

func (gs *GameSession) signalGameOver() {
	if !gs.GameOver {
		gs.GameOver = true
	}

	select {
	case <-gs.gameOverChan:
	default:
		close(gs.gameOverChan)
	}
}

func (gs *GameSession) askRematch() {
	ask := func(conn net.Conn) bool {
		network.SendPDU(conn, "menu", "🔁 Do you want to play again?\n1. Yes\n2. No")
		pdu, err := network.ReadPDU(conn)
		if err != nil {
			return false
		}
		return strings.TrimSpace(pdu.Payload) == "1"
	}

	playAgain1 := ask(gs.Conn1)
	playAgain2 := ask(gs.Conn2)

	if playAgain1 && playAgain2 {
		network.SendPDU(gs.Conn1, "info", "🔄 Restarting game...")
		network.SendPDU(gs.Conn2, "info", "🔄 Restarting game...")

		mode1 := getPlayerMode(gs.Conn1)
		mode2 := getPlayerMode(gs.Conn2)
		mode := mode1 && mode2

		go StartGameSession(gs.Player1, gs.Player2, gs.Conn1, gs.Conn2, mode)
	} else {
		network.SendPDU(gs.Conn1, "info", "👋 Game over. Thank you for playing!")
		network.SendPDU(gs.Conn2, "info", "👋 Game over. Thank you for playing!")
		gs.Conn1.Close()
		gs.Conn2.Close()
	}
}

func getPlayerMode(conn net.Conn) bool {
	network.SendPDU(conn, "menu", "Choose game mode:\n1. Timed Game (3 minutes)\n2. Untimed Game\nEnter 1 or 2:")
	pdu, err := network.ReadPDU(conn)
	if err != nil {
		return false
	}
	return strings.TrimSpace(pdu.Payload) == "1"
}

func (gs *GameSession) TakeTurn() {
	if gs.IsTimedGame && gs.GameTimer.IsTimeUp() && !gs.GameOver {
		gs.Mutex.Lock()
		if !gs.GameOver {
			gs.GameOver = true
			gs.endGameByTime()
			gs.signalGameOver()
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

	menu := fmt.Sprintf("🎯 Your turn, %s", active.Username)
	if gs.IsTimedGame {
		menu += fmt.Sprintf(" (Time Left: %s)", gs.GameTimer.FormattedTimeRemaining())
	}
	menu += "\n1. Attack Tower\n2. Show Status"
	network.SendPDU(conn, "menu", menu)

	pdu, err := network.ReadPDU(conn)
	if err != nil {
		gs.signalGameOver()
		network.SendPDU(conn, "error", "⚠️ Connection lost.")
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
		network.SendPDU(conn, "event", fmt.Sprintf("✨ %s joins your hand!", newTroop.Name))
	}

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

func (gs *GameSession) HandleAttack(attacker, defender *models.Player, conn net.Conn) {
	if len(attacker.Troops) == 0 {
		network.SendPDU(conn, "error", "❌ You have no troops to attack with.")
		return
	}

	troopList := "Choose a troop to attack with:\n"
	for i, t := range attacker.Troops {
		troopList += fmt.Sprintf("%d. %s (ATK: %d, DEF: %d, Mana: %d)\n", i+1, t.Name, t.ATK, t.DEF, t.Mana)
	}
	network.SendPDU(conn, "select", troopList)
	pdu, _ := network.ReadPDU(conn)
	troopIndex := parseIndex(pdu.Payload) - 1
	if troopIndex < 0 || troopIndex >= len(attacker.Troops) {
		network.SendPDU(conn, "error", "❌ Invalid troop selection.")
		return
	}
	troop := attacker.Troops[troopIndex]
	if attacker.Mana < troop.Mana {
		network.SendPDU(conn, "error", "❌ Not enough mana.")
		return
	}

	// Troop queen
	if strings.ToLower(troop.Name) == "queen" {
		var lowest *models.Tower
		for i := range attacker.Towers {
			t := &attacker.Towers[i]
			if t.HP > 0 && (lowest == nil || t.HP < lowest.HP) {
				lowest = t
			}
		}
		if lowest != nil {
			oldHP := lowest.HP
			heal := QueenHealAmount
			if oldHP+heal > QueenMaxHealHP {
				heal = QueenMaxHealHP - oldHP
			}
			if heal > 0 {
				lowest.HP += heal
				network.SendPDU(conn, "result", fmt.Sprintf("💖 Queen healed your %s by %d HP (from %d ➡ %d)", lowest.Type, heal, oldHP, lowest.HP))
			} else {
				network.SendPDU(conn, "event", "⚠️ Tower already at full HP.")
			}
		} else {
			network.SendPDU(conn, "event", "⚠️ No towers to heal.")
		}
		return
	}

	// Crit 20%
	useCrit := false
	if attacker.CritsLeft > 0 {
		network.SendPDU(conn, "select", fmt.Sprintf("⚡ You have %d CRIT(s). Use one?\n1. Yes\n2. No", attacker.CritsLeft))
		pdu, _ := network.ReadPDU(conn)
		if strings.TrimSpace(pdu.Payload) == "1" {
			useCrit = true
			attacker.CritsLeft--
		}
	}

	guardsDown := true
	for _, t := range defender.Towers {
		if t.Type == "Guard Tower" && t.HP > 0 {
			guardsDown = false
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
	network.SendPDU(conn, "select", targetList)
	pdu, _ = network.ReadPDU(conn)
	targetIndex := parseIndex(pdu.Payload) - 1
	if targetIndex < 0 || targetIndex >= len(defender.Towers) {
		network.SendPDU(conn, "error", "❌ Invalid tower selection.")
		return
	}
	tower := &defender.Towers[targetIndex]
	fmt.Printf("DEBUG: %s attacking tower %s (DEF: %d)\n", attacker.Username, tower.Type, tower.DEF)
	damage := utils.CalculateDamage(troop.ATK, tower.DEF, useCrit)
	tower.HP -= damage
	// Gửi kết quả cho người chơi đang hành động
	network.SendPDU(conn, "result", fmt.Sprintf("💥 %s dealt %d damage to %s", troop.Name, damage, tower.Type))

	// Gửi cập nhật HP cho cả hai client
	// gs.Broadcast(fmt.Sprintf("📉 %s HP is now %d", tower.Type, tower.HP))
	attacker.Mana -= troop.Mana
	attacker.Troops = append(attacker.Troops[:troopIndex], attacker.Troops[troopIndex+1:]...)

	if tower.HP <= 0 {
		gs.Broadcast(fmt.Sprintf("🏰 %s destroyed!", tower.Type))
		if tower.Type == "King Tower" {
			gs.GameOver = true
			gs.Broadcast(fmt.Sprintf("🎉 %s wins by destroying the King Tower!", attacker.Username))
			AddExp(attacker, 30)
			AddExp(defender, 10)
			gs.signalGameOver()
		}
	}
}

func (gs *GameSession) endGameByTime() {
	p1Destroyed := countDestroyedTowers(gs.Player2)
	p2Destroyed := countDestroyedTowers(gs.Player1)

	gs.Broadcast("⏰ Time is up! Calculating results...")

	switch {
	case p1Destroyed > p2Destroyed:
		gs.Broadcast(fmt.Sprintf("🎉 %s wins (%d towers destroyed)!", gs.Player1.Username, p1Destroyed))
		AddExp(gs.Player1, 20)
		AddExp(gs.Player2, 5)
	case p2Destroyed > p1Destroyed:
		gs.Broadcast(fmt.Sprintf("🎉 %s wins (%d towers destroyed)!", gs.Player2.Username, p2Destroyed))
		AddExp(gs.Player2, 20)
		AddExp(gs.Player1, 5)
	default:
		gs.Broadcast("🤝 It's a draw!")
		AddExp(gs.Player1, 10)
		AddExp(gs.Player2, 10)
	}
}

func countDestroyedTowers(player *models.Player) int {
	count := 0
	for _, t := range player.Towers {
		if t.HP <= 0 {
			count++
		}
	}
	return count
}

func showStatus(conn net.Conn, player *models.Player) {
	status := map[string]interface{}{
		"username":  player.Username,
		"level":     player.Level,
		"exp":       player.EXP,
		"mana":      player.Mana,
		"towers":    player.Towers,
		"troops":    player.Troops,
		"critsLeft": player.CritsLeft,
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

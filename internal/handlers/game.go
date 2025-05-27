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
	Player1   *models.Player
	Player2   *models.Player
	Conn1     net.Conn
	Conn2     net.Conn
	GameOver  bool
	TurnOwner *models.Player
	Mutex     *sync.Mutex
}

func StartGameSession(p1, p2 *models.Player, conn1, conn2 net.Conn) {
	session := &GameSession{
		Player1:   p1,
		Player2:   p2,
		Conn1:     conn1,
		Conn2:     conn2,
		GameOver:  false,
		TurnOwner: p1,
		Mutex:     &sync.Mutex{},
	}

	troops, err := utils.LoadTroopsFromFile("data/troop.json")
	if err != nil {
		fmt.Printf("❌ Failed to load troops.json: %v\n", err)
		network.SendPDU(conn1, "error", "❌ Server error: cannot load troop data.")
		network.SendPDU(conn2, "error", "❌ Server error: cannot load troop data.")
		return
	}

	if len(troops) < 3 {
		fmt.Printf("⚠️ Not enough troops in data file. Only %d troops found\n", len(troops))
		network.SendPDU(conn1, "error", "⚠️ Not enough troop data on server.")
		network.SendPDU(conn2, "error", "⚠️ Not enough troop data on server.")
		return
	}

	// p1.Troops = append([]models.Troop{}, getRandomTroops(troops, 3)...)
	// p2.Troops = append([]models.Troop{}, getRandomTroops(troops, 3)...)
	p1.Troops = getRandomTroops(troops, 3)
	p2.Troops = getRandomTroops(troops, 3)

	fmt.Printf("DEBUG: %s got %d troops\n", p1.Username, len(p1.Troops))
	fmt.Printf("DEBUG: %s got %d troops\n", p2.Username, len(p2.Troops))

	session.Broadcast("🔥 Match found! " + p1.Username + " vs " + p2.Username)
	session.Broadcast("🎯 " + p1.Username + " will go first!")

	for !session.GameOver {
		session.TakeTurn()
	}
}

func (gs *GameSession) TakeTurn() {
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

	menu := "🎯 Your turn, " + active.Username + "\n1. Attack Tower\n2. Show Status"
	network.SendPDU(conn, "menu", menu)

	pdu, err := network.ReadPDU(conn)
	if err != nil {
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
		troops, _ := utils.LoadTroopsFromFile("data/troops.json")
		newTroop := getRandomTroops(troops, 1)[0]
		active.Troops = append(active.Troops, newTroop)
		fmt.Printf("DEBUG: %s restored troop %s\n", active.Username, newTroop.Name)
		network.SendPDU(conn, "event", fmt.Sprintf("✨ %s is restored to your hand!", newTroop.Name))
	}

	if gs.TurnOwner == gs.Player1 {
		gs.TurnOwner = gs.Player2
	} else {
		gs.TurnOwner = gs.Player1
	}
}

func (gs *GameSession) Broadcast(msg string) {
	network.SendPDU(gs.Conn1, "broadcast", msg)
	network.SendPDU(gs.Conn2, "broadcast", msg)
}

func (gs *GameSession) HandleAttack(attacker, defender *models.Player, conn net.Conn) {
	fmt.Printf("DEBUG: %s troop count before attack: %d\n", attacker.Username, len(attacker.Troops))
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
			gs.GameOver = true
			gs.Broadcast(fmt.Sprintf("🎉 %s wins by destroying the King Tower!", attacker.Username))
			AddExp(attacker, 30)
			AddExp(defender, 10)
		}
	}
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
	jsonData, _ := json.MarshalIndent(status, "", "  ")
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

// func LoadTroopsFromFile(path string) ([]models.Troop, error) {
// 	file, err := os.Open(path)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()
// 	var troops []models.Troop
// 	decoder := json.NewDecoder(file)
// 	if err := decoder.Decode(&troops); err != nil {
// 		return nil, err
// 	}

// 	return troops, nil
// }

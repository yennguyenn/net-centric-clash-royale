package handlers

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"

	"tcr_project/models"
	"tcr_project/network"
	"tcr_project/utils"
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

	menu := "🎯 Your turn, " + active.Username + "\n1. Deploy Troop\n2. Attack Tower\n3. Show Status"
	network.SendPDU(conn, "menu", menu)

	pdu, err := network.ReadPDU(conn)
	if err != nil {
		return
	}
	choice := strings.TrimSpace(pdu.Payload)

	switch choice {
	case "1":
		deployTroop(conn, active, gs.Mutex)
	case "2":
		gs.HandleAttack(active, opponent, conn)
	case "3":
		showStatus(conn, active)
	default:
		network.SendPDU(conn, "error", "❗ Invalid choice.")
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
	if len(attacker.Troops) == 0 {
		network.SendPDU(conn, "error", "❌ You have no troops.")
		return
	}

	// Select target tower
	towerList := "Choose tower to attack:\n"
	for i, t := range defender.Towers {
		towerList += fmt.Sprintf("%d. %s (HP: %d)\n", i+1, t.Type, t.HP)
	}
	network.SendPDU(conn, "select", towerList)

	pdu, err := network.ReadPDU(conn)
	if err != nil {
		return
	}
	targetIndex := parseIndex(pdu.Payload) - 1
	if targetIndex < 0 || targetIndex >= len(defender.Towers) {
		network.SendPDU(conn, "error", "❌ Invalid tower selection.")
		return
	}

	// Select troop
	troopList := "Choose your troop to attack with:\n"
	for i, t := range attacker.Troops {
		troopList += fmt.Sprintf("%d. %s (HP: %d, ATK: %d)\n", i+1, t.Name, t.HP, t.ATK)
	}
	network.SendPDU(conn, "select", troopList)

	pdu, err = network.ReadPDU(conn)
	if err != nil {
		return
	}
	troopIndex := parseIndex(pdu.Payload) - 1
	if troopIndex < 0 || troopIndex >= len(attacker.Troops) {
		network.SendPDU(conn, "error", "❌ Invalid troop.")
		return
	}

	troop := attacker.Troops[troopIndex]
	attacker.Troops = append(attacker.Troops[:troopIndex], attacker.Troops[troopIndex+1:]...)

	tower := &defender.Towers[targetIndex]
	damage := utils.CalculateDamage(troop.ATK, tower.DEF, tower.CRIT)
	tower.HP -= damage

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

func deployTroop(conn net.Conn, player *models.Player, mutex *sync.Mutex) {
	loadOnce.Do(func() {
		troops, err := utils.LoadTroopsFromFile("data/troops.json")
		if err != nil {
			network.SendPDU(conn, "error", "❌ Failed to load troop list.")
			return
		}
		troopList = troops
	})

	available := "Available Troops:\n"
	for _, t := range troopList {
		available += fmt.Sprintf("- %s (Mana: %d, ATK: %d, DEF: %d)\n", t.Name, t.Mana, t.ATK, t.DEF)
	}
	network.SendPDU(conn, "select", available+"\nEnter troop name to deploy:")

	pdu, err := network.ReadPDU(conn)
	if err != nil {
		return
	}
	troopName := strings.TrimSpace(pdu.Payload)
	troop := utils.GetTroopByName(troopList, troopName)
	if troop == nil {
		network.SendPDU(conn, "error", "❌ Troop not found.")
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	if player.Mana < troop.Mana {
		network.SendPDU(conn, "error", fmt.Sprintf("❌ Not enough mana. You have %d, need %d.", player.Mana, troop.Mana))
		return
	}

	if strings.ToLower(troop.Special) == "heal" {
		tower := getLowestHPTower(player.Towers)
		if tower != nil {
			tower.HP += 300
			network.SendPDU(conn, "event", fmt.Sprintf("❤️ Queen healed %s by 300 HP!", tower.Type))
		}
	}

	player.Troops = append(player.Troops, *troop)
	player.Mana -= troop.Mana

	network.SendPDU(conn, "result", fmt.Sprintf("✅ %s deployed! Remaining mana: %d", troop.Name, player.Mana))
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

func getLowestHPTower(towers []models.Tower) *models.Tower {
	if len(towers) == 0 {
		return nil
	}
	lowest := &towers[0]
	for i := range towers {
		if towers[i].HP < lowest.HP {
			lowest = &towers[i]
		}
	}
	return lowest
}

func parseIndex(input string) int {
	var idx int
	fmt.Sscanf(input, "%d", &idx)
	return idx
}

var (
	troopList []models.Troop
	loadOnce  sync.Once
)

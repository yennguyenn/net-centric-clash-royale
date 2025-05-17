package handlers

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"tcr_project/models"
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

// Hàm entry point để bắt đầu trận
func StartGameSession(p1, p2 *models.Player, c1, c2 net.Conn) {
	session := &GameSession{
		Player1:   p1,
		Player2:   p2,
		Conn1:     c1,
		Conn2:     c2,
		TurnOwner: p1,
		Mutex:     &sync.Mutex{},
	}

	session.Broadcast("🎮 Game started between " + p1.Username + " and " + p2.Username)

	startTime := time.Now()
	gameDuration := 3 * time.Minute

	for !session.GameOver {
		elapsed := time.Since(startTime)
		remaining := gameDuration - elapsed
		if remaining <= 0 {
			session.EndByTimeout()
			return
		}

		mins := int(remaining.Minutes())
		secs := int(remaining.Seconds()) % 60
		timerDisplay := fmt.Sprintf("⏱ Time left: %02d:%02d", mins, secs)
		session.Broadcast(timerDisplay)

		session.TakeTurn()
	}
}

func (gs *GameSession) Broadcast(msg string) {
	fmt.Fprintln(gs.Conn1, msg)
	fmt.Fprintln(gs.Conn2, msg)
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

	fmt.Fprintf(conn, "\n🎯 Your turn, %s\n", active.Username)
	fmt.Fprintln(conn, "1. Deploy Troop")
	fmt.Fprintln(conn, "2. Attack Tower")
	fmt.Fprintln(conn, "3. Show Status")
	fmt.Fprint(conn, "Choose: ")

	reader := NewReader(conn)
	choice := strings.TrimSpace(reader.ReadLine())

	switch choice {
	case "1":
		deployTroop(conn, active, gs.Mutex)
	case "2":
		gs.HandleAttack(active, opponent, conn)
	case "3":
		showStatus(conn, active)
	default:
		fmt.Fprintln(conn, "❗ Invalid choice.")
	}

	// Chuyển lượt
	if gs.TurnOwner == gs.Player1 {
		gs.TurnOwner = gs.Player2
	} else {
		gs.TurnOwner = gs.Player1
	}
}

func (gs *GameSession) HandleAttack(attacker, defender *models.Player, conn net.Conn) {
	if len(attacker.Troops) == 0 {
		fmt.Fprintln(conn, "❌ You have no troops.")
		return
	}

	// Chọn mục tiêu
	fmt.Fprintln(conn, "Choose tower to attack:")
	for i, t := range defender.Towers {
		fmt.Fprintf(conn, "%d. %s (HP: %d)\n", i+1, t.Type, t.HP)
	}
	fmt.Fprint(conn, "Target #: ")
	var targetIndex int
	fmt.Fscanln(conn, &targetIndex)
	targetIndex--

	if targetIndex < 0 || targetIndex >= len(defender.Towers) {
		fmt.Fprintln(conn, "❌ Invalid selection.")
		return
	}

	// Danh sách troop để chọn
	fmt.Fprintln(conn, "Choose your troop to attack with:")
	for i, t := range attacker.Troops {
		fmt.Fprintf(conn, "%d. %s (HP: %d, ATK: %d)\n", i+1, t.Name, t.HP, t.ATK)
	}
	fmt.Fprint(conn, "Troop #: ")
	var troopIndex int
	fmt.Fscanln(conn, &troopIndex)
	troopIndex--

	if troopIndex < 0 || troopIndex >= len(attacker.Troops) {
		fmt.Fprintln(conn, "❌ Invalid troop.")
		return
	}
	troop := attacker.Troops[troopIndex]

	tower := &defender.Towers[targetIndex]
	damage := utils.CalculateDamage(troop.ATK, tower.DEF, tower.CRIT)
	attacker.Troops = append(attacker.Troops[:troopIndex], attacker.Troops[troopIndex+1:]...)

	tower.HP -= damage
	fmt.Fprintf(conn, "💥 %s dealt %d damage to %s\n", troop.Name, damage, tower.Type)

	if tower.HP <= 0 {
		fmt.Fprintf(conn, "🏰 %s destroyed!\n", tower.Type)
		if tower.Type == "King Tower" {
			gs.GameOver = true
			gs.Broadcast(fmt.Sprintf("🎉 %s wins by destroying the King Tower!", attacker.Username))
			AddExp(attacker, 30)
			AddExp(defender, 10)
		}

	}

}

func (gs *GameSession) EndByTimeout() {
	gs.GameOver = true
	// So sánh số tower còn sống
	count1 := countAliveTowers(gs.Player1.Towers)
	count2 := countAliveTowers(gs.Player2.Towers)

	if count1 > count2 {
		gs.Broadcast(fmt.Sprintf("⏰ Time's up! %s wins!", gs.Player1.Username))
		AddExp(gs.Player1, 30)
		AddExp(gs.Player2, 10)
	} else if count2 > count1 {
		gs.Broadcast(fmt.Sprintf("⏰ Time's up! %s wins!", gs.Player2.Username))
		AddExp(gs.Player2, 30)
		AddExp(gs.Player1, 10)
	} else {
		gs.Broadcast("⏰ Time's up! It's a draw!")
		AddExp(gs.Player1, 10)
		AddExp(gs.Player2, 10)
	}

}

func countAliveTowers(towers []models.Tower) int {
	count := 0
	for _, t := range towers {
		if t.HP > 0 {
			count++
		}
	}
	return count
}

type ConnReader struct {
	conn net.Conn
}

func NewReader(conn net.Conn) *ConnReader {
	return &ConnReader{conn: conn}
}

func (r *ConnReader) ReadLine() string {
	buf := make([]byte, 1024)
	n, err := r.conn.Read(buf)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(buf[:n]))
}

var troopList []models.Troop
var loadOnce sync.Once

func deployTroop(conn net.Conn, player *models.Player, mutex *sync.Mutex) {
	reader := NewReader(conn)

	// Load troops from JSON file once
	loadOnce.Do(func() {
		troops, err := utils.LoadTroopsFromFile("data/troops.json")
		if err != nil {
			fmt.Fprintln(conn, "❌ Failed to load troop list.")
			return
		}
		troopList = troops
	})

	fmt.Fprint(conn, "Enter troop name to deploy: ")
	troopName := strings.TrimSpace(reader.ReadLine())

	troop := utils.GetTroopByName(troopList, troopName)
	if troop == nil {
		fmt.Fprintln(conn, "❌ Troop not found.")
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	if player.Mana < troop.Mana {
		fmt.Fprintf(conn, "❌ Not enough mana. You have %d, need %d.\n", player.Mana, troop.Mana)
		return
	}

	// Handle special effect: Queen heals tower
	if strings.ToLower(troop.Special) == "heal" {
		tower := getLowestHPTower(player.Towers)
		if tower != nil {
			tower.HP += 300
			fmt.Fprintf(conn, "❤️ Queen healed %s by 300 HP!\n", tower.Type)
		}
	}

	// Deploy troop
	player.Troops = append(player.Troops, *troop)
	player.Mana -= troop.Mana

	fmt.Fprintf(conn, "✅ %s deployed! Remaining mana: %d\n", troop.Name, player.Mana)
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
func showStatus(conn net.Conn, player *models.Player) {
	fmt.Fprintf(conn, "\n🎖 Player: %s\n", player.Username)
	fmt.Fprintf(conn, "Level: %d | EXP: %d | Mana: %d\n", player.Level, player.EXP, player.Mana)

	fmt.Fprintln(conn, "🏰 Towers:")
	if len(player.Towers) == 0 {
		fmt.Fprintln(conn, "(no towers)")
	} else {
		for _, t := range player.Towers {
			status := "✅"
			if t.HP <= 0 {
				status = "❌"
			}
			fmt.Fprintf(conn, "- %s: %d HP %s\n", t.Type, t.HP, status)
		}
	}

	fmt.Fprintln(conn, "🪖 Troops:")
	if len(player.Troops) == 0 {
		fmt.Fprintln(conn, "(none)")
	} else {
		for i, troop := range player.Troops {
			fmt.Fprintf(conn, " %d. %s (HP: %d, ATK: %d, DEF: %d)\n", i+1, troop.Name, troop.HP, troop.ATK, troop.DEF)
		}
	}
}

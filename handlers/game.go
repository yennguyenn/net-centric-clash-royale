package handlers

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"tcr_project/models"
)

func HandleGame(conn net.Conn, player *models.Player, mutex *sync.Mutex) {
	reader := NewReader(conn)

	for {
		fmt.Fprintln(conn, "\n🛡 Game Menu:")
		fmt.Fprintln(conn, "1. Deploy Troop")
		fmt.Fprintln(conn, "2. Show Status")
		fmt.Fprintln(conn, "3. Exit Game")
		fmt.Fprint(conn, "Choose an option: ")

		option := strings.TrimSpace(reader.ReadLine())

		switch option {
		case "1":
			deployTroop(conn, player, mutex)
		case "2":
			showStatus(conn, player)
		case "3":
			fmt.Fprintln(conn, "👋 Exiting game...")
			return
		default:
			fmt.Fprintln(conn, "❗ Invalid option.")
		}
	}
}

func deployTroop(conn net.Conn, player *models.Player, mutex *sync.Mutex) {
	reader := NewReader(conn)

	fmt.Fprint(conn, "Enter troop name to deploy: ")
	troopName := strings.TrimSpace(reader.ReadLine())

	fmt.Fprint(conn, "Enter mana cost: ")
	var cost int
	fmt.Fscanln(conn, &cost)

	mutex.Lock()
	defer mutex.Unlock()

	if player.Mana < cost {
		fmt.Fprintf(conn, "❌ Not enough mana. You have %d mana.\n", player.Mana)
		return
	}

	// Giả định Triển khai troop đơn giản
	troop := models.Troop{
		Name: troopName,
		HP:   100, // default HP
		ATK:  10,  // default attack
	}

	player.Troops = append(player.Troops, troop)
	player.Mana -= cost

	fmt.Fprintf(conn, "✅ Deployed %s! Remaining mana: %d\n", troopName, player.Mana)
}

func showStatus(conn net.Conn, player *models.Player) {
	fmt.Fprintf(conn, "\n🎖 Status for %s:\n", player.Username)
	fmt.Fprintf(conn, "Level: %d | EXP: %d | Mana: %d\n", player.Level, player.EXP, player.Mana)
	fmt.Fprintf(conn, "🪖 Troops deployed:\n")

	if len(player.Troops) == 0 {
		fmt.Fprintln(conn, "  (none)")
		return
	}

	for i, t := range player.Troops {
		fmt.Fprintf(conn, "  %d. %s (HP: %d, ATK: %d)\n", i+1, t.Name, t.HP, t.ATK)
	}
}

// Helper struct for reading from conn
type ConnReader struct {
	conn net.Conn
}

func NewReader(conn net.Conn) *ConnReader {
	return &ConnReader{conn: conn}
}

func (r *ConnReader) ReadLine() string {
	buf := make([]byte, 1024)
	n, _ := r.conn.Read(buf)
	return string(buf[:n])
}

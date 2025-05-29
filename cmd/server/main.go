package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"net-centric-clash-royale/internal/handlers"
	"net-centric-clash-royale/internal/models"
	"net-centric-clash-royale/internal/network"
)

// playerQueueEntry holds information for a player waiting in a matchmaking queue.
type playerQueueEntry struct {
	player   *models.Player
	conn     net.Conn
	waitChan chan bool
}

var (
	// Separate queues for timed and untimed games.
	timedQueue   map[string]playerQueueEntry
	untimedQueue map[string]playerQueueEntry
	queueMutex   sync.Mutex

	// Global mutex for player data and other shared resources.
	globalPlayerMutex sync.Mutex
)

func init() {
	timedQueue = make(map[string]playerQueueEntry)
	untimedQueue = make(map[string]playerQueueEntry)
}

func main() {
	fmt.Println("🚀 Starting TCR Server on port 9000...")

	players, err := handlers.LoadPlayers()
	if err != nil {
		log.Fatalf("❌ Failed to load players: %v", err)
	}

	network.StartTCPServer("9000", func(conn net.Conn) {
		handleConnectionWithPDU(conn, players)
	})
}

func handleConnectionWithPDU(conn net.Conn, playerMap map[string]*models.Player) {

	// Authenticate user (register/login)
	player := handlers.Authenticate(conn, &playerMap, &globalPlayerMutex)
	if player == nil {
		conn.Close()
		return
	}

	network.SendPDU(conn, "info", fmt.Sprintf("Welcome, %s!", player.Username))

	for {
		// --- Game Mode Selection Logic (re-integrated) ---
		var isTimedGame bool
		for {
			network.SendPDU(conn, "menu", "Choose game mode:\n1. Timed Game (3 minutes)\n2. Untimed Game (play following turn)\nEnter 1 or 2:")
			pdu, err := network.ReadPDU(conn)
			if err != nil {
				fmt.Println("❌ Failed to read PDU for game mode selection:", err)
				return
			}
			choice := strings.TrimSpace(pdu.Payload)

			switch choice {
			case "1":
				isTimedGame = true
				network.SendPDU(conn, "info", "You selected: Timed Game (3 minutes)")
				break
			case "2":
				isTimedGame = false
				network.SendPDU(conn, "info", "You selected: Untimed Game (play following turn)")
				break
			default:
				network.SendPDU(conn, "error", "❗ Invalid choice. Please enter 1 or 2.")
				continue
			}
			break
		}
		// --- End Game Mode Selection Logic ---

		var currentQueue map[string]playerQueueEntry

		if isTimedGame {
			currentQueue = timedQueue
		} else {
			currentQueue = untimedQueue
		}

		// --- Matchmaking Logic (re-integrated) ---
		queueMutex.Lock() // Lock for queue access

		var opponentEntry playerQueueEntry
		var foundOpponent bool

		// Check if there's a waiting player in the chosen queue
		if len(currentQueue) > 0 {
			for _, entry := range currentQueue {
				opponentEntry = entry
				foundOpponent = true
				break
			}
		}

		if !foundOpponent {
			waitChan := make(chan bool)
			entry := playerQueueEntry{
				player:   player,
				conn:     conn,
				waitChan: waitChan,
			}
			currentQueue[player.Username] = entry
			queueMutex.Unlock()

			network.SendPDU(conn, "info", "Waiting for an opponent to join. Please wait (timeout in 30s)...")

			select {
			case matchFound := <-waitChan:
				if !matchFound {
					network.SendPDU(conn, "info", "❌ Matchmaking timed out. No opponent found. Please choose game mode again.")
					continue
				}

				network.SendPDU(conn, "info", "Opponent found! Starting game...")
				_, err := network.ReadPDU(conn)
				if err != nil {
					fmt.Printf("DEBUG: Player %s's connection closed after game started: %v\n", player.Username, err)
				}
				return

			case <-time.After(30 * time.Second):
				// Timeout occurred
				queueMutex.Lock()
				if _, ok := currentQueue[player.Username]; ok {
					delete(currentQueue, player.Username)
				}
				queueMutex.Unlock()
				network.SendPDU(conn, "info", "❌ Matchmaking timed out. No opponent found. Please choose game mode again.")
				continue
			}

		} else {
			// Opponent found in this mode's queue, initiate game
			delete(currentQueue, opponentEntry.player.Username)
			queueMutex.Unlock()

			// Signal the waiting player that a match has been found
			if opponentEntry.waitChan != nil {
				opponentEntry.waitChan <- true
				close(opponentEntry.waitChan)
			}

			network.SendPDU(conn, "info", "Opponent found! Starting game...")

			// Start mana regeneration and game session in a new goroutine
			// The game session will manage the connections and close them when the game ends.
			go func(p1, p2 *models.Player, c1, c2 net.Conn, timedGame bool) {
				playersArr := []*models.Player{p1, p2}
				handlers.StartManaRegeneration(playersArr, &globalPlayerMutex)

				// Start the game session and get the game over channel
				gameOver := handlers.StartGameSession(p1, p2, c1, c2, timedGame)
				<-gameOver
				fmt.Printf("DEBUG: Game session between %s and %s ended.\n", p1.Username, p2.Username)
				// Connections are closed by GameSession when it signals completion.
			}(opponentEntry.player, player, opponentEntry.conn, conn, isTimedGame)

			_, err := network.ReadPDU(conn)
			if err != nil {
				fmt.Printf("DEBUG: Player %s's connection closed after game started: %v\n", player.Username, err)
			}
			return
		}
	}
}

// sendInfoPDU sends a basic info message to the client
func sendInfoPDU(conn net.Conn, message string) {
	pdu := network.PDU{Type: "info", Payload: message}
	data, err := network.EncodePDU(pdu)
	if err == nil {
		conn.Write(append(data, '\n'))
	}
}

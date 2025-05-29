package main

import (
	"fmt"
	"log"
	"net"
	"strings" // Re-added for game mode selection logic
	"sync"
	"time" // Re-added for matchmaking timeout logic

	"net-centric-clash-royale/internal/handlers" // Still needed for Authenticate, LoadPlayers, StartManaRegeneration, StartGameSession
	"net-centric-clash-royale/internal/models"
	"net-centric-clash-royale/internal/network"
)

// playerQueueEntry holds information for a player waiting in a matchmaking queue.
type playerQueueEntry struct {
	player   *models.Player
	conn     net.Conn
	waitChan chan bool // true for match found, false for timeout
}

var (
	// Separate queues for timed and untimed games.
	timedQueue   map[string]playerQueueEntry
	untimedQueue map[string]playerQueueEntry
	queueMutex   sync.Mutex // Protects access to both queues

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
	// IMPORTANT: Removed defer conn.Close() from here.
	// The GameSession (in internal/handlers/game.go) is now solely responsible
	// for closing the connections (conn1 and conn2) when the game ends
	// or a player disconnects during the game.
	// This goroutine will block until the game session completes and closes the connection.

	// Authenticate user (register/login)
	player := handlers.Authenticate(conn, &playerMap, &globalPlayerMutex)
	if player == nil {
		conn.Close() // Explicitly close connection if authentication fails
		return       // Authentication failed or user disconnected
	}

	network.SendPDU(conn, "info", fmt.Sprintf("Welcome, %s!", player.Username))

	for { // Loop to allow re-choosing game mode after timeout or failed match
		// --- Game Mode Selection Logic (re-integrated) ---
		var isTimedGame bool
		for { // Inner loop for game mode selection
			network.SendPDU(conn, "menu", "Choose game mode:\n1. Timed Game (3 minutes)\n2. Untimed Game (play following turn)\nEnter 1 or 2:")
			pdu, err := network.ReadPDU(conn)
			if err != nil {
				fmt.Println("❌ Failed to read PDU for game mode selection:", err)
				return // Client disconnected
			}
			choice := strings.TrimSpace(pdu.Payload)

			switch choice {
			case "1":
				isTimedGame = true
				network.SendPDU(conn, "info", "You selected: Timed Game (3 minutes)")
				break // Exit inner loop
			case "2":
				isTimedGame = false
				network.SendPDU(conn, "info", "You selected: Untimed Game (play following turn)")
				break // Exit inner loop
			default:
				network.SendPDU(conn, "error", "❗ Invalid choice. Please enter 1 or 2.")
				continue // Loop back to ask again
			}
			break // Exit inner loop after valid choice
		}
		// --- End Game Mode Selection Logic ---

		var currentQueue map[string]playerQueueEntry // Reference to the correct queue (timed or untimed)

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
			for _, entry := range currentQueue { // Get the single entry (since map only holds one)
				opponentEntry = entry
				foundOpponent = true
				break
			}
		}

		if !foundOpponent {
			// No one is waiting in this mode's queue, add current player to queue
			waitChan := make(chan bool) // Channel to signal match found (true) or timeout (false)
			entry := playerQueueEntry{
				player:   player,
				conn:     conn,
				waitChan: waitChan,
			}
			currentQueue[player.Username] = entry // Add to queue
			queueMutex.Unlock()                   // Release mutex after updating queue

			network.SendPDU(conn, "info", "Waiting for an opponent to join. Please wait (timeout in 30s)...")

			select {
			case matchFound := <-waitChan: // Block until signal from another player or timeout
				if !matchFound {
					// Timeout occurred, loop back to game mode selection
					network.SendPDU(conn, "info", "❌ Matchmaking timed out. No opponent found. Please choose game mode again.")
					continue // Loop back to the beginning of the `for` loop
				}
				// Match found, the game has been initiated by the other player's goroutine.
				network.SendPDU(conn, "info", "Opponent found! Starting game...")

				// This goroutine (the waiting player) now waits for the game to complete.
				// The game session will close the connections when it's done.
				// Reading from the connection will block until it's closed or data arrives.
				_, err := network.ReadPDU(conn)
				if err != nil {
					fmt.Printf("DEBUG: Player %s's connection closed after game started: %v\n", player.Username, err)
				}
				return // Game finished, or connection closed. This goroutine can now exit.

			case <-time.After(30 * time.Second):
				// Timeout occurred
				queueMutex.Lock()                               // Re-acquire mutex to safely remove player from queue
				if _, ok := currentQueue[player.Username]; ok { // Check if still in queue
					delete(currentQueue, player.Username)
				}
				queueMutex.Unlock()
				network.SendPDU(conn, "info", "❌ Matchmaking timed out. No opponent found. Please choose game mode again.")
				continue // Loop back to game mode selection
			}

		} else {
			// Opponent found in this mode's queue, initiate game
			delete(currentQueue, opponentEntry.player.Username) // Remove opponent from queue
			queueMutex.Unlock()                                 // Release mutex after updating queue

			// Signal the waiting player that a match has been found
			if opponentEntry.waitChan != nil {
				opponentEntry.waitChan <- true // Signal match found
				close(opponentEntry.waitChan)  // Close channel after signaling
			}

			network.SendPDU(conn, "info", "Opponent found! Starting game...")

			// Start mana regeneration and game session in a new goroutine
			// The game session will manage the connections and close them when the game ends.
			go func(p1, p2 *models.Player, c1, c2 net.Conn, timedGame bool) {
				playersArr := []*models.Player{p1, p2}
				handlers.StartManaRegeneration(playersArr, &globalPlayerMutex)

				// Start the game session and get the game over channel
				gameOver := handlers.StartGameSession(p1, p2, c1, c2, timedGame)
				<-gameOver // Block until the game session signals completion
				fmt.Printf("DEBUG: Game session between %s and %s ended.\n", p1.Username, p2.Username)
				// Connections are closed by GameSession when it signals completion.
			}(opponentEntry.player, player, opponentEntry.conn, conn, isTimedGame)

			// This goroutine (the one that found the match) also needs to wait for the game to end.
			// It will implicitly wait as its connection is being used by the game session.
			// When the game session closes the connection, network.ReadPDU will return an error,
			// causing this handleConnectionWithPDU to return.
			_, err := network.ReadPDU(conn) // Attempt to read, will block until conn is closed or data arrives
			if err != nil {
				fmt.Printf("DEBUG: Player %s's connection closed after game started: %v\n", player.Username, err)
			}
			return // Game finished, or connection closed. This goroutine can now exit.
		}
	}
}

// sendInfoPDU sends a basic info message to the client
func sendInfoPDU(conn net.Conn, message string) {
	pdu := network.PDU{Type: "info", Payload: message}
	data, err := network.EncodePDU(pdu)
	if err == nil {
		conn.Write(append(data, '\n')) // Ensure newline for scanner
	}
}

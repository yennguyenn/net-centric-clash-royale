package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	"tcr_project/internal/handlers"
	"tcr_project/internal/models"
	"tcr_project/internal/network"
)

var (
	waitingPlayer *models.Player
	waitingConn   net.Conn
	mutex         sync.Mutex
)

func main() {
	fmt.Println("🚀 Starting TCR Server on port 9000...")

	// Load player data from disk
	players, err := handlers.LoadPlayers()
	if err != nil {
		log.Fatalf("❌ Failed to load players: %v", err)
	}

	// Start TCP server and pass in handler function
	network.StartTCPServer("9000", func(conn net.Conn) {
		handleConnectionWithPDU(conn, players)
	})
}

func handleConnectionWithPDU(conn net.Conn, playerMap map[string]*models.Player) {
	defer conn.Close()

	// Authenticate user (register/login)
	player := handlers.Authenticate(conn, &playerMap, &mutex)
	if player == nil {
		return
	}

	// Send welcome message
	sendInfoPDU(conn, fmt.Sprintf("👋 Welcome, %s! Waiting for an opponent...", player.Username))

	mutex.Lock()
	if waitingPlayer == nil {
		// No one is waiting – put this player into waiting queue
		waitingPlayer = player
		waitingConn = conn
		mutex.Unlock()

		// Block until someone joins (keep connection open)
		waitChan := make(chan struct{})
		<-waitChan
	} else {
		// Match found
		player1 := waitingPlayer
		conn1 := waitingConn
		player2 := player
		conn2 := conn

		// Reset waiting queue
		waitingPlayer = nil
		waitingConn = nil
		mutex.Unlock()

		// Start mana regeneration and game session
		players := []*models.Player{player1, player2}
		handlers.StartManaRegeneration(players, &mutex)
		handlers.StartGameSession(player1, player2, conn1, conn2)
	}
}

// sendInfoPDU sends a basic info message to the client
func sendInfoPDU(conn net.Conn, message string) {
	pdu := network.PDU{Type: "info", Payload: message}
	data, err := network.EncodePDU(pdu)
	if err == nil {
		conn.Write(data)
		conn.Write([]byte("\n")) // Ensure newline for scanner
	}
}

package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	"tcr_project/handlers"
	"tcr_project/models"
	"tcr_project/network"
)

var (
	waitingPlayer *models.Player
	waitingConn   net.Conn
	mutex         sync.Mutex
)

func main() {
	fmt.Println("\U0001F680 Starting TCR Server...")

	// Load player data
	players, err := handlers.LoadPlayers()
	if err != nil {
		log.Fatalf("❌ Failed to load players: %v", err)
	}

	network.StartTCPServer("9000", func(conn net.Conn) {
		handleConnection(conn, players)
	})
}

func handleConnection(conn net.Conn, playerMap map[string]*models.Player) {
	defer conn.Close()

	player := handlers.Authenticate(conn, &playerMap, &mutex)
	if player == nil {
		return
	}

	fmt.Fprintf(conn, "✅ Welcome, %s! Waiting for an opponent...\n", player.Username)

	mutex.Lock()
	if waitingPlayer == nil {
		waitingPlayer = player
		waitingConn = conn
		mutex.Unlock()
		select {} // keep connection alive until opponent joins
	} else {
		player1 := waitingPlayer
		conn1 := waitingConn
		player2 := player
		conn2 := conn
		waitingPlayer = nil
		waitingConn = nil
		mutex.Unlock()

		players := []*models.Player{player1, player2}
		handlers.StartManaRegeneration(players, &mutex)
		handlers.StartGameSession(player1, player2, conn1, conn2)
	}
}

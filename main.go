package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	"tcr_project/handlers"
	"tcr_project/models"
)

var (
	waitingPlayer *models.Player
	waitingConn   net.Conn
	mutex         sync.Mutex
)

func main() {
	fmt.Println("🚀 Starting TCR Server...")

	// Load player data from file
	players, err := handlers.LoadPlayers()
	if err != nil {
		log.Fatalf("❌ Failed to load players: %v", err)
	}

	// Start listening on TCP port 9000
	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
	defer ln.Close()
	fmt.Println("✅ Server listening on port 9000...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("⚠️ Connection error:", err)
			continue
		}
		go handleConnection(conn, players)
	}
}

func handleConnection(conn net.Conn, playerMap map[string]*models.Player) {
	defer conn.Close()

	player := handlers.Authenticate(conn, &playerMap, &mutex)
	if player == nil {
		return
	}

	fmt.Fprintf(conn, "✅ Welcome, %s! Waiting for an opponent...\n", player.Username)

	// Ghép cặp 2 người chơi
	mutex.Lock()
	if waitingPlayer == nil {
		waitingPlayer = player
		waitingConn = conn
		mutex.Unlock()
		select {} // giữ kết nối mở, chờ đối thủ
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

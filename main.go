package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	"tcr_project/handlers"
)

func main() {
	fmt.Println("Starting TCR Server...")

	players, err := handlers.LoadPlayers()
	if err != nil {
		log.Fatalf("❌ Failed to load players: %v", err)
	}
	var mutex sync.Mutex

	// TCP server đơn giản
	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
	defer ln.Close()
	fmt.Println("✅ Server listening on port 9000...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go func(c net.Conn) {
			defer c.Close()
			player := handlers.Authenticate(c, &players, &mutex)
			if player != nil {
				fmt.Fprintf(c, "Welcome, %s! 🎮\n", player.Username)
			}
		}(conn)
	}
}

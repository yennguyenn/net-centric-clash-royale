package network

import (
	"fmt"
	"log"
	"net"
)

func StartTCPServer(port string, handleConn func(net.Conn)) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer ln.Close()
	fmt.Println("✅ TCP Server listening on port", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go handleConn(conn)
	}
}

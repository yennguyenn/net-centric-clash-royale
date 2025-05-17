package network

import (
	"fmt"
	"net"
)

// StartTCPServer starts a TCP server on the given port.
// For each accepted connection, it starts a new goroutine and calls the provided handler.
func StartTCPServer(port string, handleConn func(net.Conn)) {
	address := ":" + port
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Printf("❌ Failed to start TCP server on port %s: %v\n", port, err)
		return
	}
	defer listener.Close()

	fmt.Printf("✅ TCP server started on port %s\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("⚠️ Failed to accept connection: %v\n", err)
			continue
		}

		// Launch a new goroutine for each client
		go handleConn(conn)
	}
}

package network

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func StartTCPClient(address string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("❌ Error connecting:", err)
		return
	}
	defer conn.Close()

	// Start goroutine to receive and print messages from server
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println("📥", line)
		}
	}()

	// Read input and send as PDU
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("▶️ ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		pdu := PDU{
			Type:    "input",
			Payload: input,
		}

		data, err := EncodePDU(pdu)
		if err == nil {
			conn.Write(append(data, '\n'))
		}
	}
}

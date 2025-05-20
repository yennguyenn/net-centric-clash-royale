package network

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func StartTCPClient(address string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("❌ Error connecting:", err)
		return
	}
	defer conn.Close()

	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		fmt.Fprint(conn, input)
	}
}

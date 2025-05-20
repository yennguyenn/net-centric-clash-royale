package main

import "net-centric-clash-royale/internal/network"

func main() {
	network.StartTCPClient("localhost:9000")
}

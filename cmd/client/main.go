package main

import "tcr_project/internal/network"

func main() {
	network.StartTCPClient("localhost:9000")
}

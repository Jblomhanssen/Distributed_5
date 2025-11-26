package main

import (
	"flag"
	"log"
)

func main() {
	serverAddr := flag.String("server", "localhost:5001", "primary server address")
	flag.Parse()

	client, err := NewAuctionClient(*serverAddr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	log.Printf("Connected to auction server at %s", *serverAddr)

	runTestScenarios(client)
}

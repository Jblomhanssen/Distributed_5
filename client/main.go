package main

import (
	"flag"
	"log"
)

func main() {
	primaryAddr := flag.String("primary", "localhost:5001", "primary server address")
	backupAddr := flag.String("backup", "localhost:5002", "backup server address")
	flag.Parse()

	client, err := NewAuctionClient(*primaryAddr, *backupAddr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	log.Printf("Auction client ready (primary: %s, backup: %s)", *primaryAddr, *backupAddr)

	runTestScenarios(client)
}

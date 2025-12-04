package main

import (
	"flag"
	"fmt"
	"log"
	"time"
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

	placeBid(client, "Alice", 100)
	time.Sleep(500 * time.Millisecond)
	
	placeBid(client, "Bob", 150)
	time.Sleep(500 * time.Millisecond)
	
	placeBid(client, "Charlie", 200)
	time.Sleep(500 * time.Millisecond)

	fmt.Println("\nKill primary now (Ctrl+C in primary terminal), then press Enter")
	fmt.Scanln()

	placeBid(client, "David", 250)
	placeBid(client, "Eve", 300)
	
	getResult(client)
}

func placeBid(client *AuctionClient, bidder string, amount int32) {
	response, err := client.PlaceBid(bidder, amount)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("%s bid %d: %s\n", bidder, amount, response.Outcome)
}

func getResult(client *AuctionClient) {
	result, err := client.GetResult()
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("Winner: %s with %d\n", result.Winner, result.HighestBid)
}

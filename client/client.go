package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/joachimblom-hanssen/Distributed_5/proto"
	"google.golang.org/grpc"
)

type AuctionClient struct {
	client         pb.AuctionServiceClient
	conn           *grpc.ClientConn
	currentServer  string
	primaryAddr    string
	backupAddr     string
}

func NewAuctionClient(primaryAddress, backupAddress string) (*AuctionClient, error) {
	client := &AuctionClient{
		primaryAddr: primaryAddress,
		backupAddr:  backupAddress,
	}
	
	// Try to connect to primary first
	if err := client.connectToServer(primaryAddress); err != nil {
		log.Printf("Failed to connect to primary, trying backup: %v", err)
		// If primary fails, try backup
		if err := client.connectToServer(backupAddress); err != nil {
			return nil, fmt.Errorf("failed to connect to both primary and backup: %v", err)
		}
	}
	
	return client, nil
}

func (c *AuctionClient) connectToServer(address string) error {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(3*time.Second))
	if err != nil {
		return err
	}
	
	// Close old connection if exists
	if c.conn != nil {
		c.conn.Close()
	}
	
	c.conn = conn
	c.client = pb.NewAuctionServiceClient(conn)
	c.currentServer = address
	log.Printf("Connected to server at %s", address)
	
	return nil
}

func (c *AuctionClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *AuctionClient) PlaceBid(clientID string, amount int32) (*pb.BidResponse, error) {
	requestID := fmt.Sprintf("%s-%d", clientID, time.Now().UnixNano())
	
	request := &pb.BidRequest{
		Amount:    amount,
		ClientId:  clientID,
		RequestId: requestID,
	}
	
	// Try with retry logic
	return c.executeWithFailover(func(client pb.AuctionServiceClient) (*pb.BidResponse, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return client.Bid(ctx, request)
	})
}

func (c *AuctionClient) GetResult() (*pb.ResultResponse, error) {
	return c.executeResultWithFailover(func(client pb.AuctionServiceClient) (*pb.ResultResponse, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return client.Result(ctx, &pb.ResultRequest{})
	})
}

// executeWithFailover tries the operation and falls back to backup if primary fails
func (c *AuctionClient) executeWithFailover(operation func(pb.AuctionServiceClient) (*pb.BidResponse, error)) (*pb.BidResponse, error) {
	response, err := operation(c.client)
	
	if err != nil {
		log.Printf("Request failed on %s: %v", c.currentServer, err)
		
		// Determine which server to try next
		nextServer := c.backupAddr
		if c.currentServer == c.backupAddr {
			nextServer = c.primaryAddr
		}
		
		log.Printf("Attempting failover to %s", nextServer)
		
		// Try to reconnect to other server
		if err := c.connectToServer(nextServer); err != nil {
			return nil, fmt.Errorf("failover failed: %v", err)
		}
		
		// Retry operation on new server
		response, err = operation(c.client)
		if err != nil {
			return nil, fmt.Errorf("operation failed on failover server: %v", err)
		}
		
		log.Println("Failover successful")
	}
	
	return response, nil
}

func (c *AuctionClient) executeResultWithFailover(operation func(pb.AuctionServiceClient) (*pb.ResultResponse, error)) (*pb.ResultResponse, error) {
	response, err := operation(c.client)
	
	if err != nil {
		log.Printf("Request failed on %s: %v", c.currentServer, err)
		
		// Determine which server to try next
		nextServer := c.backupAddr
		if c.currentServer == c.backupAddr {
			nextServer = c.primaryAddr
		}
		
		log.Printf("Attempting failover to %s", nextServer)
		
		// Try to reconnect to other server
		if err := c.connectToServer(nextServer); err != nil {
			return nil, fmt.Errorf("failover failed: %v", err)
		}
		
		// Retry operation on new server
		response, err = operation(c.client)
		if err != nil {
			return nil, fmt.Errorf("operation failed on failover server: %v", err)
		}
		
		log.Println("Failover successful")
	}
	
	return response, nil
}

func runTestScenarios(client *AuctionClient) {
	fmt.Println("\n=== Auction System Test ===\n")

	fmt.Println("--- Scenario 1: Normal Bidding ---")
	placeBidAndLog(client, "Alice", 100)
	time.Sleep(500 * time.Millisecond)
	
	placeBidAndLog(client, "Bob", 150)
	time.Sleep(500 * time.Millisecond)
	
	placeBidAndLog(client, "Charlie", 200)
	time.Sleep(500 * time.Millisecond)

	getResultAndLog(client)

	fmt.Println("\n--- Scenario 2: Invalid Bids ---")
	placeBidAndLog(client, "David", 150)
	placeBidAndLog(client, "Eve", -50)
	placeBidAndLog(client, "Frank", 0)

	fmt.Println("\n--- Scenario 3: Same Bidder Multiple Bids ---")
	placeBidAndLog(client, "Alice", 250)
	time.Sleep(500 * time.Millisecond)
	
	placeBidAndLog(client, "Alice", 240)
	placeBidAndLog(client, "Alice", 300)

	fmt.Println("\n--- Final Result ---")
	getResultAndLog(client)

	fmt.Println("\n=== Testing Primary Crash Resilience ===")
	fmt.Println("Now you can crash the primary (Ctrl+C in primary terminal)")
	fmt.Println("The client will automatically failover to backup")
	time.Sleep(2 * time.Second)
	
	fmt.Println("\n--- After Primary Crash ---")
	placeBidAndLog(client, "Grace", 350)
	placeBidAndLog(client, "Henry", 400)
	
	getResultAndLog(client)
}

func placeBidAndLog(client *AuctionClient, bidder string, amount int32) {
	response, err := client.PlaceBid(bidder, amount)
	if err != nil {
		log.Printf("Error placing bid: %v", err)
		return
	}

	outcomeStr := outcomeToString(response.Outcome)
	fmt.Printf("%s bid %d: %s - %s\n", bidder, amount, outcomeStr, response.Message)
}

func getResultAndLog(client *AuctionClient) {
	result, err := client.GetResult()
	if err != nil {
		log.Printf("Error getting result: %v", err)
		return
	}

	statusStr := "ONGOING"
	if result.Status == pb.AuctionStatus_CLOSED {
		statusStr = "CLOSED"
	}

	fmt.Printf("Auction Status: %s\n", statusStr)
	fmt.Printf("Highest Bid: %d\n", result.HighestBid)
	if result.Winner != "" {
		fmt.Printf("Current Leader: %s\n", result.Winner)
	}
}

func outcomeToString(outcome pb.Outcome) string {
	switch outcome {
	case pb.Outcome_SUCCESS:
		return "SUCCESS"
	case pb.Outcome_FAIL:
		return "FAIL"
	case pb.Outcome_EXCEPTION:
		return "EXCEPTION"
	default:
		return "UNKNOWN"
	}
}

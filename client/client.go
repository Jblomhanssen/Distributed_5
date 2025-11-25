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
	client pb.AuctionServiceClient
	conn   *grpc.ClientConn
}

func NewAuctionClient(serverAddress string) (*AuctionClient, error) {
	conn, err := grpc.Dial(serverAddress, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	return &AuctionClient{
		client: pb.NewAuctionServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *AuctionClient) Close() {
	c.conn.Close()
}

func (c *AuctionClient) PlaceBid(clientID string, amount int32) (*pb.BidResponse, error) {
	requestID := fmt.Sprintf("%s-%d", clientID, time.Now().UnixNano())
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request := &pb.BidRequest{
		Amount:    amount,
		ClientId:  clientID,
		RequestId: requestID,
	}

	response, err := c.client.Bid(ctx, request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *AuctionClient) GetResult() (*pb.ResultResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := c.client.Result(ctx, &pb.ResultRequest{})
	if err != nil {
		return nil, err
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

	fmt.Println("\n--- Waiting for auction to close (100 seconds) ---")
	fmt.Println("(In real test, you would wait or reduce auction duration)")
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

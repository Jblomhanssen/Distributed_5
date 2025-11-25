package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/joachimblom-hanssen/Distributed_5/auction"
	pb "github.com/joachimblom-hanssen/Distributed_5/proto"
	"google.golang.org/grpc"
)

type PrimaryServer struct {
	pb.UnimplementedAuctionServiceServer
	auctionState      *auction.Auction
	processedRequests map[string]*pb.BidResponse
	backupClient      pb.ReplicationServiceClient
	mutex             sync.Mutex
}

func NewPrimaryServer(backupAddress string) (*PrimaryServer, error) {
	conn, err := grpc.Dial(backupAddress, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to backup: %v", err)
	}

	return &PrimaryServer{
		auctionState:      auction.NewAuction(time.Now()),
		processedRequests: make(map[string]*pb.BidResponse),
		backupClient:      pb.NewReplicationServiceClient(conn),
	}, nil
}

func (s *PrimaryServer) Bid(ctx context.Context, req *pb.BidRequest) (*pb.BidResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Stage 2: Coordination - check for duplicate request
	if cachedResponse, exists := s.processedRequests[req.RequestId]; exists {
		log.Printf("Duplicate request %s, returning cached response", req.RequestId)
		return cachedResponse, nil
	}

	// Stage 3: Execution
	outcome := s.auctionState.PlaceBid(req.ClientId, req.Amount, time.Now())
	
	response := &pb.BidResponse{
		Outcome: outcome,
		Message: outcomeMessage(outcome, req.Amount),
	}

	// Stage 4: Agreement - replicate to backup and wait for ACK
	update := &pb.UpdateRequest{
		RequestId: req.RequestId,
		Type:      pb.UpdateType_BID,
		Amount:    req.Amount,
		ClientId:  req.ClientId,
		Outcome:   outcome,
	}

	if err := s.replicateToBackup(ctx, update); err != nil {
		log.Printf("Failed to replicate to backup: %v", err)
		return &pb.BidResponse{
			Outcome: pb.Outcome_EXCEPTION,
			Message: "replication failed",
		}, nil
	}

	s.processedRequests[req.RequestId] = response

	// Stage 5: Response
	return response, nil
}

func (s *PrimaryServer) Result(ctx context.Context, req *pb.ResultRequest) (*pb.ResultResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	status, highestBid, winner := s.auctionState.GetResult(time.Now())

	return &pb.ResultResponse{
		Status:     status,
		HighestBid: highestBid,
		Winner:     winner,
	}, nil
}

func (s *PrimaryServer) replicateToBackup(ctx context.Context, update *pb.UpdateRequest) error {
	ackCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	ack, err := s.backupClient.ReplicateUpdate(ackCtx, update)
	if err != nil {
		return err
	}

	if !ack.Acknowledged {
		return fmt.Errorf("backup did not acknowledge update")
	}

	return nil
}

func outcomeMessage(outcome pb.Outcome, amount int32) string {
	switch outcome {
	case pb.Outcome_SUCCESS:
		return fmt.Sprintf("bid of %d accepted", amount)
	case pb.Outcome_FAIL:
		return "bid rejected - too low or auction closed"
	case pb.Outcome_EXCEPTION:
		return "invalid bid amount"
	default:
		return "unknown outcome"
	}
}

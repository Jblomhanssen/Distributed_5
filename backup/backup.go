package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/joachimblom-hanssen/Distributed_5/auction"
	pb "github.com/joachimblom-hanssen/Distributed_5/proto"
)

type BackupServer struct {
	pb.UnimplementedReplicationServiceServer
	pb.UnimplementedAuctionServiceServer
	auctionState      *auction.Auction
	processedRequests map[string]*pb.BidResponse
	mutex             sync.Mutex
	
	// Failure detection and promotion
	isPrimary         bool
	lastHeartbeat     time.Time
	heartbeatMutex    sync.Mutex
}

func NewBackupServer(startTime time.Time) *BackupServer {
	s := &BackupServer{
		auctionState:      auction.NewAuction(startTime),
		processedRequests: make(map[string]*pb.BidResponse),
		isPrimary:         false,
		lastHeartbeat:     time.Now(),
	}
	
	// Start monitoring for primary failure
	go s.monitorPrimaryHealth()
	
	return s
}

// monitorPrimaryHealth checks if heartbeats from primary have stopped
func (s *BackupServer) monitorPrimaryHealth() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		s.heartbeatMutex.Lock()
		timeSinceLastHeartbeat := time.Since(s.lastHeartbeat)
		s.heartbeatMutex.Unlock()
		
		// If no heartbeat/update for 5 seconds and we're not already primary, promote
		if timeSinceLastHeartbeat > 5*time.Second && !s.isPrimary {
			s.promoteToPrimary()
		}
	}
}

// promoteToPrimary handles the transition from backup to primary
func (s *BackupServer) promoteToPrimary() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.isPrimary {
		return // Already promoted
	}
	
	log.Println("=== PRIMARY FAILURE DETECTED ===")
	log.Println("Promoting backup to primary role")
	
	s.isPrimary = true
	
	log.Println("Backup is now serving as PRIMARY")
	log.Println("System continues operating with current auction state")
}

func (s *BackupServer) ReplicateUpdate(ctx context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Update heartbeat timestamp - receiving updates means primary is alive
	s.heartbeatMutex.Lock()
	s.lastHeartbeat = time.Now()
	s.heartbeatMutex.Unlock()

	// Check for duplicate
	if cachedResponse, exists := s.processedRequests[req.RequestId]; exists {
		log.Printf("Duplicate update %s, acknowledging with cached response", req.RequestId)
		return &pb.UpdateResponse{Acknowledged: true}, nil
	}

	// Apply the operation with the same outcome as primary decided
	// This ensures consistency - we don't re-execute, we just record
	s.auctionState.PlaceBid(req.ClientId, req.Amount, time.Now())
	
	// Store the response for idempotency
	response := &pb.BidResponse{
		Outcome: req.Outcome,
		Message: outcomeMessage(req.Outcome, req.Amount),
	}
	s.processedRequests[req.RequestId] = response

	log.Printf("Replicated bid from %s: %d, outcome: %s", req.ClientId, req.Amount, req.Outcome)

	return &pb.UpdateResponse{Acknowledged: true}, nil
}

// Heartbeat handles heartbeat messages from primary
func (s *BackupServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	s.heartbeatMutex.Lock()
	s.lastHeartbeat = time.Now()
	s.heartbeatMutex.Unlock()
	
	return &pb.HeartbeatResponse{Alive: true}, nil
}

func (s *BackupServer) Result(ctx context.Context, req *pb.ResultRequest) (*pb.ResultResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	status, highestBid, winner := s.auctionState.GetResult(time.Now())

	return &pb.ResultResponse{
		Status:     status,
		HighestBid: highestBid,
		Winner:     winner,
	}, nil
}

// Bid now works if we've been promoted to primary
func (s *BackupServer) Bid(ctx context.Context, req *pb.BidRequest) (*pb.BidResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// If we're not primary, reject
	if !s.isPrimary {
		return &pb.BidResponse{
			Outcome: pb.Outcome_EXCEPTION,
			Message: "bids must be directed to primary",
		}, nil
	}
	
	// We're now primary - handle bid directly
	// Stage 2: Coordination - check for duplicate request
	if cachedResponse, exists := s.processedRequests[req.RequestId]; exists {
		log.Printf("Duplicate request %s, returning cached response", req.RequestId)
		return cachedResponse, nil
	}

	// Stage 3: Execution (no replication since we're operating with f=0)
	outcome := s.auctionState.PlaceBid(req.ClientId, req.Amount, time.Now())
	
	response := &pb.BidResponse{
		Outcome: outcome,
		Message: outcomeMessage(outcome, req.Amount),
	}
	
	s.processedRequests[req.RequestId] = response
	
	log.Printf("Processing bid as PRIMARY: %s bid %d, outcome: %s", req.ClientId, req.Amount, outcome)

	return response, nil
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

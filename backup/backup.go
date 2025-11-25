package main

import (
	"context"
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
	processedRequests map[string]bool
	mutex             sync.Mutex
}

func NewBackupServer(startTime time.Time) *BackupServer {
	return &BackupServer{
		auctionState:      auction.NewAuction(startTime),
		processedRequests: make(map[string]bool),
	}
}

func (s *BackupServer) ReplicateUpdate(ctx context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.processedRequests[req.RequestId] {
		log.Printf("Duplicate update %s, acknowledging", req.RequestId)
		return &pb.UpdateResponse{Acknowledged: true}, nil
	}

	s.auctionState.PlaceBid(req.ClientId, req.Amount, time.Now())
	
	s.processedRequests[req.RequestId] = true

	log.Printf("Replicated bid from %s: %d, outcome: %s", req.ClientId, req.Amount, req.Outcome)

	return &pb.UpdateResponse{Acknowledged: true}, nil
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

func (s *BackupServer) Bid(ctx context.Context, req *pb.BidRequest) (*pb.BidResponse, error) {
	return &pb.BidResponse{
		Outcome: pb.Outcome_EXCEPTION,
		Message: "bids must be directed to primary",
	}, nil
}

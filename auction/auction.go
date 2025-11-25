package auction

import (
	"time"

	pb "github.com/joachimblom-hanssen/Distributed_5/proto"
)

const AuctionDuration = 100 * time.Second

type Auction struct {
	highestBid    int32
	highestBidder string
	bidders       map[string]int32
	startTime     time.Time
	closed        bool
}

func NewAuction(startTime time.Time) *Auction {
	return &Auction{
		highestBid:    0,
		highestBidder: "",
		bidders:       make(map[string]int32),
		startTime:     startTime,
		closed:        false,
	}
}

func (a *Auction) PlaceBid(clientID string, amount int32, currentTime time.Time) pb.Outcome {
	if a.IsClosed(currentTime) {
		return pb.Outcome_FAIL
	}

	if amount <= 0 {
		return pb.Outcome_EXCEPTION
	}

	if amount <= a.highestBid {
		return pb.Outcome_FAIL
	}

	previousBid, exists := a.bidders[clientID]
	if exists && amount <= previousBid {
		return pb.Outcome_FAIL
	}

	a.bidders[clientID] = amount
	a.highestBid = amount
	a.highestBidder = clientID

	return pb.Outcome_SUCCESS
}

func (a *Auction) GetResult(currentTime time.Time) (pb.AuctionStatus, int32, string) {
	if a.IsClosed(currentTime) {
		return pb.AuctionStatus_CLOSED, a.highestBid, a.highestBidder
	}
	return pb.AuctionStatus_ONGOING, a.highestBid, a.highestBidder
}

func (a *Auction) IsClosed(currentTime time.Time) bool {
	if a.closed {
		return true
	}

	if currentTime.Sub(a.startTime) >= AuctionDuration {
		a.closed = true
		return true
	}

	return false
}

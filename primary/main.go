package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	pb "github.com/joachimblom-hanssen/Distributed_5/proto"
	"google.golang.org/grpc"
)

func main() {
	port := flag.Int("port", 5000, "primary server port")
	backupAddr := flag.String("backup", "localhost:5001", "backup server address")
	flag.Parse()

	primaryServer, err := NewPrimaryServer(*backupAddr)
	if err != nil {
		log.Fatalf("Failed to create primary server: %v", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAuctionServiceServer(grpcServer, primaryServer)

	log.Printf("Primary server listening on port %d", *port)
	log.Printf("Connected to backup at %s", *backupAddr)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	pb "github.com/joachimblom-hanssen/Distributed_5/proto"
	"google.golang.org/grpc"
)

func main() {
	port := flag.Int("port", 5002, "backup server port")
	flag.Parse()

	backupServer := NewBackupServer(time.Now())

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterReplicationServiceServer(grpcServer, backupServer)
	pb.RegisterAuctionServiceServer(grpcServer, backupServer)

	log.Printf("Backup server listening on port %d", *port)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

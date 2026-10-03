package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	pb "github.com/ojasmishra2406/distributed-Value/api/proto"
	"github.com/ojasmishra2406/distributed-Value/internal/network"
	"github.com/ojasmishra2406/distributed-Value/internal/storage"
	"google.golang.org/grpc"
)

func main() {
	port := flag.Int("port", 50051, "Server port")
	dir := flag.String("dir", "data", "Data directory")
	flag.Parse()

	os.MkdirAll(*dir, 0755)
	engine, err := storage.NewStorageEngine(*dir)
	if err != nil {
		log.Fatalf("failed to start engine: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterKVServiceServer(s, network.NewStorageServer(engine))
	
	go func() {
		s.Serve(lis)
	}()
	select {}
}

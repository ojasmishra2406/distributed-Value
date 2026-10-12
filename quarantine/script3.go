package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/ojasmishra2406/distributed-Value/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial("localhost:50053", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewKVServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := client.Get(ctx, &pb.GetRequest{Key: []byte("mykey")})
	if err != nil {
		fmt.Printf("GET failed: %v\n", err)
	} else {
		fmt.Printf("Data survived on kv-node-2: %s\n", string(resp.Value))
	}
}

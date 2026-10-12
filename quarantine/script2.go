package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ojasmishra2406/distributed-Value/internal/network"
)

func main() {
	router := network.NewRouter(3, 3, 2, 2)
	router.RegisterNode("localhost:50051")
	router.RegisterNode("localhost:50052")
	router.RegisterNode("localhost:50053")

	fmt.Println("Waiting 3s for Node 3 to be killed...")
	time.Sleep(3 * time.Second)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := router.Put(ctx, []byte("key_down"), []byte("val_down"), nil)
	fmt.Printf("PUT key_down: err=%v\n", err)

	fmt.Println("Waiting 10s for Node 3 to restart and handoff to trigger...")
	time.Sleep(10 * time.Second)
	
	fmt.Println("Done script2.")
}

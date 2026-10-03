package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ojasmishra2406/distributed-Value/internal/network"
)

func main() {
	router := network.NewRouter(3, 3, 2, 2)
	err1 := router.RegisterNode("localhost:50051")
	err2 := router.RegisterNode("localhost:50052")
	err3 := router.RegisterNode("localhost:50053")

	if err1 != nil || err2 != nil || err3 != nil {
		fmt.Printf("Failed to register nodes: %v, %v, %v\n", err1, err2, err3)
		return
	}
	fmt.Println("Cluster nodes registered: localhost:50051, 50052, 50053")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := router.Put(ctx, []byte("mykey"), []byte("myval"), nil)
	if err != nil {
		fmt.Printf("PUT failed: %v\n", err)
		return
	}
	fmt.Println("PUT mykey=myval successful with W=2")

	val, _, _, _, err := router.Get(ctx, []byte("mykey"))
	if err != nil {
		fmt.Printf("GET failed: %v\n", err)
		return
	}
	fmt.Printf("GET mykey: %s (R=2)\n", string(val))
}

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
	
	ctxPut, _ := context.WithTimeout(context.Background(), 2*time.Second)
	err := router.Put(ctxPut, []byte("key_repair"), []byte("new_val"), nil)
	fmt.Printf("PUT new_val (Node 1,2): err=%v\n", err)
	
	router.RegisterNode("localhost:50053")
	time.Sleep(1 * time.Second)
	
	ctxGet, _ := context.WithTimeout(context.Background(), 2*time.Second)
	val, _, _, _, err := router.Get(ctxGet, []byte("key_repair"))
	fmt.Printf("GET key_repair (Triggers Read Repair): %s, err: %v\n", string(val), err)
	
	time.Sleep(2 * time.Second)
	fmt.Println("Done")
}

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ojasmishra2406/distributed-Value/internal/network"
)

func main() {
	// tickInterval=2s, maxMisses=3 → needs 6s to declare dead, then 1 more tick to replay
	router := network.NewRouter(3, 3, 2, 2)
	router.RegisterNode("localhost:50051")
	router.RegisterNode("localhost:50052")
	router.RegisterNode("localhost:50053")

	fmt.Println("PUT partition_key during partition (Node 3 isolated by iptables)...")
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	err := router.Put(ctx, []byte("partition_key"), []byte("partition_val"), nil)
	fmt.Printf("PUT during partition: err=%v\n", err)

	ctx2, _ := context.WithTimeout(context.Background(), 2*time.Second)
	val, _, _, _, err := router.Get(ctx2, []byte("partition_key"))
	fmt.Printf("GET during partition: val=%s err=%v\n", string(val), err)

	// Partition is healed externally (iptables rules removed by phase3.sh after 5s)
	// Wait 20s: detector needs 3 missed beats (6s) to declare dead, then 1 successful beat to ReplayHints
	fmt.Println("Waiting 20s for failure detector to detect dead→alive transition and replay hints...")
	time.Sleep(20 * time.Second)
	fmt.Println("phase3test done.")
}

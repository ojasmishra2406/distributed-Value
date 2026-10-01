package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mishr/distributed-kv/internal/metrics"
	"github.com/mishr/distributed-kv/internal/network"
)

func main() {
	fmt.Println("Starting High-Concurrency Benchmark...")

	go metrics.ServeMetrics(":8081")

	router := network.NewRouter(128, 3, 2, 2)
	router.RegisterNode("localhost:50051")
	router.RegisterNode("localhost:50052")
	router.RegisterNode("localhost:50053")

	numWorkers := 100
	opsPerWorker := 1000

	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < opsPerWorker; j++ {
				key := []byte(fmt.Sprintf("bench-key-%d-%d", workerID, j))
				val := []byte("bench-value")

				opStart := time.Now()
				err := router.Put(context.Background(), key, val)
				metrics.RecordLatency(float64(time.Since(opStart).Milliseconds()))
				metrics.RecordOp(err)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start).Seconds()
	
	ops := float64(numWorkers * opsPerWorker)
	p50, p95, p99 := metrics.GetPercentiles()

	fmt.Printf("Total Ops: %.0f\n", ops)
	fmt.Printf("Throughput: %.2f ops/sec\n", ops/duration)
	fmt.Printf("p50 Latency: %.2f ms\n", p50)
	fmt.Printf("p95 Latency: %.2f ms\n", p95)
	fmt.Printf("p99 Latency: %.2f ms\n", p99)
	fmt.Println("Metrics available at http://localhost:8081/metrics")
}

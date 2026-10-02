package main
import (
	"context"
	"fmt"
	"sync"
	"time"
	"sync/atomic"
	pb "github.com/ojasmishra2406/distributed-Value/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)
func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { panic(err) }
	defer conn.Close()
	client := pb.NewKVServiceClient(conn)
	numRequests := 1000
	concurrency := 20
	fmt.Printf("Starting Benchmark: %d requests, %d concurrent workers\n", numRequests, concurrency)
	start := time.Now()
	var wg sync.WaitGroup
	var successCount int32
	var errorCount int32
	ch := make(chan int, numRequests)
	for i := 0; i < numRequests; i++ { ch <- i }
	close(ch)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for req := range ch {
				key := []byte(fmt.Sprintf("key-%d", req))
				val := []byte(fmt.Sprintf("val-%d", req))
				_, err := client.Put(context.Background(), &pb.PutRequest{ Key: key, Value: val, Timestamp: time.Now().UnixNano() })
				if err == nil { atomic.AddInt32(&successCount, 1) } else { atomic.AddInt32(&errorCount, 1) }
			}
		}()
	}
	wg.Wait()
	duration := time.Since(start)
	opsPerSec := float64(successCount) / duration.Seconds()
	fmt.Printf("Writes Finished in %v (Success: %d, Errors: %d)\n", duration, successCount, errorCount)
	fmt.Printf("Throughput: %.2f ops/sec\n", opsPerSec)
	if successCount > 0 { fmt.Printf("Average Latency: %v\n", duration/time.Duration(successCount/int32(concurrency))) }
}

package main

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"time"

	"github.com/ojasmishra2406/distributed-Value/internal/storage"
)

func main() {
	os.RemoveAll("bench_vector_db")
	os.MkdirAll("bench_vector_db", 0755)

	engine, _ := storage.NewStorageEngine("bench_vector_db")

	const numVectors = 10000
	const dim = 128
	
	fmt.Printf("--- REAL INGESTION BENCHMARK ---\n")
	fmt.Printf("Generating %d vectors of dimension %d...\n", numVectors, dim)
	
	vectors := make([][]float32, numVectors)
	keys := make([][]byte, numVectors)
	for i := 0; i < numVectors; i++ {
		vec := make([]float32, dim)
		for j := 0; j < dim; j++ {
			vec[j] = rand.Float32()
		}
		vectors[i] = vec
		keys[i] = []byte(fmt.Sprintf("vec_%d", i))
	}
	
	fmt.Println("Starting ingestion...")
	start := time.Now()
	for i := 0; i < numVectors; i++ {
		engine.Put(keys[i], []byte("val"), time.Now().UnixNano(), vectors[i])
	}
	duration := time.Since(start)
	
	throughput := float64(numVectors) / duration.Seconds()
	fmt.Printf("Ingestion completed in %v\n", duration)
	fmt.Printf("Throughput: %.2f vectors/sec (persisted to disk)\n\n", throughput)

	// Query benchmark
	const numQueries = 100
	fmt.Printf("--- REAL QUERY BENCHMARK (Single Node Flat Search) ---\n")
	fmt.Printf("Executing %d random queries (topK=10)...\n", numQueries)
	
	latencies := make([]time.Duration, numQueries)
	for i := 0; i < numQueries; i++ {
		queryVec := make([]float32, dim)
		for j := 0; j < dim; j++ {
			queryVec[j] = rand.Float32()
		}
		
		qStart := time.Now()
		engine.Search(queryVec, 10)
		latencies[i] = time.Since(qStart)
	}
	
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})
	
	p50 := latencies[numQueries/2]
	p95 := latencies[int(float64(numQueries)*0.95)]
	max := latencies[numQueries-1]
	
	fmt.Printf("p50 Latency: %v\n", p50)
	fmt.Printf("p95 Latency: %v\n", p95)
	fmt.Printf("Max Latency: %v\n", max)
}

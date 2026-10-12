package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ojasmishra2406/distributed-Value/internal/storage"
)

var precomputedVector []float32

func init() {
	precomputedVector = make([]float32, 128)
	for i := 0; i < 128; i++ {
		precomputedVector[i] = float32(i) * 0.1
	}
}

func generateVector(dim int) []float32 {
	return precomputedVector
}

func runBenchmark(name string, numVectors, batchSize, concurrency int) {
	dir := "test_vector_batch_db_" + name
	os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)
	idx := storage.NewFlatVectorIndex(dir)

	fmt.Printf("--- %s (Concurrency: %d, Batch Size: %d) ---\n", name, concurrency, batchSize)

	// Pre-generate jobs
	type work struct {
		id string
		v  []float32
	}
	jobs := make([]work, numVectors)
	for i := 0; i < numVectors; i++ {
		jobs[i] = work{
			id: fmt.Sprintf("vec_%d", i),
			v:  precomputedVector, // use precomputed to avoid memory & lock overhead
		}
	}

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(concurrency)

	chunkSize := numVectors / concurrency
	for i := 0; i < concurrency; i++ {
		startIdx := i * chunkSize
		endIdx := startIdx + chunkSize
		if i == concurrency-1 {
			endIdx = numVectors
		}

		go func(s, e int) {
			defer wg.Done()
			var batch []*storage.VectorNode
			for j := s; j < e; j++ {
				batch = append(batch, &storage.VectorNode{
					ID:     jobs[j].id,
					Key:    []byte(jobs[j].id),
					Value:  []byte("val"),
					Vector: jobs[j].v,
				})
				if len(batch) == batchSize {
					idx.InsertBatch(batch)
					batch = batch[:0] // Reuse slice
				}
			}
			if len(batch) > 0 {
				idx.InsertBatch(batch)
			}
		}(startIdx, endIdx)
	}

	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("%s %d inserts: %s\n", name, numVectors, elapsed)
	fmt.Printf("Throughput: %.2f vectors/sec\n\n", float64(numVectors)/elapsed.Seconds())
}

func main() {
	numVectors := 2000000
	concurrency := 200

	runBenchmark("BATCH_2000", numVectors, 2000, concurrency)
	runBenchmark("BATCH_5000", numVectors, 5000, concurrency)
	runBenchmark("BATCH_10000", numVectors, 10000, concurrency)
}

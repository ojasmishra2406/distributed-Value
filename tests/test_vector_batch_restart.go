package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/ojasmishra2406/distributed-Value/internal/storage"
)

func generateVector(dim int) []float32 {
	v := make([]float32, dim)
	for i := 0; i < dim; i++ {
		v[i] = rand.Float32()
	}
	return v
}

func main() {
	dim := 128
	
	os.RemoveAll("test_vector_batch_restart_db")
	os.MkdirAll("test_vector_batch_restart_db", 0755)
	idx := storage.NewFlatVectorIndex("test_vector_batch_restart_db")

	var batch []*storage.VectorNode
	for i := 0; i < 500; i++ {
		id := fmt.Sprintf("vec_%d", i)
		batch = append(batch, &storage.VectorNode{ID: id, Key: []byte(id), Value: []byte("val"), Vector: generateVector(dim)})
	}
	idx.InsertBatch(batch)
	
	fmt.Println("Batch of 500 inserted. Querying vec_499 before restart...")
	results := idx.Search(batch[499].Vector, 1)
	if len(results) > 0 {
		fmt.Printf("Found 1 results. Top result ID: %s\n", results[0].ID)
	}
	
	// Simulate restart
	fmt.Println("\nRestarting engine (reading from disk)...")
	idx2 := storage.NewFlatVectorIndex("test_vector_batch_restart_db")
	
	results2 := idx2.Search(batch[499].Vector, 1)
	if len(results2) > 0 {
		fmt.Printf("Search after restart: Found 1 results. Top result ID: %s\n", results2[0].ID)
	}
}

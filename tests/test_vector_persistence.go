package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ojasmishra2406/distributed-Value/internal/storage"
)

func main() {
	os.RemoveAll("test_vector_db")
	os.MkdirAll("test_vector_db", 0755)

	engine, _ := storage.NewStorageEngine("test_vector_db")
	
	fmt.Println("Inserting vector for key1...")
	vec1 := []float32{1.0, 2.0, 3.0}
	engine.Put([]byte("key1"), []byte("val1"), time.Now().UnixNano(), vec1)
	
	results := engine.Search(vec1, 1)
	fmt.Printf("Search before restart: Found %d results. Top result ID: %s\n", len(results), results[0].ID)
	
	fmt.Println("\nRestarting engine (reading from disk)...")
	engine2, _ := storage.NewStorageEngine("test_vector_db")
	
	results2 := engine2.Search(vec1, 1)
	if len(results2) > 0 {
		fmt.Printf("Search after restart: Found %d results. Top result ID: %s\n", len(results2), results2[0].ID)
	} else {
		fmt.Println("Search after restart: Found 0 results!")
	}
}

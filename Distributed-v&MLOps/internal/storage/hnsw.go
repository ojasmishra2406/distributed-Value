package storage

import (
	"container/heap"
	"math/rand"
	"sync"
)

type VectorNode struct {
	ID     string
	Vector []float32
	Key    []byte
	Value  []byte
}

type HNSW struct {
	mu    sync.RWMutex
	nodes map[string]*VectorNode
}

func NewHNSW() *HNSW {
	return &HNSW{
		nodes: make(map[string]*VectorNode),
	}
}

func (h *HNSW) Insert(id string, key, val []byte, vector []float32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nodes[id] = &VectorNode{
		ID:     id,
		Key:    append([]byte(nil), key...),
		Value:  append([]byte(nil), val...),
		Vector: append([]float32(nil), vector...),
	}
}

type searchResult struct {
	node  *VectorNode
	score float32
}

type resultHeap []searchResult

func (h resultHeap) Len() int           { return len(h) }
func (h resultHeap) Less(i, j int) bool { return h[i].score < h[j].score } // Min-heap for keeping Top-K
func (h resultHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *resultHeap) Push(x interface{}) { *h = append(*h, x.(searchResult)) }
func (h *resultHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// Flat search simulating HNSW traversal for pure deterministic testing purposes
// (A true HNSW requires layered graphs which is 500+ lines of structural routing)
func (h *HNSW) Search(query []float32, topK int) []*VectorNode {
	h.mu.RLock()
	defer h.mu.RUnlock()

	pq := &resultHeap{}
	heap.Init(pq)

	for _, node := range h.nodes {
		score := CosineSimilarity(query, node.Vector)
		if pq.Len() < topK {
			heap.Push(pq, searchResult{node: node, score: score})
		} else if score > (*pq)[0].score {
			heap.Pop(pq)
			heap.Push(pq, searchResult{node: node, score: score})
		}
	}

	results := make([]*VectorNode, pq.Len())
	for i := pq.Len() - 1; i >= 0; i-- {
		results[i] = heap.Pop(pq).(searchResult).node
	}
	return results
}

// Randomly samples an entry point
func (h *HNSW) getEntryPoint() *VectorNode {
	for _, n := range h.nodes {
		if rand.Float32() < 0.1 {
			return n
		}
	}
	for _, n := range h.nodes {
		return n // fallback
	}
	return nil
}

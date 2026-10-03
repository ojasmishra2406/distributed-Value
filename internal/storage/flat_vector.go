package storage

import (
	"container/heap"
	"encoding/binary"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
)

type VectorNode struct {
	ID     string
	Vector []float32
	Key    []byte
	Value  []byte
}

type FlatVectorIndex struct {
	mu    sync.RWMutex
	nodes map[string]*VectorNode
	file  *os.File
}

func NewFlatVectorIndex(dir string) *FlatVectorIndex {
	idx := &FlatVectorIndex{
		nodes: make(map[string]*VectorNode),
	}
	if dir != "" {
		filePath := filepath.Join(dir, "vectors.wal")
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		if err == nil {
			idx.file = file
			idx.recover(filePath)
		}
	}
	return idx
}

func (h *FlatVectorIndex) recover(filePath string) {
	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	for {
		var idLen, keyLen, valLen, vecLen uint32
		if err := binary.Read(f, binary.LittleEndian, &idLen); err != nil {
			break
		}
		binary.Read(f, binary.LittleEndian, &keyLen)
		binary.Read(f, binary.LittleEndian, &valLen)
		binary.Read(f, binary.LittleEndian, &vecLen)

		idBuf := make([]byte, idLen)
		io.ReadFull(f, idBuf)
		keyBuf := make([]byte, keyLen)
		io.ReadFull(f, keyBuf)
		valBuf := make([]byte, valLen)
		io.ReadFull(f, valBuf)
		vec := make([]float32, vecLen)
		if vecLen > 0 {
			binary.Read(f, binary.LittleEndian, &vec)
		}

		h.nodes[string(idBuf)] = &VectorNode{
			ID:     string(idBuf),
			Key:    keyBuf,
			Value:  valBuf,
			Vector: vec,
		}
	}
}

func (h *FlatVectorIndex) Insert(id string, key, val []byte, vector []float32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nodes[id] = &VectorNode{
		ID:     id,
		Key:    append([]byte(nil), key...),
		Value:  append([]byte(nil), val...),
		Vector: append([]float32(nil), vector...),
	}

	if h.file != nil {
		binary.Write(h.file, binary.LittleEndian, uint32(len(id)))
		binary.Write(h.file, binary.LittleEndian, uint32(len(key)))
		binary.Write(h.file, binary.LittleEndian, uint32(len(val)))
		binary.Write(h.file, binary.LittleEndian, uint32(len(vector)))
		h.file.Write([]byte(id))
		h.file.Write(key)
		h.file.Write(val)
		if len(vector) > 0 {
			binary.Write(h.file, binary.LittleEndian, vector)
		}
		h.file.Sync()
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

func (h *FlatVectorIndex) Search(query []float32, topK int) []*VectorNode {
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

func (h *FlatVectorIndex) getEntryPoint() *VectorNode {
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

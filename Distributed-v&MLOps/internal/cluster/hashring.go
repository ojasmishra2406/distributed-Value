package cluster

import (
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
)

type HashRing struct {
	mu         sync.RWMutex
	tokens     []uint64
	nodes      map[uint64]string
	vnodeCount int
}

func NewHashRing(vnodeCount int) *HashRing {
	return &HashRing{
		tokens:     make([]uint64, 0),
		nodes:      make(map[uint64]string),
		vnodeCount: vnodeCount,
	}
}

func (r *HashRing) AddNode(node string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := 0; i < r.vnodeCount; i++ {
		vnodeID := fmt.Sprintf("%s#%d", node, i)
		hash := r.hashKey([]byte(vnodeID))
		r.tokens = append(r.tokens, hash)
		r.nodes[hash] = node
	}
	r.sortTokens()
}

func (r *HashRing) RemoveNode(node string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var newTokens []uint64
	for _, token := range r.tokens {
		if r.nodes[token] != node {
			newTokens = append(newTokens, token)
		} else {
			delete(r.nodes, token)
		}
	}
	r.tokens = newTokens
}

func (r *HashRing) GetNodes(key []byte, count int) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.tokens) == 0 {
		return nil, errors.New("hash ring is empty")
	}

	hash := r.hashKey(key)
	idx := sort.Search(len(r.tokens), func(i int) bool {
		return r.tokens[i] >= hash
	})

	if idx == len(r.tokens) {
		idx = 0
	}

	seen := make(map[string]bool)
	var targets []string

	for i := 0; i < len(r.tokens); i++ {
		currIdx := (idx + i) % len(r.tokens)
		nodeAddr := r.nodes[r.tokens[currIdx]]

		if !seen[nodeAddr] {
			seen[nodeAddr] = true
			targets = append(targets, nodeAddr)
			if len(targets) == count {
				break
			}
		}
	}

	if len(targets) == 0 {
		return nil, errors.New("no nodes available")
	}

	return targets, nil
}

func (r *HashRing) hashKey(key []byte) uint64 {
	h := fnv.New64a()
	h.Write(key)
	return h.Sum64()
}

func (r *HashRing) sortTokens() {
	sort.Slice(r.tokens, func(i, j int) bool {
		return r.tokens[i] < r.tokens[j]
	})
}

package cluster

import (
	"context"
	"sync"
	"time"

	pb "github.com/ojasmishra2406/distributed-Value/api/proto"
)

type HintedHandoffManager struct {
	mu    sync.Mutex
	hints map[string][]*pb.PutRequest
}

func NewHintedHandoffManager() *HintedHandoffManager {
	return &HintedHandoffManager{
		hints: make(map[string][]*pb.PutRequest),
	}
}

func (h *HintedHandoffManager) AddHint(node string, key, val []byte, ts int64, tomb bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	req := &pb.PutRequest{
		Key:       key,
		Value:     val,
		Timestamp: ts,
	}

	h.hints[node] = append(h.hints[node], req)
}

func (h *HintedHandoffManager) ReplayHints(node string, client pb.KVServiceClient) {
	h.mu.Lock()
	pending, exists := h.hints[node]
	if !exists || len(pending) == 0 {
		h.mu.Unlock()
		return
	}
	delete(h.hints, node)
	h.mu.Unlock()

	go func(target string, reqs []*pb.PutRequest, cl pb.KVServiceClient) {
		for _, req := range reqs {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, err := cl.Put(ctx, req)
			cancel()
			if err != nil {
				h.AddHint(target, req.Key, req.Value, req.Timestamp, false)
			}
		}
	}(node, pending, client)
}

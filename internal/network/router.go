package network

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	pb "github.com/ojasmishra2406/distributed-Value/api/proto"
	"github.com/ojasmishra2406/distributed-Value/internal/cluster"
	"github.com/ojasmishra2406/distributed-Value/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Router struct {
	ring        *cluster.HashRing
	conns       map[string]*grpc.ClientConn
	clients     map[string]pb.KVServiceClient
	coordinator *cluster.QuorumCoordinator
	detector    *cluster.FailureDetector
	handoff     *cluster.HintedHandoffManager
	mu          sync.RWMutex
}

func NewRouter(vnodeCount, n, w, r int) *Router {
	ring := cluster.NewHashRing(vnodeCount)
	handoff := cluster.NewHintedHandoffManager()
	coord := cluster.NewQuorumCoordinator(n, w, r, ring, handoff)
	detector := cluster.NewFailureDetector(2*time.Second, 3, handoff)
	
	router := &Router{
		ring:        ring,
		conns:       make(map[string]*grpc.ClientConn),
		clients:     make(map[string]pb.KVServiceClient),
		coordinator: coord,
		detector:    detector,
		handoff:     handoff,
	}
	detector.Start()
	return router
}

func (r *Router) RegisterNode(addr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.conns[addr]; exists {
		return nil
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to node %s: %v", addr, err)
	}

	client := pb.NewKVServiceClient(conn)
	r.conns[addr] = conn
	r.clients[addr] = client
	
	r.ring.AddNode(addr)
	r.coordinator.RegisterClient(addr, client)
	r.detector.RegisterNode(addr, client)
	
	return nil
}

func (r *Router) Put(ctx context.Context, key, value []byte, vector []float32) error {
	return r.coordinator.Put(ctx, key, value, vector)
}

func (r *Router) Get(ctx context.Context, key []byte) ([]byte, bool, int64, bool, error) {
	return r.coordinator.Get(ctx, key)
}

func (r *Router) Delete(ctx context.Context, key []byte) error {
	return nil
}

// Scatter-Gather Vector Search
func (r *Router) Search(ctx context.Context, vector []float32, topK int) ([]*pb.Record, error) {
	r.mu.RLock()
	clients := make([]pb.KVServiceClient, 0, len(r.clients))
	for _, c := range r.clients {
		clients = append(clients, c)
	}
	r.mu.RUnlock()

	resCh := make(chan *pb.SearchResponse, len(clients))
	var wg sync.WaitGroup

	req := &pb.SearchRequest{Vector: vector, TopK: int32(topK)}

	for _, client := range clients {
		wg.Add(1)
		go func(c pb.KVServiceClient) {
			defer wg.Done()
			tctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			resp, err := c.Search(tctx, req)
			if err == nil && resp != nil {
				resCh <- resp
			}
		}(client)
	}

	wg.Wait()
	close(resCh)

	var allRecords []*pb.Record
	for resp := range resCh {
		allRecords = append(allRecords, resp.Records...)
	}

	sort.Slice(allRecords, func(i, j int) bool {
		sI := storage.CosineSimilarity(vector, allRecords[i].Vector)
		sJ := storage.CosineSimilarity(vector, allRecords[j].Vector)
		return sI > sJ // Descending score
	})

	if len(allRecords) > topK {
		allRecords = allRecords[:topK]
	}

	return allRecords, nil
}

func (r *Router) Close() {
	r.detector.Stop()
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, conn := range r.conns {
		conn.Close()
	}
}

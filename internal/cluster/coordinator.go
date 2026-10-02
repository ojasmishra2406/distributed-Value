package cluster

import (
	"context"
	"errors"
	"sync"
	"time"

	pb "github.com/ojasmishra2406/distributed-Value/api/proto"
)

var (
	ErrWriteQuorumFailed = errors.New("write quorum failed")
	ErrReadQuorumFailed  = errors.New("read quorum failed")
)

type QuorumCoordinator struct {
	N, W, R int
	Ring    *HashRing
	Clients map[string]pb.KVServiceClient
	Handoff *HintedHandoffManager
	mu      sync.RWMutex
}

func NewQuorumCoordinator(n, w, r int, ring *HashRing, handoff *HintedHandoffManager) *QuorumCoordinator {
	return &QuorumCoordinator{N: n, W: w, R: r, Ring: ring, Clients: make(map[string]pb.KVServiceClient), Handoff: handoff}
}

func (c *QuorumCoordinator) RegisterClient(addr string, client pb.KVServiceClient) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Clients[addr] = client
}

type writeResult struct {
	node string
	err  error
}

func (c *QuorumCoordinator) Put(ctx context.Context, key, value []byte, vector []float32) error {
	nodes, err := c.Ring.GetNodes(key, c.N)
	if err != nil {
		return err
	}

	ts := time.Now().UnixNano()
	req := &pb.PutRequest{Key: key, Value: value, Timestamp: ts, Vector: vector}
	resCh := make(chan writeResult, c.N)
	doneCh := make(chan error, 1)

	c.mu.RLock()
	for _, node := range nodes {
		client, ok := c.Clients[node]
		if !ok {
			resCh <- writeResult{node: node, err: errors.New("offline")}
			continue
		}
		go func(n string, cl pb.KVServiceClient) {
			tctx, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()
			_, err := cl.Put(tctx, req)
			resCh <- writeResult{node: n, err: err}
		}(node, client)
	}
	c.mu.RUnlock()

	go func() {
		successes := 0
		signaled := false
		for i := 0; i < c.N; i++ {
			res := <-resCh
			if res.err == nil {
				successes++
				if successes == c.W && !signaled {
					doneCh <- nil
					signaled = true
				}
			} else {
				c.Handoff.AddHint(res.node, key, value, ts, false)
			}
		}
		if successes < c.W && !signaled {
			doneCh <- ErrWriteQuorumFailed
		}
	}()

	return <-doneCh
}

type readResult struct {
	node string
	resp *pb.GetResponse
	err  error
}

func (c *QuorumCoordinator) Get(ctx context.Context, key []byte) ([]byte, bool, int64, bool, error) {
	nodes, err := c.Ring.GetNodes(key, c.N)
	if err != nil {
		return nil, false, 0, false, err
	}

	req := &pb.GetRequest{Key: key}
	resCh := make(chan readResult, c.N)
	doneCh := make(chan readResult, 1)

	c.mu.RLock()
	for _, node := range nodes {
		client, ok := c.Clients[node]
		if !ok {
			resCh <- readResult{node: node, err: errors.New("offline")}
			continue
		}
		go func(n string, cl pb.KVServiceClient) {
			tctx, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()
			resp, err := cl.Get(tctx, req)
			resCh <- readResult{node: n, resp: resp, err: err}
		}(node, client)
	}
	c.mu.RUnlock()

	go func() {
		var successes int
		var bestResp *pb.GetResponse
		var staleNodes []string
		signaled := false

		for i := 0; i < c.N; i++ {
			res := <-resCh
			if res.err == nil {
				successes++
				if bestResp == nil || res.resp.Timestamp > bestResp.Timestamp {
					if bestResp != nil {
						staleNodes = append(staleNodes, res.node) // Previous best is now stale
					}
					bestResp = res.resp
				} else if res.resp.Timestamp < bestResp.Timestamp {
					staleNodes = append(staleNodes, res.node)
				}

				if successes == c.R && !signaled {
					doneCh <- readResult{resp: bestResp, err: nil}
					signaled = true
				}
			}
		}

		if successes < c.R && !signaled {
			doneCh <- readResult{err: ErrReadQuorumFailed}
		}

		if bestResp != nil && len(staleNodes) > 0 {
			c.triggerReadRepair(staleNodes, key, bestResp)
		}
	}()

	finalRes := <-doneCh
	if finalRes.err != nil {
		return nil, false, 0, false, finalRes.err
	}
	return finalRes.resp.Value, finalRes.resp.Found, finalRes.resp.Timestamp, finalRes.resp.Tombstone, nil
}

func (c *QuorumCoordinator) triggerReadRepair(staleNodes []string, key []byte, latest *pb.GetResponse) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	req := &pb.PutRequest{Key: key, Value: latest.Value, Timestamp: latest.Timestamp}
	for _, n := range staleNodes {
		if cl, ok := c.Clients[n]; ok {
			go func(client pb.KVServiceClient) {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				client.Put(ctx, req)
			}(cl)
		}
	}
}

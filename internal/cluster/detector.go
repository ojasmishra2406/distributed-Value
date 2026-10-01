package cluster

import (
	"context"
	"sync"
	"time"

	pb "github.com/mishr/distributed-kv/api/proto"
)

type NodeStatus string

const (
	StatusAlive      NodeStatus = "ALIVE"
	StatusDead       NodeStatus = "DEAD"
	StatusRecovering NodeStatus = "RECOVERING"
)

type FailureDetector struct {
	mu           sync.RWMutex
	statuses     map[string]NodeStatus
	missedBeats  map[string]int
	clients      map[string]pb.KVServiceClient
	handoff      *HintedHandoffManager
	tickInterval time.Duration
	maxMisses    int
	stopCh       chan struct{}
}

func NewFailureDetector(interval time.Duration, maxMisses int, handoff *HintedHandoffManager) *FailureDetector {
	return &FailureDetector{
		statuses:     make(map[string]NodeStatus),
		missedBeats:  make(map[string]int),
		clients:      make(map[string]pb.KVServiceClient),
		handoff:      handoff,
		tickInterval: interval,
		maxMisses:    maxMisses,
		stopCh:       make(chan struct{}),
	}
}

func (fd *FailureDetector) RegisterNode(addr string, client pb.KVServiceClient) {
	fd.mu.Lock()
	defer fd.mu.Unlock()
	fd.clients[addr] = client
	fd.statuses[addr] = StatusAlive
	fd.missedBeats[addr] = 0
}

func (fd *FailureDetector) Start() {
	go func() {
		ticker := time.NewTicker(fd.tickInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fd.pingAll()
			case <-fd.stopCh:
				return
			}
		}
	}()
}

func (fd *FailureDetector) Stop() {
	close(fd.stopCh)
}

func (fd *FailureDetector) pingAll() {
	fd.mu.RLock()
	nodes := make([]string, 0, len(fd.clients))
	for node := range fd.clients {
		nodes = append(nodes, node)
	}
	fd.mu.RUnlock()

	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			fd.pingNode(n)
		}(node)
	}
	wg.Wait()
}

func (fd *FailureDetector) pingNode(node string) {
	fd.mu.RLock()
	client := fd.clients[node]
	status := fd.statuses[node]
	fd.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), fd.tickInterval/2)
	defer cancel()

	_, err := client.Heartbeat(ctx, &pb.HeartbeatRequest{NodeId: "gateway"})

	fd.mu.Lock()
	defer fd.mu.Unlock()

	if err != nil {
		fd.missedBeats[node]++
		if fd.missedBeats[node] >= fd.maxMisses && status == StatusAlive {
			fd.statuses[node] = StatusDead
		}
	} else {
		fd.missedBeats[node] = 0
		if status == StatusDead {
			fd.statuses[node] = StatusRecovering
			fd.handoff.ReplayHints(node, client)
			fd.statuses[node] = StatusAlive
		}
	}
}

package internal

import (
	"context"
	"testing"
	"time"

	pb "github.com/ojasmishra2406/distributed-Value/api/proto"
	"github.com/ojasmishra2406/distributed-Value/internal/cluster"
)

func TestChaos_JepsenNetworkPartition(t *testing.T) {
	ring := cluster.NewHashRing(1)
	handoff := cluster.NewHintedHandoffManager()
	
	// N=5, W=3, R=3
	coord := cluster.NewQuorumCoordinator(5, 3, 3, ring, handoff)

	nodes := make(map[string]*MockKVClient)
	for _, n := range []string{"A", "B", "C", "D", "E"} {
		node := &MockKVClient{data: make(map[string]*pb.PutRequest)}
		nodes[n] = node
		ring.AddNode(n)
		coord.RegisterClient(n, node)
	}

	// 1. Simulate a Network Partition splitting {A,B,C} and {D,E}
	// Assuming coordinator routes to A,B,C,D,E in that order for the ring.
	// If the coordinator is isolated with D and E (Minority):
	nodes["A"].setOffline(true)
	nodes["B"].setOffline(true)
	nodes["C"].setOffline(true)

	err := coord.Put(context.Background(), []byte("key-minority"), []byte("val"), nil)
	if err == nil {
		t.Fatalf("Expected minority write to fail (W=3, but only 2 reachable)")
	}

	// 2. Heal the partition, and isolate D and E instead (Coordinator with Majority)
	nodes["A"].setOffline(false)
	nodes["B"].setOffline(false)
	nodes["C"].setOffline(false)
	
	nodes["D"].setOffline(true)
	nodes["E"].setOffline(true)

	ts := time.Now().UnixNano()
	err = coord.Put(context.Background(), []byte("key-majority"), []byte("val"), nil)
	if err != nil {
		t.Fatalf("Expected majority write to succeed, got error: %v", err)
	}

	// Verify majority holds the data
	for _, n := range []string{"A", "B", "C"} {
		nodes[n].mu.RLock()
		if string(nodes[n].data["key-majority"].Value) != "val" {
			t.Fatalf("Node %s missing data", n)
		}
		nodes[n].mu.RUnlock()
	}

	// 3. Heal the network completely and trigger hinted handoff for D and E
	nodes["D"].setOffline(false)
	nodes["E"].setOffline(false)

	// In a real system, the failure detector triggers this. Manually trigger here:
	handoff.ReplayHints("D", nodes["D"])
	handoff.ReplayHints("E", nodes["E"])

	time.Sleep(100 * time.Millisecond) // Let goroutines flush

	// Verify hinted handoff successfully converged D and E
	for _, n := range []string{"D", "E"} {
		nodes[n].mu.RLock()
		req, exists := nodes[n].data["key-majority"]
		nodes[n].mu.RUnlock()
		
		if !exists {
			t.Fatalf("Node %s missed hinted handoff recovery", n)
		}
		if req.Timestamp != ts {
			t.Fatalf("Timestamp mismatch on recovered node %s", n)
		}
	}
}

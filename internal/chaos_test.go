package internal

import (
	"context"
	"testing"
	"time"
	"sync"

	pb "github.com/ojasmishra2406/distributed-Value/api/proto"
	"github.com/ojasmishra2406/distributed-Value/internal/cluster"
	"google.golang.org/grpc"
)

type MockKVClient struct {
	mu        sync.RWMutex
	data      map[string]*pb.PutRequest
	isOffline bool
}

func (m *MockKVClient) Put(ctx context.Context, in *pb.PutRequest, opts ...grpc.CallOption) (*pb.PutResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.isOffline {
		return nil, context.DeadlineExceeded
	}
	m.data[string(in.Key)] = in
	return &pb.PutResponse{}, nil
}

func (m *MockKVClient) Get(ctx context.Context, in *pb.GetRequest, opts ...grpc.CallOption) (*pb.GetResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.isOffline {
		return nil, context.DeadlineExceeded
	}
	if req, ok := m.data[string(in.Key)]; ok {
		return &pb.GetResponse{Value: req.Value, Found: true, Timestamp: req.Timestamp}, nil
	}
	return &pb.GetResponse{Found: false}, nil
}

func (m *MockKVClient) Delete(ctx context.Context, in *pb.DeleteRequest, opts ...grpc.CallOption) (*pb.DeleteResponse, error) { return &pb.DeleteResponse{}, nil }
func (m *MockKVClient) Search(ctx context.Context, in *pb.SearchRequest, opts ...grpc.CallOption) (*pb.SearchResponse, error) { return &pb.SearchResponse{}, nil }
func (m *MockKVClient) StreamPut(ctx context.Context, opts ...grpc.CallOption) (pb.KVService_StreamPutClient, error) { return nil, nil }
func (m *MockKVClient) Heartbeat(ctx context.Context, in *pb.HeartbeatRequest, opts ...grpc.CallOption) (*pb.HeartbeatResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.isOffline { return nil, context.DeadlineExceeded }
	return &pb.HeartbeatResponse{Alive: true}, nil
}
func (m *MockKVClient) Sync(ctx context.Context, in *pb.SyncRequest, opts ...grpc.CallOption) (*pb.SyncResponse, error) { return &pb.SyncResponse{}, nil }

func (m *MockKVClient) setOffline(status bool) {
	m.mu.Lock()
	m.isOffline = status
	m.mu.Unlock()
}

func TestChaos_QuorumReadRepair(t *testing.T) {
	ring := cluster.NewHashRing(1)
	handoff := cluster.NewHintedHandoffManager()
	coord := cluster.NewQuorumCoordinator(3, 2, 2, ring, handoff)

	nodeA := &MockKVClient{data: make(map[string]*pb.PutRequest)}
	nodeB := &MockKVClient{data: make(map[string]*pb.PutRequest)}
	nodeC := &MockKVClient{data: make(map[string]*pb.PutRequest)}

	ring.AddNode("A"); ring.AddNode("B"); ring.AddNode("C")
	coord.RegisterClient("A", nodeA)
	coord.RegisterClient("B", nodeB)
	coord.RegisterClient("C", nodeC)

	key := []byte("repair-key")
	
	// Simulate node A missing the update by pre-injecting old data
	nodeA.data[string(key)] = &pb.PutRequest{Value: []byte("old"), Timestamp: 100}
	nodeB.data[string(key)] = &pb.PutRequest{Value: []byte("new"), Timestamp: 200}
	nodeC.data[string(key)] = &pb.PutRequest{Value: []byte("new"), Timestamp: 200}

	val, _, _, _, err := coord.Get(context.Background(), key)
	if err != nil { t.Fatal(err) }
	if string(val) != "new" { t.Fatalf("Expected 'new', got '%s'", string(val)) }

	// Allow read repair to run
	time.Sleep(100 * time.Millisecond)

	nodeA.mu.RLock()
	repairedVal := nodeA.data[string(key)].Value
	nodeA.mu.RUnlock()
	if string(repairedVal) != "new" { t.Fatalf("Node A was not repaired") }
}

func TestChaos_GracefulDegradationAndRecovery(t *testing.T) {
	ring := cluster.NewHashRing(1)
	handoff := cluster.NewHintedHandoffManager()
	coord := cluster.NewQuorumCoordinator(3, 2, 2, ring, handoff)
	detector := cluster.NewFailureDetector(10*time.Millisecond, 2, handoff)

	nodeA := &MockKVClient{data: make(map[string]*pb.PutRequest)}
	nodeB := &MockKVClient{data: make(map[string]*pb.PutRequest)}
	nodeC := &MockKVClient{data: make(map[string]*pb.PutRequest)}

	ring.AddNode("A"); ring.AddNode("B"); ring.AddNode("C")
	coord.RegisterClient("A", nodeA); detector.RegisterNode("A", nodeA)
	coord.RegisterClient("B", nodeB); detector.RegisterNode("B", nodeB)
	coord.RegisterClient("C", nodeC); detector.RegisterNode("C", nodeC)

	detector.Start()

	nodeA.setOffline(true)
	
	// W=2 should still succeed
	key := []byte("crash-key")
	err := coord.Put(context.Background(), key, []byte("crash-val"), nil)
	if err != nil { t.Fatalf("Write should succeed with 2 nodes, got err: %v", err) }

	time.Sleep(50 * time.Millisecond) // Let failure detector mark DEAD
	
	nodeA.setOffline(false)
	time.Sleep(50 * time.Millisecond) // Let failure detector RECOVER and flush handoff

	nodeA.mu.RLock()
	_, exists := nodeA.data[string(key)]
	nodeA.mu.RUnlock()

	if !exists { t.Fatalf("Node A did not recover data via hinted handoff") }
}

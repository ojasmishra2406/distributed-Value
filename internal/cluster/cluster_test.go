package cluster

import (
	"context"
	"testing"
	"time"
	"sync"
	"google.golang.org/grpc"
	pb "github.com/ojasmishra2406/distributed-Value/api/proto"
)

func TestHashRing_Assignment(t *testing.T) {
	ring := NewHashRing(3)
	ring.AddNode("nodeA")
	ring.AddNode("nodeB")
	nodes, _ := ring.GetNodes([]byte("key1"), 1)
	if len(nodes) == 0 { t.Fatalf("Expected a node") }
	ring.RemoveNode("nodeA")
	nodes2, _ := ring.GetNodes([]byte("key1"), 1)
	if len(nodes2) == 0 || nodes2[0] != "nodeB" { t.Fatalf("Remapping failed") }
}

type MockClient struct {
	mu        sync.RWMutex
	data      map[string]*pb.PutRequest
	isOffline bool
}

func (m *MockClient) Put(ctx context.Context, in *pb.PutRequest, opts ...grpc.CallOption) (*pb.PutResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.isOffline { return nil, context.DeadlineExceeded }
	if m.data == nil { m.data = make(map[string]*pb.PutRequest) }
	m.data[string(in.Key)] = in
	return &pb.PutResponse{}, nil
}

func (m *MockClient) Get(ctx context.Context, in *pb.GetRequest, opts ...grpc.CallOption) (*pb.GetResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.isOffline { return nil, context.DeadlineExceeded }
	if req, ok := m.data[string(in.Key)]; ok {
		return &pb.GetResponse{Value: req.Value, Found: true, Timestamp: req.Timestamp}, nil
	}
	return &pb.GetResponse{Found: false}, nil
}

func (m *MockClient) Delete(ctx context.Context, in *pb.DeleteRequest, opts ...grpc.CallOption) (*pb.DeleteResponse, error) { return &pb.DeleteResponse{}, nil }
func (m *MockClient) Search(ctx context.Context, in *pb.SearchRequest, opts ...grpc.CallOption) (*pb.SearchResponse, error) { return &pb.SearchResponse{}, nil }
func (m *MockClient) StreamPut(ctx context.Context, opts ...grpc.CallOption) (pb.KVService_StreamPutClient, error) { return nil, nil }
func (m *MockClient) Heartbeat(ctx context.Context, in *pb.HeartbeatRequest, opts ...grpc.CallOption) (*pb.HeartbeatResponse, error) { return &pb.HeartbeatResponse{Alive: true}, nil }
func (m *MockClient) Sync(ctx context.Context, in *pb.SyncRequest, opts ...grpc.CallOption) (*pb.SyncResponse, error) { return &pb.SyncResponse{}, nil }

func TestCoordinator_QuorumThreshold(t *testing.T) {
	ring := NewHashRing(3)
	handoff := NewHintedHandoffManager()
	coord := NewQuorumCoordinator(3, 2, 2, ring, handoff)

	nodeA := &MockClient{data: make(map[string]*pb.PutRequest)}
	nodeB := &MockClient{data: make(map[string]*pb.PutRequest), isOffline: true}
	nodeC := &MockClient{data: make(map[string]*pb.PutRequest), isOffline: true}

	ring.AddNode("A"); ring.AddNode("B"); ring.AddNode("C")
	coord.RegisterClient("A", nodeA)
	coord.RegisterClient("B", nodeB)
	coord.RegisterClient("C", nodeC)

	// W=2 should FAIL because only A is online
	err := coord.Put(context.Background(), []byte("key"), []byte("val"), nil)
	if err == nil { t.Fatalf("Expected quorum write to fail") }

	// Bring B online, W=2 should SUCCEED
	nodeB.mu.Lock(); nodeB.isOffline = false; nodeB.mu.Unlock()
	err = coord.Put(context.Background(), []byte("key"), []byte("val"), nil)
	if err != nil { t.Fatalf("Expected quorum write to succeed: %v", err) }
}

func TestCoordinator_LWWConflictResolution(t *testing.T) {
	ring := NewHashRing(1)
	handoff := NewHintedHandoffManager()
	coord := NewQuorumCoordinator(3, 2, 2, ring, handoff)

	nodeA := &MockClient{data: make(map[string]*pb.PutRequest)}
	nodeB := &MockClient{data: make(map[string]*pb.PutRequest)}
	nodeC := &MockClient{data: make(map[string]*pb.PutRequest)}

	ring.AddNode("A"); ring.AddNode("B"); ring.AddNode("C")
	coord.RegisterClient("A", nodeA)
	coord.RegisterClient("B", nodeB)
	coord.RegisterClient("C", nodeC)

	// Inject conflicting data: Node A has older timestamp, Node B has newer
	nodeA.data["key"] = &pb.PutRequest{Key: []byte("key"), Value: []byte("val1"), Timestamp: 100}
	nodeB.data["key"] = &pb.PutRequest{Key: []byte("key"), Value: []byte("val2"), Timestamp: 200}
	nodeC.data["key"] = &pb.PutRequest{Key: []byte("key"), Value: []byte("val2"), Timestamp: 200}

	val, _, _, _, err := coord.Get(context.Background(), []byte("key"))
	if err != nil { t.Fatal(err) }
	
	if string(val) != "val2" {
		t.Fatalf("Expected val2 (LWW), got %s", string(val))
	}
}

func TestHintedHandoff_QueueAndReplay(t *testing.T) {
	handoff := NewHintedHandoffManager()
	handoff.AddHint("nodeA", []byte("k1"), []byte("v1"), 100, false)
	
	handoff.mu.Lock()
	if len(handoff.hints["nodeA"]) != 1 {
		t.Fatalf("Hint not queued")
	}
	handoff.mu.Unlock()

	nodeA := &MockClient{data: make(map[string]*pb.PutRequest)}
	
	handoff.ReplayHints("nodeA", nodeA)
	
	// Replay is async, wait a bit
	time.Sleep(100 * time.Millisecond)

	nodeA.mu.RLock()
	req, ok := nodeA.data["k1"]
	nodeA.mu.RUnlock()

	if !ok || string(req.Value) != "v1" {
		t.Fatalf("Hint was not replayed correctly")
	}
	
	handoff.mu.Lock()
	if len(handoff.hints["nodeA"]) != 0 {
		t.Fatalf("Hint queue not cleared after replay")
	}
	handoff.mu.Unlock()
}

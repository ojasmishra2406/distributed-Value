package cluster

import (
	"context"

	pb "github.com/mishr/distributed-kv/api/proto"
	"github.com/mishr/distributed-kv/internal/storage"
)

type NodeSynchronizer struct {
	engine  *storage.StorageEngine
	ring    *HashRing
	clients map[string]pb.KVServiceClient
}

func NewNodeSynchronizer(engine *storage.StorageEngine, ring *HashRing, clients map[string]pb.KVServiceClient) *NodeSynchronizer {
	return &NodeSynchronizer{
		engine:  engine,
		ring:    ring,
		clients: clients,
	}
}

func (s *NodeSynchronizer) SyncFromPeers(minTimestamp int64) error {
	for _, client := range s.clients {
		resp, err := client.Sync(context.Background(), &pb.SyncRequest{MinTimestamp: minTimestamp})
		if err == nil {
			for _, rec := range resp.Records {
				s.engine.ApplyReplication(rec.Key, rec.Value, rec.Timestamp, rec.Tombstone)
			}
			return nil
		}
	}
	return nil
}

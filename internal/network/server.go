package network

import (
	"context"

	pb "github.com/mishr/distributed-kv/api/proto"
	"github.com/mishr/distributed-kv/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StorageServer struct {
	engine *storage.StorageEngine
}

func NewStorageServer(engine *storage.StorageEngine) *StorageServer {
	return &StorageServer{engine: engine}
}

func (s *StorageServer) Put(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	err := s.engine.Put(req.Key, req.Value, req.Timestamp, req.Vector)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to put: %v", err)
	}
	return &pb.PutResponse{}, nil
}

func (s *StorageServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	val, tomb, ts, found, err := s.engine.Get(req.Key)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get: %v", err)
	}
	return &pb.GetResponse{
		Value:     val,
		Found:     found,
		Tombstone: tomb,
		Timestamp: ts,
	}, nil
}

func (s *StorageServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	err := s.engine.Delete(req.Key, req.Timestamp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete: %v", err)
	}
	return &pb.DeleteResponse{}, nil
}

func (s *StorageServer) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	results := s.engine.Search(req.Vector, int(req.TopK))
	
	resp := &pb.SearchResponse{
		Records: make([]*pb.Record, 0, len(results)),
	}

	for _, n := range results {
		resp.Records = append(resp.Records, &pb.Record{
			Key:    n.Key,
			Value:  n.Value,
			Vector: n.Vector,
		})
	}

	return resp, nil
}

func (s *StorageServer) StreamPut(stream pb.KVService_StreamPutServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return stream.SendAndClose(&pb.PutResponse{})
		}
		s.engine.Put(req.Key, req.Value, req.Timestamp, req.Vector)
	}
}

func (s *StorageServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	return &pb.HeartbeatResponse{Alive: true}, nil
}

func (s *StorageServer) Sync(ctx context.Context, req *pb.SyncRequest) (*pb.SyncResponse, error) {
	keys, err := s.engine.ScanSince(0)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to scan: %v", err)
	}
	
	resp := &pb.SyncResponse{
		Records: make([]*pb.Record, 0, len(keys)),
	}
	
	for _, k := range keys {
		v, tomb, ts, found, err := s.engine.Get(k)
		if err == nil && found {
			resp.Records = append(resp.Records, &pb.Record{
				Key: k,
				Value: v,
				Timestamp: ts,
				Tombstone: tomb,
			})
		}
	}
	return resp, nil
}

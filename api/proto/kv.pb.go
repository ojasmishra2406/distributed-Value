package proto

import (
	"context"
	"google.golang.org/grpc"
)

// Message definitions
type PutRequest struct {
	Key       []byte    `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value     []byte    `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	Timestamp int64     `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Vector    []float32 `protobuf:"fixed32,4,rep,packed,name=vector,proto3" json:"vector,omitempty"`
}
func (*PutRequest) Reset() {}
func (*PutRequest) String() string { return "PutRequest" }
func (*PutRequest) ProtoMessage() {}

type PutResponse struct{}
func (*PutResponse) Reset() {}
func (*PutResponse) String() string { return "PutResponse" }
func (*PutResponse) ProtoMessage() {}

type GetRequest struct {
	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}
func (*GetRequest) Reset() {}
func (*GetRequest) String() string { return "GetRequest" }
func (*GetRequest) ProtoMessage() {}

type GetResponse struct {
	Value     []byte `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	Found     bool   `protobuf:"varint,2,opt,name=found,proto3" json:"found,omitempty"`
	Tombstone bool   `protobuf:"varint,3,opt,name=tombstone,proto3" json:"tombstone,omitempty"`
	Timestamp int64  `protobuf:"varint,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}
func (*GetResponse) Reset() {}
func (*GetResponse) String() string { return "GetResponse" }
func (*GetResponse) ProtoMessage() {}

type DeleteRequest struct {
	Key       []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Timestamp int64  `protobuf:"varint,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}
func (*DeleteRequest) Reset() {}
func (*DeleteRequest) String() string { return "DeleteRequest" }
func (*DeleteRequest) ProtoMessage() {}

type DeleteResponse struct{}
func (*DeleteResponse) Reset() {}
func (*DeleteResponse) String() string { return "DeleteResponse" }
func (*DeleteResponse) ProtoMessage() {}

type SearchRequest struct {
	Vector []float32 `protobuf:"fixed32,1,rep,packed,name=vector,proto3" json:"vector,omitempty"`
	TopK   int32     `protobuf:"varint,2,opt,name=top_k,json=topK,proto3" json:"top_k,omitempty"`
}
func (*SearchRequest) Reset() {}
func (*SearchRequest) String() string { return "SearchRequest" }
func (*SearchRequest) ProtoMessage() {}

type Record struct {
	Key       []byte    `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value     []byte    `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	Timestamp int64     `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Tombstone bool      `protobuf:"varint,4,opt,name=tombstone,proto3" json:"tombstone,omitempty"`
	Vector    []float32 `protobuf:"fixed32,5,rep,packed,name=vector,proto3" json:"vector,omitempty"`
}
func (*Record) Reset() {}
func (*Record) String() string { return "Record" }
func (*Record) ProtoMessage() {}

type SearchResponse struct {
	Records []*Record `protobuf:"bytes,1,rep,name=records,proto3" json:"records,omitempty"`
}
func (*SearchResponse) Reset() {}
func (*SearchResponse) String() string { return "SearchResponse" }
func (*SearchResponse) ProtoMessage() {}

type HeartbeatRequest struct {
	NodeId string `protobuf:"bytes,1,opt,name=node_id,json=nodeId,proto3" json:"node_id,omitempty"`
}
func (*HeartbeatRequest) Reset() {}
func (*HeartbeatRequest) String() string { return "HeartbeatRequest" }
func (*HeartbeatRequest) ProtoMessage() {}

type HeartbeatResponse struct {
	Alive bool `protobuf:"varint,1,opt,name=alive,proto3" json:"alive,omitempty"`
}
func (*HeartbeatResponse) Reset() {}
func (*HeartbeatResponse) String() string { return "HeartbeatResponse" }
func (*HeartbeatResponse) ProtoMessage() {}

type SyncRequest struct {
	MinTimestamp int64 `protobuf:"varint,1,opt,name=min_timestamp,json=minTimestamp,proto3" json:"min_timestamp,omitempty"`
}
func (*SyncRequest) Reset() {}
func (*SyncRequest) String() string { return "SyncRequest" }
func (*SyncRequest) ProtoMessage() {}

type SyncResponse struct {
	Records []*Record `protobuf:"bytes,1,rep,name=records,proto3" json:"records,omitempty"`
}
func (*SyncResponse) Reset() {}
func (*SyncResponse) String() string { return "SyncResponse" }
func (*SyncResponse) ProtoMessage() {}

// Client Interface
type KVServiceClient interface {
	Put(ctx context.Context, in *PutRequest, opts ...grpc.CallOption) (*PutResponse, error)
	StreamPut(ctx context.Context, opts ...grpc.CallOption) (KVService_StreamPutClient, error)
	Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error)
	Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error)
	Search(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error)
	Heartbeat(ctx context.Context, in *HeartbeatRequest, opts ...grpc.CallOption) (*HeartbeatResponse, error)
	Sync(ctx context.Context, in *SyncRequest, opts ...grpc.CallOption) (*SyncResponse, error)
}

type kVServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewKVServiceClient(cc grpc.ClientConnInterface) KVServiceClient {
	return &kVServiceClient{cc}
}

func (c *kVServiceClient) Put(ctx context.Context, in *PutRequest, opts ...grpc.CallOption) (*PutResponse, error) {
	out := new(PutResponse)
	err := c.cc.Invoke(ctx, "/kv.KVService/Put", in, out, opts...)
	return out, err
}

func (c *kVServiceClient) StreamPut(ctx context.Context, opts ...grpc.CallOption) (KVService_StreamPutClient, error) {
	stream, err := c.cc.NewStream(ctx, &KVService_ServiceDesc.Streams[0], "/kv.KVService/StreamPut", opts...)
	if err != nil {
		return nil, err
	}
	x := &kVServiceStreamPutClient{stream}
	return x, nil
}

type KVService_StreamPutClient interface {
	Send(*PutRequest) error
	CloseAndRecv() (*PutResponse, error)
	grpc.ClientStream
}

type kVServiceStreamPutClient struct {
	grpc.ClientStream
}

func (x *kVServiceStreamPutClient) Send(m *PutRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *kVServiceStreamPutClient) CloseAndRecv() (*PutResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(PutResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *kVServiceClient) Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error) {
	out := new(GetResponse)
	err := c.cc.Invoke(ctx, "/kv.KVService/Get", in, out, opts...)
	return out, err
}

func (c *kVServiceClient) Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error) {
	out := new(DeleteResponse)
	err := c.cc.Invoke(ctx, "/kv.KVService/Delete", in, out, opts...)
	return out, err
}

func (c *kVServiceClient) Search(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error) {
	out := new(SearchResponse)
	err := c.cc.Invoke(ctx, "/kv.KVService/Search", in, out, opts...)
	return out, err
}

func (c *kVServiceClient) Heartbeat(ctx context.Context, in *HeartbeatRequest, opts ...grpc.CallOption) (*HeartbeatResponse, error) {
	out := new(HeartbeatResponse)
	err := c.cc.Invoke(ctx, "/kv.KVService/Heartbeat", in, out, opts...)
	return out, err
}

func (c *kVServiceClient) Sync(ctx context.Context, in *SyncRequest, opts ...grpc.CallOption) (*SyncResponse, error) {
	out := new(SyncResponse)
	err := c.cc.Invoke(ctx, "/kv.KVService/Sync", in, out, opts...)
	return out, err
}

// Server Interface
type KVServiceServer interface {
	Put(context.Context, *PutRequest) (*PutResponse, error)
	StreamPut(KVService_StreamPutServer) error
	Get(context.Context, *GetRequest) (*GetResponse, error)
	Delete(context.Context, *DeleteRequest) (*DeleteResponse, error)
	Search(context.Context, *SearchRequest) (*SearchResponse, error)
	Heartbeat(context.Context, *HeartbeatRequest) (*HeartbeatResponse, error)
	Sync(context.Context, *SyncRequest) (*SyncResponse, error)
}

func RegisterKVServiceServer(s grpc.ServiceRegistrar, srv KVServiceServer) {
	s.RegisterService(&KVService_ServiceDesc, srv)
}

type KVService_StreamPutServer interface {
	SendAndClose(*PutResponse) error
	Recv() (*PutRequest, error)
	grpc.ServerStream
}

type kVServiceStreamPutServer struct {
	grpc.ServerStream
}

func (x *kVServiceStreamPutServer) SendAndClose(m *PutResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *kVServiceStreamPutServer) Recv() (*PutRequest, error) {
	m := new(PutRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _KVService_Put_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PutRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KVServiceServer).Put(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kv.KVService/Put",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KVServiceServer).Put(ctx, req.(*PutRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _KVService_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KVServiceServer).Get(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kv.KVService/Get",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KVServiceServer).Get(ctx, req.(*GetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _KVService_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KVServiceServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kv.KVService/Delete",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KVServiceServer).Delete(ctx, req.(*DeleteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _KVService_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SearchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KVServiceServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kv.KVService/Search",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KVServiceServer).Search(ctx, req.(*SearchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _KVService_Heartbeat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HeartbeatRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KVServiceServer).Heartbeat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kv.KVService/Heartbeat",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KVServiceServer).Heartbeat(ctx, req.(*HeartbeatRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _KVService_Sync_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SyncRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KVServiceServer).Sync(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kv.KVService/Sync",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KVServiceServer).Sync(ctx, req.(*SyncRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _KVService_StreamPut_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(KVServiceServer).StreamPut(&kVServiceStreamPutServer{stream})
}

var KVService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "kv.KVService",
	HandlerType: (*KVServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Put",
			Handler:    _KVService_Put_Handler,
		},
		{
			MethodName: "Get",
			Handler:    _KVService_Get_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _KVService_Delete_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _KVService_Search_Handler,
		},
		{
			MethodName: "Heartbeat",
			Handler:    _KVService_Heartbeat_Handler,
		},
		{
			MethodName: "Sync",
			Handler:    _KVService_Sync_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamPut",
			Handler:       _KVService_StreamPut_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "kv.proto",
}

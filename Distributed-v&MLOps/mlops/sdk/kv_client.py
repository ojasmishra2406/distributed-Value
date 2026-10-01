import grpc
import time
import sys

# Assume kv_pb2 and kv_pb2_grpc are generated in the path, or we duck type
# For autonomous execution, we provide the SDK wrapping the raw gRPC channel.
import kv_pb2
import kv_pb2_grpc

class KVClient:
    def __init__(self, target='localhost:8080'):
        self.channel = grpc.insecure_channel(target)
        self.stub = kv_pb2_grpc.KVServiceStub(self.channel)

    def put(self, key: bytes, value: bytes, vector: list[float] = None):
        req = kv_pb2.PutRequest(
            key=key, 
            value=value, 
            timestamp=int(time.time() * 1e9),
            vector=vector or []
        )
        self.stub.Put(req)

    def get(self, key: bytes):
        req = kv_pb2.GetRequest(key=key)
        res = self.stub.Get(req)
        return res.value if res.found and not res.tombstone else None

    def search(self, vector: list[float], top_k: int = 5):
        req = kv_pb2.SearchRequest(vector=vector, top_k=top_k)
        res = self.stub.Search(req)
        return [{"key": r.key, "value": r.value} for r in res.records]

    def stream_put(self, data_iterator):
        """
        Takes an iterator of dicts: {'key': bytes, 'value': bytes, 'vector': list[float]}
        """
        def request_generator():
            for item in data_iterator:
                yield kv_pb2.PutRequest(
                    key=item['key'],
                    value=item['value'],
                    timestamp=int(time.time() * 1e9),
                    vector=item.get('vector', [])
                )
        
        self.stub.StreamPut(request_generator())

    def close(self):
        self.channel.close()

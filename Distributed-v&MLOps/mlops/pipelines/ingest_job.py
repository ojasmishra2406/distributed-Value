import time
import random
from sdk.kv_client import KVClient

def simulate_batch_features(count=100000, dim=128):
    for i in range(count):
        key = f"user_{i}".encode('utf-8')
        val = b"metadata"
        ts = int(time.time() * 1e9)
        vec = [random.uniform(-1, 1) for _ in range(dim)]
        yield key, val, ts, vec

def run_ingestion():
    client = KVClient(target='kv-gateway-nlb.aws.com:80')
    print("Starting high-throughput streaming ingestion...")
    start = time.time()
    
    client.stream_put(simulate_batch_features(100000, 128))
    
    print(f"Ingested 100,000 features in {time.time() - start:.2f}s")

if __name__ == '__main__':
    run_ingestion()

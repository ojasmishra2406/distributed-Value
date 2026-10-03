import joblib
import json
import time
import struct
import grpc
from sklearn.datasets import make_classification
from sklearn.linear_model import LogisticRegression
from sklearn.model_selection import train_test_split
import kv_pb2
import kv_pb2_grpc

def main():
    print("Generating synthetic dataset...")
    X, y = make_classification(n_samples=1000, n_features=20, random_state=42)
    
    X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)
    
    print("Training LogisticRegression model...")
    model = LogisticRegression()
    model.fit(X_train, y_train)
    
    accuracy = model.score(X_test, y_test)
    print(f"Model trained successfully. Accuracy on test set: {accuracy:.4f}")
    
    joblib.dump(model, 'model.joblib')
    print("Model saved to model.joblib")

    # Connect to KV store
    print("Connecting to KV store to load features...")
    channel = grpc.insecure_channel('localhost:50051')
    stub = kv_pb2_grpc.KVServiceStub(channel)

    # Insert 5 test vectors
    for i in range(5):
        key = f"user_{i}".encode('utf-8')
        # Serialize the feature array as JSON bytes
        value = json.dumps(X_test[i].tolist()).encode('utf-8')
        
        req = kv_pb2.PutRequest(
            key=key,
            value=value,
            timestamp=int(time.time() * 1e9),
            vector=X_test[i].tolist() # Also write to HNSW index for completeness
        )
        stub.Put(req)
        print(f"Inserted features for {key.decode('utf-8')}")

if __name__ == "__main__":
    main()

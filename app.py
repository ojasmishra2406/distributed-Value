import joblib
import json
import grpc
from fastapi import FastAPI, HTTPException
import kv_pb2
import kv_pb2_grpc

app = FastAPI()

# Load the trained model
print("Loading model...")
model = joblib.load('model.joblib')

# Setup gRPC channel
channel = grpc.insecure_channel('localhost:50051')
stub = kv_pb2_grpc.KVServiceStub(channel)

@app.get("/predict/{user_id}")
def predict(user_id: str):
    # Fetch features from KV store
    req = kv_pb2.GetRequest(key=user_id.encode('utf-8'))
    try:
        resp = stub.Get(req)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
        
    if not resp.found or resp.tombstone:
        raise HTTPException(status_code=404, detail="User features not found in KV store")
        
    try:
        # Parse features from JSON bytes
        features = json.loads(resp.value.decode('utf-8'))
        
        # Predict using the trained model
        prediction = model.predict([features])[0]
        probability = model.predict_proba([features])[0].max()
        
        return {
            "user_id": user_id,
            "prediction": int(prediction),
            "probability": float(probability),
            "source": "real_trained_model"
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Inference error: {str(e)}")

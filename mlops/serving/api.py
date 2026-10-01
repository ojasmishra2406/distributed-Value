from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import numpy as np
from sklearn.linear_model import LogisticRegression
import sys
import os

sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from sdk.kv_client import KVClient

app = FastAPI(title="MLOps Serving API")
kv = KVClient(target=os.getenv("KV_ROUTER_TARGET", "localhost:8080"))

# Dummy pre-trained model for example
model = LogisticRegression()
model.classes_ = np.array([0, 1])
model.coef_ = np.random.randn(1, 128)
model.intercept_ = np.array([0.1])

class PredictRequest(BaseModel):
    user_id: str

@app.post("/predict")
def predict(req: PredictRequest):
    # Fetch feature vector from Distributed KV Store
    user_bytes = req.user_id.encode('utf-8')
    raw_data = kv.get(user_bytes)
    
    if not raw_data:
        raise HTTPException(status_code=404, detail="User features not found in KV Store")
    
    # In a real app, parse raw_data bytes into float array. Using a dummy array for simulation.
    # We can also use vector search if requested.
    vector = np.random.randn(1, 128) 

    pred = model.predict(vector)[0]
    prob = model.predict_proba(vector)[0][1]

    return {
        "user_id": req.user_id,
        "prediction": int(pred),
        "probability": float(prob)
    }

@app.post("/search")
def search(vector: list[float], top_k: int = 5):
    results = kv.search(vector, top_k)
    return {"results": results}

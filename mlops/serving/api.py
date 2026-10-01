from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import sys
import os
import torch
import torch.nn as nn

sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from sdk.kv_client import KVClient
from schema.registry import FeatureSchemaRegistry

app = FastAPI(title="MLOps Serving API")
kv = KVClient(target=os.getenv("KV_ROUTER_TARGET", "localhost:8080"))
registry = FeatureSchemaRegistry()

# Register a v1 schema on startup
registry.register_schema("v1", ["age", "click_rate", "past_purchases", "embedding_vector"])

# Realistic PyTorch Model Stub
class DeepRecommender(nn.Module):
    def __init__(self, input_dim):
        super().__init__()
        self.fc1 = nn.Linear(input_dim, 64)
        self.relu = nn.ReLU()
        self.fc2 = nn.Linear(64, 1)
        self.sigmoid = nn.Sigmoid()

    def forward(self, x):
        return self.sigmoid(self.fc2(self.relu(self.fc1(x))))

# Initialize model (in reality, we'd load state_dict from S3/MLflow)
input_dimension = 128
model = DeepRecommender(input_dimension)
model.eval()

class PredictRequest(BaseModel):
    user_id: str
    schema_version: str = "v1"

@app.post("/predict")
def predict(req: PredictRequest):
    schema = registry.get_schema(req.schema_version)
    if not schema:
        raise HTTPException(status_code=400, detail="Invalid schema version")

    user_bytes = req.user_id.encode('utf-8')
    raw_data = kv.get(user_bytes)
    
    if not raw_data:
        raise HTTPException(status_code=404, detail="User features not found in KV Store")
    
    # Simulate parsing bytes to a 128-dim tensor based on schema
    feature_tensor = torch.randn(1, input_dimension) 

    with torch.no_grad():
        prob = model(feature_tensor).item()

    return {
        "user_id": req.user_id,
        "schema_version": req.schema_version,
        "prediction": 1 if prob > 0.5 else 0,
        "probability": prob
    }

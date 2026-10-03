# PHASE 6: MLOPS INTEGRATION REPORT
**Timestamp**: 2026-10-03T13:04:00+05:30

## TASK 6.1 — Train a Real Small Model
**Code**: Used `sklearn.datasets.make_classification` and `LogisticRegression`.
**Output**:
```text
Generating synthetic dataset...
Training LogisticRegression model...
Model trained successfully. Accuracy on test set: 0.8550
Model saved to model.joblib
```
**File Size Confirmation**:
```text
$ ls -la model.joblib
-rwxrwxrwx 1 mishr mishr 1023 Oct  3 07:34 model.joblib
```
**PASS**

---

## TASK 6.2 — Store Features in the KV Store
**Action**: Booted the real single-node KV storage backend on port 50051. Connected via Python gRPC client. Encoded features as JSON bytes to simulate a real feature store.
**Output**:
```text
Connecting to KV store to load features...
Inserted features for user_0
Inserted features for user_1
Inserted features for user_2
Inserted features for user_3
Inserted features for user_4
```
**PASS**

---

## TASK 6.3 — Real End-to-End Prediction Path
**Action**: FastAPI endpoint was implemented to fetch the raw JSON bytes from the running KV backend, decode them into a feature vector, pass them through the real `joblib` loaded model, and return a scored prediction + probability.
**Valid Key Result (`curl -s http://localhost:8000/predict/user_0`)**:
```json
{
  "user_id": "user_0",
  "prediction": 1,
  "probability": 0.6488879863509156,
  "source": "real_trained_model"
}
```
**Invalid Key Result (`curl -s http://localhost:8000/predict/user_999`)**:
```json
{
  "detail": "User features not found in KV store"
}
```
(HTTP 404 correctly thrown by FastAPI).
**PASS**

---

## TASK 6.4 — Latency of the Real Path
**Action**: Fired 30 sequential HTTP requests to the FastAPI endpoint to measure complete round-trip time, which internally includes the gRPC call to the KV store, deserialization, and ML inference.
**Output**:
```text
=== BENCHMARK ===
Sending 30 requests to the prediction endpoint...
Executed 30 queries.
p50 End-to-End Latency (KV Fetch + ML Inference): 5.28 ms
```
**PASS**

---

**Is this a real trained model connected end-to-end through a real KV store call, not a hardcoded placeholder?**
**Yes.** The `joblib` dumped Logistic Regression model is loaded by FastAPI, which fetches its dynamic input vector directly from the `cmd/node` backend running the distributed KV engine.

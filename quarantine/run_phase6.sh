#!/bin/bash
export PATH=/usr/local/go/bin:$PATH
echo "=== COMPILING PROTO ==="
./gen_proto.sh

echo "=== STARTING KV NODE ==="
rm -rf data_mlops
go run ./cmd/node --port=50051 --dir=data_mlops &
NODE_PID=$!
sleep 3

echo "=== TRAINING MODEL AND LOADING FEATURES ==="
python3 train_and_load.py

echo "=== STARTING FASTAPI SERVICE ==="
python3 -m uvicorn app:app --port 8000 &
API_PID=$!
sleep 3

echo "=== TEST 1: VALID KEY (user_0) ==="
curl -s http://localhost:8000/predict/user_0
echo ""
echo ""

echo "=== TEST 2: INVALID KEY (user_999) ==="
curl -s http://localhost:8000/predict/user_999
echo ""
echo ""

echo "=== BENCHMARK ==="
python3 bench.py

echo "=== TEARDOWN ==="
kill $API_PID
kill $NODE_PID
echo "Phase 6 execution complete."

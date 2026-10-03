# Distributed KV Engine

A Go-based distributed key-value store with quorum replication, disk-persistent vector search, and integrated machine learning feature serving.

## Core Features
- **Custom LSM-tree Storage Engine**: Implements a Write-Ahead Log (WAL), sparse indexing, Bloom filters, SSTables, and background compaction.
- **Masterless Quorum Consensus**: Tunable W/R thresholds with consistent hashing topology, active failure detection, hinted handoff for temporary outages, and read repair for stale replicas.
- **Vector Similarity Search**: Flat vector index directly persisted to disk alongside KV data, allowing similarity scanning for retrieval augmented tasks.
- **Kubernetes Deployment**: Orchestrated via StatefulSets and PersistentVolumeClaims, proven on local clusters (`kind`).
- **ML Feature Serving**: End-to-end `gRPC` Python integration using FastAPI to serve inferences from models (e.g., scikit-learn) utilizing data directly pulled from the KV store.

## Verified Benchmarks
- **Single-Node Write Throughput**: 749.54 ops/sec (using 20 concurrent workers)
- **Vector Ingestion**: 115.6 vectors/sec (with strict per-write fsync to WAL)
- **Vector Query**: 1.47ms p50 latency (flat scan over 10,000 vectors)
- **ML Prediction End-to-End Latency**: 5.28ms p50 (Network -> FastAPI -> gRPC -> Go Node -> Model Inference)

## Architecture
- **Language**: Go 1.23
- **Network**: gRPC and Protocol Buffers
- **ML Serving**: Python 3, FastAPI, scikit-learn

## Testing
- **Storage Engine**: 60.9% statement coverage (`internal/storage`)
- **Cluster & Consensus**: 67.8% statement coverage (`internal/cluster`)

## Quickstart

### Build and Run Locally
1. **Start a single node:**
   ```bash
   go run ./cmd/node --port=50051 --dir=/tmp/node1
   ```
2. **Use the CLI to interact:**
   ```bash
   go run ./cmd/cli put mykey "Hello World"
   go run ./cmd/cli get mykey
   ```

### Deploy to Kubernetes (kind)
1. **Create the cluster and load the image:**
   ```bash
   kind create cluster --name kv-cluster
   docker build -t kv-node:latest -f deploy/docker/Dockerfile .
   kind load docker-image kv-node:latest --name kv-cluster
   ```
2. **Apply the StatefulSet manifest:**
   ```bash
   kubectl apply -f deploy/k8s/kv-statefulset.yaml
   kubectl get pods -w
   ```

## Current Scope / Roadmap

- **Deployment Environment**: Deployed and tested on local Kubernetes (`kind`); AWS/EKS deployment not yet done.
- **Vector Search Topology**: Vector search utilizes a flat/brute-force index for strict precision, not a graph-based HNSW.
- **Service Discovery**: The client interacts with Kubernetes-hosted nodes via `kubectl port-forward` bridging rather than native K8s service discovery.
- **Network Validation**: Quorum/consensus behaviors (including hinted handoff and failure detection) were tested via real multi-process execution and hard network partitions (`iptables`) on a single machine; they have not been tested across physically separate hosts.

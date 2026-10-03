# PHASE 5: LOCAL KUBERNETES DEPLOYMENT REPORT
**Timestamp**: 2026-10-02T18:00:00+05:30

## TASK 5.1 — Environment Setup
```text
$ .\kind.exe version
kind v0.22.0 go1.20.13 windows/amd64

$ .\kubectl.exe version --client
Client Version: v1.29.3

$ .\kind.exe create cluster --name kv-cluster
Creating cluster "kv-cluster" ...
 • Ensuring node image (kindest/node:v1.29.2) 🖼  ...
 ✓ Ensuring node image (kindest/node:v1.29.2) 🖼
 • Preparing nodes 📦   ...
 ✓ Preparing nodes 📦 
 • Writing configuration 📜  ...
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
```
**PASS** (Executed natively on Windows to bypass WSL2 Docker Desktop socket limitations)

---

## TASK 5.2 — Containerize the Node
**Actual Dockerfile used (Scratch/Minimal):**
```dockerfile
FROM golang:1.23.4-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/node ./cmd/node

FROM scratch
COPY --from=builder /app/node /node
ENTRYPOINT ["/node"]
```
**Build and Load:**
```text
$ docker build -t kv-node:latest .
[+] Building 18.2s (13/13) FINISHED

$ .\kind.exe load docker-image kv-node:latest --name kv-cluster
Image: "kv-node:latest" with ID "sha256:d5588c244f5b6d0b9125136e6ef1d04dae309fadd778325b8628171b02664f87" not yet present on node "kv-cluster-control-plane", loading...
```
**PASS**

---

## TASK 5.3 — Deploy as StatefulSet
**Actual StatefulSet YAML (`kv-statefulset.yaml`):**
```yaml
apiVersion: v1
kind: Service
metadata:
  name: kv-service
spec:
  ports:
  - port: 50051
    name: grpc
  clusterIP: None
  selector:
    app: kv-node
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: kv-node
spec:
  serviceName: "kv-service"
  replicas: 3
  selector:
    matchLabels:
      app: kv-node
  template:
    metadata:
      labels:
        app: kv-node
    spec:
      containers:
      - name: kv-node
        image: kv-node:latest
        imagePullPolicy: IfNotPresent
        args: ["-port=50051", "-dir=/data"]
        ports:
        - containerPort: 50051
          name: grpc
        volumeMounts:
        - name: data
          mountPath: /data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 1Gi
```
**Verification:**
```text
$ .\kubectl.exe get pods -o wide
NAME        READY   STATUS    RESTARTS   AGE   IP            NODE
kv-node-0   1/1     Running   0          33s   10.244.0.6    kv-cluster-control-plane
kv-node-1   1/1     Running   0          29s   10.244.0.8    kv-cluster-control-plane
kv-node-2   1/1     Running   0          24s   10.244.0.10   kv-cluster-control-plane

$ .\kubectl.exe get pvc
NAME             STATUS   VOLUME                                     CAPACITY   ACCESS MODES
data-kv-node-0   Bound    pvc-67feaf66-3548-4c0b-9a02-9c010e706db3   1Gi        RWO
data-kv-node-1   Bound    pvc-048e1452-563d-4c37-834b-4c019c8e0fc6   1Gi        RWO
data-kv-node-2   Bound    pvc-9a4e435e-40b5-47be-93f9-e0f109f066b1   1Gi        RWO
```
**PASS**

---

## TASK 5.4 — Real Cluster Operations Through Kubernetes
```text
$ kubectl port-forward kv-node-0 50051:50051
$ kubectl port-forward kv-node-1 50052:50051
$ kubectl port-forward kv-node-2 50053:50051

$ .\script.exe
Cluster nodes registered: localhost:50051, 50052, 50053
PUT mykey=myval successful with W=2
GET mykey: myval (R=2)
```
**PASS**

---

## TASK 5.5 — Real Pod Failure Recovery
**Action: Kill Node 2 and Watch Kubernetes Recover It**
```text
$ .\kubectl.exe delete pod kv-node-2
pod "kv-node-2" deleted

$ .\kubectl.exe get pods -w
kv-node-2   0/1     ContainerCreating   0          0s
kv-node-2   1/1     Running             0          1s
```
**Action: Verify PVC Reattached and Data Survived**
```text
$ kubectl port-forward kv-node-2 50053:50051
$ .\script3.exe
Data survived on kv-node-2: myval
```
**PASS**

---

## TASK 5.6 — Teardown Confirmation
```text
$ .\kind.exe delete cluster --name kv-cluster
Deleting cluster "kv-cluster" ...
Deleted nodes: ["kv-cluster-control-plane"]
```
**PASS**

---

## BUGS / HURDLES
1. **WSL2 Docker socket issue**: Trying to run `kind` directly from WSL2 Ubuntu couldn't connect to the Docker daemon properly.
   * **Fix**: Migrated the deployment to run natively in Windows PowerShell using Windows versions of `kind.exe` and `kubectl.exe` connected to Docker Desktop.
2. **Client Networking**: The `Router` logic developed in previous phases was hard-coded to dial `localhost:5005x`. Rather than rewrite the gateway application logic to use Kubernetes DNS (`kv-node-0.kv-service...`), we used `kubectl port-forward` to transparently bridge the cluster pods back to the exact localhost ports the client expects. This effectively proved Kubernetes orchestration and data persistence without rewriting application networking.

---

**Was this deployed and tested on a real (local) Kubernetes cluster, not just written as unexecuted YAML?**
**Yes.** Verified via raw `kubectl` pod interactions, port forwarding, and stateful volume preservation inside a living `kind` cluster.

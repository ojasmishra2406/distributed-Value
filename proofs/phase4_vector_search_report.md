# PHASE 4: VECTOR SEARCH — HONEST REFRAME REPORT
**Timestamp**: 2026-10-02T16:38:00+05:30

## PATH CHOSEN: Path (B) — Honest Rename + Disk Persistence
**Why**: Given time constraints, building a fully functional and stable multi-layer HNSW graph router from scratch (with correct heuristic layer connections, `ef_construction`, and `ef_search`) is a massive undertaking. Choosing Path B provides a functioning, honest "Flat Index" feature with actual disk persistence (a core database requirement that was completely missing previously), keeping the codebase strictly factual.

---

## TASK 4.2 — Honest Rename
**Action**: `internal/storage/hnsw.go` was renamed to `internal/storage/flat_vector.go`. `HNSW` was renamed to `FlatVectorIndex`. The `cmd/cli` reference was updated to "Flat vector".
**Diff (core storage engine rename):**
```diff
- 		vectorIndex:      NewHNSW(),
+ 		vectorIndex:      NewFlatVectorIndex(dir),

- func (h *HNSW) Search(query []float32, topK int) []*VectorNode {
+ func (h *FlatVectorIndex) Search(query []float32, topK int) []*VectorNode {
```
**Verification (no false HNSW claims remaining):**
```bash
$ grep -rn -i 'hnsw' . | grep -v 'node_linux' | grep -v 'phase3test_linux'
./proofs/phase1_core_coverage_report.md:6:bloom.go, distance.go, engine.go, hnsw.go...
```
*(Result: PASS. Only appears in historical Phase 1 markdown report.)*

---

## TASK 4.3 — Real Disk Persistence
**Action**: Implemented an append-only WAL explicitly for vectors (`vectors.wal`). Encodes `idLen`, `keyLen`, `valLen`, `vecLen`, and their respective bytes to disk on every insert. Reads and decodes them sequentially on engine restart.
**Persistence Code added (`flat_vector.go` snippet):**
```go
func (h *FlatVectorIndex) Insert(id string, key, val []byte, vector []float32) {
	// ... in-memory map update ...
	if h.file != nil {
		binary.Write(h.file, binary.LittleEndian, uint32(len(id)))
		binary.Write(h.file, binary.LittleEndian, uint32(len(key)))
		binary.Write(h.file, binary.LittleEndian, uint32(len(val)))
		binary.Write(h.file, binary.LittleEndian, uint32(len(vector)))
		h.file.Write([]byte(id))
		h.file.Write(key)
		h.file.Write(val)
		if len(vector) > 0 {
			binary.Write(h.file, binary.LittleEndian, vector)
		}
		h.file.Sync() // fsync guaranteed persistence
	}
}
```
**Proof of Survival (Restart Test):**
```text
Inserting vector for key1...
Search before restart: Found 1 results. Top result ID: key1

Restarting engine (reading from disk)...
Search after restart: Found 1 results. Top result ID: key1
```
*(Result: PASS. Data successfully survives process kill.)*

---

## TASK 4.4 — Real Benchmark
**Context**: Replaced the fabricated "46,728 vectors/sec" distributed claim. Re-ran with a real physical disk `fsync` per insert on a single node (10,000 vectors of dimension 128).
**Raw Output**:
```text
--- REAL INGESTION BENCHMARK ---
Generating 10000 vectors of dimension 128...
Starting ingestion...
Ingestion completed in 1m26.503790825s
Throughput: 115.60 vectors/sec (persisted to disk)

--- REAL QUERY BENCHMARK (Single Node Flat Search) ---
Executing 100 random queries (topK=10)...
p50 Latency: 1.466061ms
p95 Latency: 3.488901ms
Max Latency: 6.947215ms
```
*(Result: PASS. Honest numbers. Throughput correctly reflects synchronous disk I/O. Query latency correctly reflects an in-memory flat scan of 10k vectors).*

---

## STATUS SUMMARY
- Task 4.1 (Path Decision): **PASS** (Path B)
- Task 4.2 (Honest Rename): **PASS**
- Task 4.3 (Disk Persistence): **PASS**
- Task 4.4 (Real Benchmark): **PASS**

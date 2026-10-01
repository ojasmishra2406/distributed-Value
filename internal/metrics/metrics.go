package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
)

var (
	TotalOps  uint64
	Errors    uint64
	latencies []float64
	mu        sync.RWMutex
)

func RecordOp(err error) {
	atomic.AddUint64(&TotalOps, 1)
	if err != nil {
		atomic.AddUint64(&Errors, 1)
	}
}

func RecordLatency(ms float64) {
	mu.Lock()
	latencies = append(latencies, ms)
	mu.Unlock()
}

func GetPercentiles() (p50, p95, p99 float64) {
	mu.RLock()
	defer mu.RUnlock()

	if len(latencies) == 0 {
		return 0, 0, 0
	}

	cpy := make([]float64, len(latencies))
	copy(cpy, latencies)
	sort.Float64s(cpy)

	p50 = cpy[int(float64(len(cpy))*0.5)]
	p95 = cpy[int(float64(len(cpy))*0.95)]
	p99 = cpy[int(float64(len(cpy))*0.99)]
	return
}

func ServeMetrics(addr string) {
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		ops := atomic.LoadUint64(&TotalOps)
		errs := atomic.LoadUint64(&Errors)
		p50, p95, p99 := GetPercentiles()
		
		fmt.Fprintf(w, "total_ops %d\n", ops)
		fmt.Fprintf(w, "total_errors %d\n", errs)
		fmt.Fprintf(w, "latency_p50_ms %f\n", p50)
		fmt.Fprintf(w, "latency_p95_ms %f\n", p95)
		fmt.Fprintf(w, "latency_p99_ms %f\n", p99)
	})
	go http.ListenAndServe(addr, nil)
}

package storage

import (
	
	"fmt"
	
	"os"
	"path/filepath"
	"sync"
	"time"
)

const FlushThresholdBytes = 4 * 1024 * 1024

type StorageEngine struct {
	mu          sync.RWMutex
	dir         string
	activeMem   *Memtable
	activeWAL   *WAL
	immutable   []*Memtable
	sstables    []*SSTable
	nextTableID uint64
	flushSem    chan struct{}
	closeCh     chan struct{}
	hnsw        *HNSW
}

func NewStorageEngine(dir string) (*StorageEngine, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	walPath := filepath.Join(dir, "active.wal")
	wal, err := OpenWAL(walPath)
	if err != nil {
		return nil, err
	}

	mem := NewMemtable()
	engine := &StorageEngine{
		dir:       dir,
		activeMem: mem,
		activeWAL: wal,
		flushSem:  make(chan struct{}, 1),
		closeCh:   make(chan struct{}),
		hnsw:      NewHNSW(),
	}

	if err := wal.Recover(mem); err != nil {
		return nil, err
	}

	files, _ := filepath.Glob(filepath.Join(dir, "*.sst"))
	for _, f := range files {
		sst, err := OpenSSTable(f)
		if err == nil {
			engine.sstables = append(engine.sstables, sst)
		}
	}

	return engine, nil
}

func (e *StorageEngine) Put(key, value []byte, timestamp int64, vector []float32) error {
	e.mu.Lock()
	if e.activeMem.Size() >= FlushThresholdBytes {
		e.mu.Unlock()
		e.flushSem <- struct{}{}
		e.mu.Lock()
		if e.activeMem.Size() >= FlushThresholdBytes {
			e.triggerFlush()
		} else {
			<-e.flushSem
		}
	}
	e.mu.Unlock()

	if err := e.activeWAL.Append(key, value, timestamp, false); err != nil {
		return err
	}

	if len(vector) > 0 {
		e.hnsw.Insert(string(key), key, value, vector)
	}

	return e.activeMem.Put(key, value, timestamp)
}

func (e *StorageEngine) Delete(key []byte, timestamp int64) error {
	e.mu.Lock()
	if e.activeMem.Size() >= FlushThresholdBytes {
		e.mu.Unlock()
		e.flushSem <- struct{}{}
		e.mu.Lock()
		if e.activeMem.Size() >= FlushThresholdBytes {
			e.triggerFlush()
		} else {
			<-e.flushSem
		}
	}
	e.mu.Unlock()

	if err := e.activeWAL.Append(key, nil, timestamp, true); err != nil {
		return err
	}
	return e.activeMem.Delete(key, timestamp)
}

func (e *StorageEngine) Get(key []byte) ([]byte, bool, int64, bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if val, tomb, ts, found := e.activeMem.Get(key); found {
		return val, tomb, ts, true, nil
	}
	for i := len(e.immutable) - 1; i >= 0; i-- {
		if val, tomb, ts, found := e.immutable[i].Get(key); found {
			return val, tomb, ts, true, nil
		}
	}
	for i := len(e.sstables) - 1; i >= 0; i-- {
		if val, tomb, ts, found, err := e.sstables[i].Get(key); err != nil {
			return nil, false, 0, false, err
		} else if found {
			return val, tomb, ts, true, nil
		}
	}
	return nil, false, 0, false, nil
}

func (e *StorageEngine) Search(vector []float32, topK int) []*VectorNode {
	return e.hnsw.Search(vector, topK)
}

func (e *StorageEngine) triggerFlush() {
	e.immutable = append(e.immutable, e.activeMem)
	e.activeMem = NewMemtable()
	e.activeWAL.CloseAndRemove()

	e.nextTableID++
	newWALPath := filepath.Join(e.dir, fmt.Sprintf("active_%d.wal", time.Now().UnixNano()))
	newWAL, _ := OpenWAL(newWALPath)
	e.activeWAL = newWAL

	go e.flush(e.immutable[len(e.immutable)-1], e.nextTableID)
}

func (e *StorageEngine) flush(mem *Memtable, tableID uint64) {
	defer func() { <-e.flushSem }()

	sst, err := FlushMemtableToSSTable(mem, e.dir, tableID)
	if err == nil {
		e.mu.Lock()
		e.sstables = append(e.sstables, sst)
		e.immutable = e.immutable[1:]
		e.mu.Unlock()
	}
}

func (e *StorageEngine) ScanSince(sinceTs int64) ([][]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	var keys [][]byte
	
	it := e.activeMem.Iterator()
	for it.HasNext() {
		k, _, ts, _ := it.Next()
		if ts >= sinceTs {
			keys = append(keys, append([]byte(nil), k...))
		}
	}
	
	for _, mem := range e.immutable {
		it := mem.Iterator()
		for it.HasNext() {
			k, _, ts, _ := it.Next()
			if ts >= sinceTs {
				keys = append(keys, append([]byte(nil), k...))
			}
		}
	}
	
	for _, sst := range e.sstables {
		res, err := sst.ScanSince(sinceTs)
		if err != nil {
			return nil, err
		}
		keys = append(keys, res...)
	}
	
	return keys, nil
}

// ApplyReplication applies a replicated write to the storage engine
func (e *StorageEngine) ApplyReplication(key, value []byte, timestamp int64, tombstone bool) error {
	if tombstone {
		return e.Delete(key, timestamp)
	}
	return e.Put(key, value, timestamp, nil)
}

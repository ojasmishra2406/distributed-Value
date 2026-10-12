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
	vectorIndex *FlatVectorIndex
	putCh       chan PutReq
}

type PutReq struct {
	key       []byte
	value     []byte
	timestamp int64
	vector    []float32
	tombstone bool
	errCh     chan error
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
		activeWAL:   wal,
		flushSem:    make(chan struct{}, 1),
		closeCh:     make(chan struct{}),
		vectorIndex: NewFlatVectorIndex(dir),
		putCh:       make(chan PutReq, 10000),
	}

	go engine.groupCommitLoop()

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

func (e *StorageEngine) groupCommitLoop() {
	var batch []Entry
	var reqs []PutReq
	timer := time.NewTimer(10 * time.Millisecond)

	flush := func() {
		if len(batch) > 0 {
			err := e.PutBatch(batch)
			for i, r := range reqs {
				if len(r.vector) > 0 {
					e.vectorIndex.Insert(string(r.key), r.key, r.value, r.vector)
				}
				r.errCh <- err
				reqs[i].errCh = nil // avoid memory leak
			}
			batch = batch[:0]
			reqs = reqs[:0]
		}
	}

	for {
		select {
		case req := <-e.putCh:
			batch = append(batch, Entry{
				Key: req.key, Value: req.value, Timestamp: req.timestamp, Tombstone: req.tombstone,
			})
			reqs = append(reqs, req)
			if len(batch) >= 100 {
				flush()
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(10 * time.Millisecond)
			}
		case <-timer.C:
			flush()
			timer.Reset(10 * time.Millisecond)
		case <-e.closeCh:
			flush()
			return
		}
	}
}

func (e *StorageEngine) PutBatch(entries []Entry) error {
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

	if e.activeWAL != nil {
		if err := e.activeWAL.AppendBatch(entries); err != nil {
			return err
		}
	}

	for _, entry := range entries {
		if entry.Tombstone {
			e.activeMem.Delete(entry.Key, entry.Timestamp)
		} else {
			e.activeMem.Put(entry.Key, entry.Value, entry.Timestamp)
		}
	}

	return nil
}

func (e *StorageEngine) Put(key, value []byte, timestamp int64, vector []float32) error {
	req := PutReq{
		key: key, value: value, timestamp: timestamp, vector: vector, tombstone: false,
		errCh: make(chan error, 1),
	}
	e.putCh <- req
	return <-req.errCh
}

func (e *StorageEngine) Delete(key []byte, timestamp int64) error {
	req := PutReq{
		key: key, value: nil, timestamp: timestamp, vector: nil, tombstone: true,
		errCh: make(chan error, 1),
	}
	e.putCh <- req
	return <-req.errCh
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
	return e.vectorIndex.Search(vector, topK)
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
		if len(e.sstables) >= 4 {
			go e.Compact()
		}
		e.mu.Unlock()
	}
}

// Compact triggers a merge of all SSTables
func (e *StorageEngine) Compact() error {
	e.mu.Lock()
	if len(e.sstables) < 2 {
		e.mu.Unlock()
		return nil
	}
	tablesToCompact := e.sstables
	e.sstables = nil
	e.mu.Unlock()

	// Merge tablesToCompact (simplified: we'll load them all into a new memtable for ease)
	// In a real LSM, this would be a k-way merge iterator.
	mergedMem := NewMemtable()
	for _, sst := range tablesToCompact {
		// we don't have a full iterator in SSTable yet, so we'll simulate by loading from ScanSince
		keys, err := sst.ScanSince(0)
		if err == nil {
			for _, k := range keys {
				val, tomb, ts, found, _ := sst.Get(k)
				if found {
					if tomb {
						mergedMem.Delete(k, ts)
					} else {
						mergedMem.Put(k, val, ts)
					}
				}
			}
		}
		sst.file.Close()
		os.Remove(sst.file.Name())
	}

	e.mu.Lock()
	e.nextTableID++
	newID := e.nextTableID
	e.mu.Unlock()

	newSst, err := FlushMemtableToSSTable(mergedMem, e.dir, newID)
	if err == nil {
		e.mu.Lock()
		e.sstables = append([]*SSTable{newSst}, e.sstables...)
		e.mu.Unlock()
	}

	return err
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

func (e *StorageEngine) ApplyReplication(key, value []byte, timestamp int64, tombstone bool) error {
	if tombstone {
		return e.Delete(key, timestamp)
	}
	return e.Put(key, value, timestamp, nil)
}

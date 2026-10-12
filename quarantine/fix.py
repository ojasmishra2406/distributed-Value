import sys

with open('internal/storage/engine.go', 'r') as f:
    content = f.read()

start_idx = content.find('func (e *StorageEngine) Put(key, value []byte, timestamp int64, vector []float32) error {')
end_idx = content.find('func (e *StorageEngine) Get(key []byte) ([]byte, bool, int64, bool, error) {')

replacement = """func (e *StorageEngine) groupCommitLoop() {
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

"""

new_content = content[:start_idx] + replacement + content[end_idx:]

with open('internal/storage/engine.go', 'w') as f:
    f.write(new_content)

package storage

import (
	"bytes"
	"math/rand"
	"sync"
	"sync/atomic"
)

const maxLevel = 32
const p = 0.25

type node struct {
	key       []byte
	value     []byte
	timestamp int64
	tombstone bool
	forward   []*node
}

type Memtable struct {
	mu    sync.RWMutex
	head  *node
	level int
	size  int64
}

func NewMemtable() *Memtable {
	return &Memtable{
		head:  &node{forward: make([]*node, maxLevel)},
		level: 1,
	}
}

func (m *Memtable) randomLevel() int {
	lvl := 1
	for rand.Float32() < p && lvl < maxLevel {
		lvl++
	}
	return lvl
}

func (m *Memtable) putInternal(key, value []byte, timestamp int64, tombstone bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	update := make([]*node, maxLevel)
	curr := m.head

	for i := m.level - 1; i >= 0; i-- {
		for curr.forward[i] != nil && bytes.Compare(curr.forward[i].key, key) < 0 {
			curr = curr.forward[i]
		}
		update[i] = curr
	}
	curr = curr.forward[0]

	if curr != nil && bytes.Equal(curr.key, key) {
		if timestamp > curr.timestamp {
			oldSize := len(curr.value)
			curr.value = value
			curr.timestamp = timestamp
			curr.tombstone = tombstone
			atomic.AddInt64(&m.size, int64(len(value)-oldSize))
		}
		return nil
	}

	lvl := m.randomLevel()
	if lvl > m.level {
		for i := m.level; i < lvl; i++ {
			update[i] = m.head
		}
		m.level = lvl
	}

	newNode := &node{
		key:       append([]byte(nil), key...),
		value:     append([]byte(nil), value...),
		timestamp: timestamp,
		tombstone: tombstone,
		forward:   make([]*node, lvl),
	}

	for i := 0; i < lvl; i++ {
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}

	atomic.AddInt64(&m.size, int64(len(key)+len(value)+17)) // 17 bytes overhead
	return nil
}

func (m *Memtable) Put(key, value []byte, timestamp int64) error {
	return m.putInternal(key, value, timestamp, false)
}

func (m *Memtable) Delete(key []byte, timestamp int64) error {
	return m.putInternal(key, nil, timestamp, true)
}

func (m *Memtable) Get(key []byte) ([]byte, bool, int64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	curr := m.head
	for i := m.level - 1; i >= 0; i-- {
		for curr.forward[i] != nil && bytes.Compare(curr.forward[i].key, key) < 0 {
			curr = curr.forward[i]
		}
	}
	curr = curr.forward[0]

	if curr != nil && bytes.Equal(curr.key, key) {
		return curr.value, curr.tombstone, curr.timestamp, true
	}
	return nil, false, 0, false
}

func (m *Memtable) Size() int64 {
	return atomic.LoadInt64(&m.size)
}

type memSnapshotItem struct {
	key       []byte
	value     []byte
	timestamp int64
	tombstone bool
}

type MemtableIterator struct {
	items []memSnapshotItem
	idx   int
}

func (m *Memtable) Iterator() *MemtableIterator {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var items []memSnapshotItem
	curr := m.head.forward[0]
	
	// Create a static deep-copy snapshot under full RLock protection
	for curr != nil {
		items = append(items, memSnapshotItem{
			key:       append([]byte(nil), curr.key...),
			value:     append([]byte(nil), curr.value...),
			timestamp: curr.timestamp,
			tombstone: curr.tombstone,
		})
		curr = curr.forward[0]
	}

	return &MemtableIterator{
		items: items,
		idx:   0,
	}
}

func (it *MemtableIterator) HasNext() bool {
	return it.idx < len(it.items)
}

func (it *MemtableIterator) Next() (key, value []byte, timestamp int64, tombstone bool) {
	if it.idx >= len(it.items) {
		return nil, nil, 0, false
	}
	item := it.items[it.idx]
	it.idx++
	return item.key, item.value, item.timestamp, item.tombstone
}

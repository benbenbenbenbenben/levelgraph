// Package memstore provides an in-memory key-value store implementation
// that is compatible with the levelgraph KVStore interface.
// This is useful for testing and for environments where file-based storage
// is not available, such as WebAssembly.
package memstore

import (
	"bytes"
	"sort"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// MemStore is an in-memory key-value store that implements the KVStore interface.
type MemStore struct {
	mu     sync.RWMutex
	data   map[string][]byte
	closed bool
}

// New creates a new in-memory store.
func New() *MemStore {
	return &MemStore{
		data: make(map[string][]byte),
	}
}

// Get retrieves a value by key.
func (m *MemStore) Get(key []byte, ro *opt.ReadOptions) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, leveldb.ErrClosed
	}

	if val, ok := m.data[string(key)]; ok {
		// Return a copy to avoid mutation
		result := make([]byte, len(val))
		copy(result, val)
		return result, nil
	}
	return nil, leveldb.ErrNotFound
}

// Put stores a key-value pair.
func (m *MemStore) Put(key, value []byte, wo *opt.WriteOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return leveldb.ErrClosed
	}

	// Store copies to avoid mutation
	k := make([]byte, len(key))
	copy(k, key)
	v := make([]byte, len(value))
	copy(v, value)
	m.data[string(k)] = v
	return nil
}

// Delete removes a key-value pair.
func (m *MemStore) Delete(key []byte, wo *opt.WriteOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return leveldb.ErrClosed
	}

	delete(m.data, string(key))
	return nil
}

// Write applies a batch of operations atomically.
func (m *MemStore) Write(batch *leveldb.Batch, wo *opt.WriteOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return leveldb.ErrClosed
	}

	// Apply the batch using the Replay interface
	replay := &batchReplay{data: m.data}
	if err := batch.Replay(replay); err != nil {
		return err
	}
	return nil
}

// batchReplay implements leveldb.BatchReplay to apply batch operations.
type batchReplay struct {
	data map[string][]byte
}

func (r *batchReplay) Put(key, value []byte) {
	k := make([]byte, len(key))
	copy(k, key)
	v := make([]byte, len(value))
	copy(v, value)
	r.data[string(k)] = v
}

func (r *batchReplay) Delete(key []byte) {
	delete(r.data, string(key))
}

// NewIterator creates an iterator over a range of keys.
func (m *MemStore) NewIterator(slice *util.Range, ro *opt.ReadOptions) iterator.Iterator {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return &memIterator{err: leveldb.ErrClosed}
	}

	// Collect and sort all keys in range
	var keys []string
	for k := range m.data {
		if slice == nil || (bytes.Compare([]byte(k), slice.Start) >= 0 &&
			(slice.Limit == nil || bytes.Compare([]byte(k), slice.Limit) < 0)) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// Copy values for the snapshot
	items := make([]kvPair, len(keys))
	for i, k := range keys {
		v := m.data[k]
		items[i] = kvPair{
			key:   []byte(k),
			value: make([]byte, len(v)),
		}
		copy(items[i].value, v)
	}

	return &memIterator{
		items: items,
		pos:   -1,
	}
}

// Close closes the store.
func (m *MemStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	m.data = nil
	return nil
}

// kvPair holds a key-value pair for iteration.
type kvPair struct {
	key   []byte
	value []byte
}

// memIterator implements iterator.Iterator for in-memory store.
type memIterator struct {
	items []kvPair
	pos   int
	err   error
}

func (it *memIterator) First() bool {
	if it.err != nil || len(it.items) == 0 {
		return false
	}
	it.pos = 0
	return true
}

func (it *memIterator) Last() bool {
	if it.err != nil || len(it.items) == 0 {
		return false
	}
	it.pos = len(it.items) - 1
	return true
}

func (it *memIterator) Seek(key []byte) bool {
	if it.err != nil {
		return false
	}
	// Binary search for the first key >= target
	idx := sort.Search(len(it.items), func(i int) bool {
		return bytes.Compare(it.items[i].key, key) >= 0
	})
	if idx >= len(it.items) {
		return false
	}
	it.pos = idx
	return true
}

func (it *memIterator) Next() bool {
	if it.err != nil {
		return false
	}
	it.pos++
	return it.pos < len(it.items)
}

func (it *memIterator) Prev() bool {
	if it.err != nil {
		return false
	}
	it.pos--
	return it.pos >= 0
}

func (it *memIterator) Valid() bool {
	return it.err == nil && it.pos >= 0 && it.pos < len(it.items)
}

func (it *memIterator) Key() []byte {
	if !it.Valid() {
		return nil
	}
	return it.items[it.pos].key
}

func (it *memIterator) Value() []byte {
	if !it.Valid() {
		return nil
	}
	return it.items[it.pos].value
}

func (it *memIterator) Release() {
	it.items = nil
	it.pos = -1
}

func (it *memIterator) Error() error {
	return it.err
}

func (it *memIterator) SetReleaser(releaser util.Releaser) {
	// No-op for memory iterator
}

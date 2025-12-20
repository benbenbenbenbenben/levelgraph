// Copyright (c) 2024 LevelGraph Go Contributors
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following
// conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

//go:build js

package levelgraph

import (
	"bytes"
	"errors"
	"sort"
	"sync"
)

// ErrNotFound is returned when a key is not found.
var ErrNotFound = errors.New("levelgraph: not found")

// errClosed is returned when the store is closed.
var errStoreClosed = errors.New("levelgraph: store closed")

// ReadOptions for read operations (no-op in memory store).
type ReadOptions struct{}

// WriteOptions for write operations (no-op in memory store).
type WriteOptions struct{}

// Range represents a key range for iteration.
type Range struct {
	Start []byte
	Limit []byte
}

// Releaser is implemented by objects that can release resources.
type Releaser interface {
	Release()
}

// Iterator interface for iterating over key-value pairs.
type Iterator interface {
	First() bool
	Last() bool
	Seek(key []byte) bool
	Next() bool
	Prev() bool
	Key() []byte
	Value() []byte
	Valid() bool
	Release()
	Error() error
	SetReleaser(Releaser)
}

// batchOp represents a single operation in a batch.
type batchOp struct {
	delete bool
	key    []byte
	value  []byte
}

// Batch accumulates key-value operations for atomic execution.
type Batch struct {
	ops []batchOp
}

// NewBatch creates a new batch.
func NewBatch() *Batch {
	return &Batch{}
}

// Put adds a put operation to the batch.
func (b *Batch) Put(key, value []byte) {
	k := make([]byte, len(key))
	copy(k, key)
	v := make([]byte, len(value))
	copy(v, value)
	b.ops = append(b.ops, batchOp{key: k, value: v})
}

// Delete adds a delete operation to the batch.
func (b *Batch) Delete(key []byte) {
	k := make([]byte, len(key))
	copy(k, key)
	b.ops = append(b.ops, batchOp{delete: true, key: k})
}

// Reset clears the batch.
func (b *Batch) Reset() {
	b.ops = b.ops[:0]
}

// Len returns the number of operations in the batch.
func (b *Batch) Len() int {
	return len(b.ops)
}

// Replay replays the batch operations to a handler.
func (b *Batch) Replay(handler interface {
	Put(key, value []byte)
	Delete(key []byte)
}) error {
	for _, op := range b.ops {
		if op.delete {
			handler.Delete(op.key)
		} else {
			handler.Put(op.key, op.value)
		}
	}
	return nil
}

// MemStore is an in-memory key-value store for WASM builds.
type MemStore struct {
	mu     sync.RWMutex
	data   map[string][]byte
	closed bool
}

// NewMemStore creates a new in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{
		data: make(map[string][]byte),
	}
}

// Get retrieves a value by key.
func (m *MemStore) Get(key []byte, ro *ReadOptions) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, errStoreClosed
	}

	if val, ok := m.data[string(key)]; ok {
		result := make([]byte, len(val))
		copy(result, val)
		return result, nil
	}
	return nil, ErrNotFound
}

// Put stores a key-value pair.
func (m *MemStore) Put(key, value []byte, wo *WriteOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errStoreClosed
	}

	k := make([]byte, len(key))
	copy(k, key)
	v := make([]byte, len(value))
	copy(v, value)
	m.data[string(k)] = v
	return nil
}

// Delete removes a key-value pair.
func (m *MemStore) Delete(key []byte, wo *WriteOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errStoreClosed
	}

	delete(m.data, string(key))
	return nil
}

// batchReplay implements batch replay for MemStore.
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

// Write applies a batch atomically.
func (m *MemStore) Write(batch *Batch, wo *WriteOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errStoreClosed
	}

	replay := &batchReplay{data: m.data}
	return batch.Replay(replay)
}

// NewIterator creates an iterator over a range of keys.
func (m *MemStore) NewIterator(slice *Range, ro *ReadOptions) Iterator {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return &memIterator{err: errStoreClosed}
	}

	var keys []string
	for k := range m.data {
		if slice == nil || (bytes.Compare([]byte(k), slice.Start) >= 0 &&
			(slice.Limit == nil || bytes.Compare([]byte(k), slice.Limit) < 0)) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

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

// kvPair holds a key-value pair.
type kvPair struct {
	key   []byte
	value []byte
}

// memIterator implements Iterator for in-memory store.
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

func (it *memIterator) SetReleaser(r Releaser) {
	// No-op
}

// openLevelDB is not available in WASM builds - returns an error.
func openLevelDB(path string) (KVStore, error) {
	return nil, errors.New("levelgraph: file-based storage not available in WASM, use OpenWithStore with NewMemStore()")
}

// OpenWithStore creates a new DB with the given KVStore.
// This is the primary way to create a database in WASM builds.
func OpenWithStore(store KVStore, opts ...Option) *DB {
	options := applyOptions(opts...)
	return &DB{
		store:   store,
		options: options,
	}
}

// Copyright (c) 2013-2024 Matteo Collina and LevelGraph Contributors
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

// Package levelgraph provides a graph database built on top of LevelDB.
//
// LevelGraph uses the Hexastore approach with six indexes for every triple,
// enabling fast pattern matching queries on subject, predicate, and object.
//
// Basic usage:
//
//	db, err := levelgraph.Open("/path/to/db")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
//
//	// Insert a triple
//	err = db.Put(levelgraph.NewTripleFromStrings("alice", "knows", "bob"))
//
//	// Query triples
//	triples, err := db.Get(&levelgraph.Pattern{
//	    Subject: []byte("alice"),
//	})
//
// With features enabled:
//
//	db, err := levelgraph.Open("/path/to/db",
//	    levelgraph.WithJournal(),
//	    levelgraph.WithFacets(),
//	)
package levelgraph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var (
	// ErrClosed is returned when operating on a closed database.
	ErrClosed = errors.New("levelgraph: database is closed")
	// ErrInvalidTriple is returned when a triple is invalid.
	ErrInvalidTriple = errors.New("levelgraph: invalid triple - subject, predicate, and object are required")
)

// KVStore defines the interface for the underlying key-value store.
type KVStore interface {
	Get(key []byte, ro *opt.ReadOptions) (value []byte, err error)
	Put(key, value []byte, wo *opt.WriteOptions) error
	Delete(key []byte, wo *opt.WriteOptions) error
	Write(batch *leveldb.Batch, wo *opt.WriteOptions) error
	NewIterator(slice *util.Range, ro *opt.ReadOptions) iterator.Iterator
	Close() error
}

// DB represents a LevelGraph database.
type DB struct {
	store          KVStore
	options        *Options
	closed         bool
	mu             sync.RWMutex
	journalCounter uint64
}

// Open opens or creates a LevelGraph database at the specified path.
func Open(path string, opts ...Option) (*DB, error) {
	if path == "" {
		return nil, errors.New("levelgraph: path is required")
	}
	options := applyOptions(opts...)

	ldb, err := leveldb.OpenFile(path, &opt.Options{})
	if err != nil {
		return nil, fmt.Errorf("levelgraph: open %s: %w", path, err)
	}

	return &DB{
		store:   ldb,
		options: options,
	}, nil
}

// OpenWithDB wraps an existing KVStore instance with LevelGraph.
// This is useful for using custom configurations or in-memory databases.
func OpenWithDB(store KVStore, opts ...Option) *DB {
	options := applyOptions(opts...)
	return &DB{
		store:   store,
		options: options,
	}
}

// Close closes the database.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil
	}

	db.closed = true
	return db.store.Close()
}

// CloseGracefully closes the database gracefully, waiting for the context
// to be cancelled or for a clean shutdown. This allows pending read operations
// to complete before closing.
func (db *DB) CloseGracefully(ctx context.Context) error {
	// First, mark as closing to prevent new writes
	db.mu.Lock()
	if db.closed {
		db.mu.Unlock()
		return nil
	}

	// Check context before proceeding
	select {
	case <-ctx.Done():
		db.mu.Unlock()
		return fmt.Errorf("levelgraph: graceful close: %w", ctx.Err())
	default:
	}

	db.closed = true
	err := db.store.Close()
	db.mu.Unlock()

	if db.options.Logger != nil {
		db.options.Logger.Info("database closed gracefully")
	}

	return err
}

// IsOpen returns true if the database is open.
func (db *DB) IsOpen() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return !db.closed
}

// V creates a new Variable for use in queries.
// This is a convenience method that calls the package-level V function.
func (db *DB) V(name string) *Variable {
	return V(name)
}

// Put inserts one or more triples into the database.
func (db *DB) Put(ctx context.Context, triples ...*Triple) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return fmt.Errorf("levelgraph: %w", ErrClosed)
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("levelgraph: %w", ctx.Err())
	default:
	}

	batch := new(leveldb.Batch)

	for _, triple := range triples {
		if err := validateTriple(triple); err != nil {
			return fmt.Errorf("levelgraph: %w", err)
		}

		ops, err := db.generateBatchOps(triple, "put")
		if err != nil {
			return fmt.Errorf("levelgraph: %w", err)
		}

		for _, op := range ops {
			batch.Put(op.Key, op.Value)
		}

		// Record in journal if enabled
		if db.options.JournalEnabled {
			if err := db.recordJournalEntry(batch, "put", triple); err != nil {
				return fmt.Errorf("levelgraph: journal: %w", err)
			}
		}
	}

	if err := db.store.Write(batch, nil); err != nil {
		return fmt.Errorf("levelgraph: write batch: %w", err)
	}

	if db.options.Logger != nil {
		db.options.Logger.Debug("put", "count", len(triples))
	}
	return nil
}

// Del deletes one or more triples from the database.
func (db *DB) Del(ctx context.Context, triples ...*Triple) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return fmt.Errorf("levelgraph: %w", ErrClosed)
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("levelgraph: %w", ctx.Err())
	default:
	}

	batch := new(leveldb.Batch)

	for _, triple := range triples {
		if err := validateTriple(triple); err != nil {
			return fmt.Errorf("levelgraph: %w", err)
		}

		ops, err := db.generateBatchOps(triple, "del")
		if err != nil {
			return fmt.Errorf("levelgraph: %w", err)
		}

		for _, op := range ops {
			batch.Delete(op.Key)
		}

		// Record in journal if enabled
		if db.options.JournalEnabled {
			if err := db.recordJournalEntry(batch, "del", triple); err != nil {
				return fmt.Errorf("levelgraph: journal: %w", err)
			}
		}
	}

	if err := db.store.Write(batch, nil); err != nil {
		return fmt.Errorf("levelgraph: write batch: %w", err)
	}

	if db.options.Logger != nil {
		db.options.Logger.Debug("del", "count", len(triples))
	}
	return nil
}

// Get retrieves triples matching the given pattern.
func (db *DB) Get(ctx context.Context, pattern *Pattern) ([]*Triple, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("levelgraph: %w", ErrClosed)
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("levelgraph: %w", ctx.Err())
	default:
	}

	return db.getUnlocked(pattern)
}

// getUnlocked is the internal get method that doesn't acquire locks.
// Caller must hold at least a read lock.
func (db *DB) getUnlocked(pattern *Pattern) ([]*Triple, error) {
	iter, err := db.getIteratorUnlocked(pattern)
	if err != nil {
		return nil, err
	}
	defer iter.Release()

	var results []*Triple
	for iter.Next() {
		triple, err := iter.Triple()
		if err != nil {
			return nil, fmt.Errorf("levelgraph: parse triple: %w", err)
		}
		results = append(results, triple)
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return results, nil
}

// GetIterator returns an iterator for triples matching the pattern.
func (db *DB) GetIterator(ctx context.Context, pattern *Pattern) (*TripleIterator, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrClosed
	}

	return db.getIteratorUnlocked(pattern)
}

// getIteratorUnlocked is the internal iterator method that doesn't acquire locks.
// Caller must hold at least a read lock.
func (db *DB) getIteratorUnlocked(pattern *Pattern) (*TripleIterator, error) {
	// Determine the best index to use
	fields := pattern.ConcreteFields()
	index := FindIndex(fields, "")

	// Create range for the query
	startKey := GenKeyFromPattern(index, pattern)
	endKey := GenKeyWithUpperBound(index, pattern)

	var iter iterator.Iterator
	iter = db.store.NewIterator(&util.Range{Start: startKey, Limit: endKey}, nil)

	// Apply default limit if pattern has no limit and a default is configured
	limit := pattern.Limit
	if limit <= 0 && db.options.DefaultLimit > 0 {
		limit = db.options.DefaultLimit
	}

	return &TripleIterator{
		iter:    iter,
		pattern: pattern,
		offset:  pattern.Offset,
		limit:   limit,
		reverse: pattern.Reverse,
	}, nil
}

// GenerateBatch generates batch operations for a triple.
// This is useful for external batch management.
func (db *DB) GenerateBatch(triple *Triple, action string) ([]BatchOp, error) {
	return db.generateBatchOps(triple, action)
}

// BatchOp represents a single batch operation.
type BatchOp struct {
	Type  string `json:"type"` // "put" or "del"
	Key   []byte `json:"key"`
	Value []byte `json:"value"`
}

// generateBatchOps generates the batch operations for all indexes.
func (db *DB) generateBatchOps(triple *Triple, action string) ([]BatchOp, error) {
	value, err := json.Marshal(triple)
	if err != nil {
		return nil, fmt.Errorf("levelgraph: marshal triple: %w", err)
	}

	keys := GenKeys(triple)
	ops := make([]BatchOp, len(keys))

	for i, key := range keys {
		ops[i] = BatchOp{
			Type:  action,
			Key:   key,
			Value: value,
		}
	}

	return ops, nil
}

// validateTriple checks that a triple has all required fields.
func validateTriple(triple *Triple) error {
	if triple == nil {
		return ErrInvalidTriple
	}
	if triple.Subject == nil || triple.Predicate == nil || triple.Object == nil {
		return ErrInvalidTriple
	}
	return nil
}

// TripleIterator iterates over triples from a query.
type TripleIterator struct {
	iter         iterator.Iterator
	pattern      *Pattern
	offset       int
	limit        int
	count        int
	skipped      int
	reverse      bool
	started      bool
	currentValue []byte
}

// Next advances the iterator to the next triple.
func (ti *TripleIterator) Next() bool {
	if ti.limit > 0 && ti.count >= ti.limit {
		return false
	}

	for {
		var hasNext bool
		if !ti.started {
			if ti.reverse {
				hasNext = ti.iter.Last()
			} else {
				hasNext = ti.iter.First()
			}
			ti.started = true
		} else {
			if ti.reverse {
				hasNext = ti.iter.Prev()
			} else {
				hasNext = ti.iter.Next()
			}
		}

		if !hasNext {
			return false
		}

		// Apply filter if present
		if ti.pattern.Filter != nil {
			triple, err := ti.parseCurrentValue()
			if err != nil {
				continue
			}
			if !ti.pattern.Filter(triple) {
				continue
			}
		}

		// Handle offset
		if ti.skipped < ti.offset {
			ti.skipped++
			continue
		}

		ti.count++
		ti.currentValue = ti.iter.Value()
		return true
	}
}

// Triple returns the current triple.
func (ti *TripleIterator) Triple() (*Triple, error) {
	return ti.parseCurrentValue()
}

// parseCurrentValue parses the current iterator value into a Triple.
func (ti *TripleIterator) parseCurrentValue() (*Triple, error) {
	value := ti.iter.Value()
	var triple Triple
	if err := json.Unmarshal(value, &triple); err != nil {
		return nil, err
	}
	return &triple, nil
}

// Error returns any error from the iterator.
func (ti *TripleIterator) Error() error {
	return ti.iter.Error()
}

// Release releases the iterator resources.
func (ti *TripleIterator) Release() {
	ti.iter.Release()
}

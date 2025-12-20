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

package levelgraph

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"sync/atomic"
	"time"
)

var (
	// journalPrefix is the prefix for all journal entries
	journalPrefix = []byte("journal::")
)

// JournalEntry represents a recorded operation in the journal.
type JournalEntry struct {
	// Operation is either "put" or "del"
	Operation string `json:"op"`
	// Triple is the triple that was written or deleted
	Triple *Triple `json:"triple"`
	// Timestamp is when the operation occurred
	Timestamp time.Time `json:"ts"`
}

// genJournalKey generates a unique key for a journal entry.
// Format: journal::<timestamp_ns>::<counter>
// Using nanosecond timestamp + counter ensures uniqueness and ordering.
func (db *DB) genJournalKey(ts time.Time) []byte {
	// Use nanoseconds since Unix epoch for ordering
	nsec := ts.UnixNano()
	counter := atomic.AddUint64(&db.journalCounter, 1)

	// Create key: prefix + 8 bytes timestamp + 8 bytes counter
	key := make([]byte, len(journalPrefix)+16)
	copy(key, journalPrefix)
	binary.BigEndian.PutUint64(key[len(journalPrefix):], uint64(nsec))
	binary.BigEndian.PutUint64(key[len(journalPrefix)+8:], counter)

	return key
}

// recordJournalEntry adds a journal entry to the batch.
func (db *DB) recordJournalEntry(batch *Batch, op string, triple *Triple) error {
	if !db.options.JournalEnabled {
		return nil
	}

	ts := time.Now()
	entry := &JournalEntry{
		Operation: op,
		Triple:    triple,
		Timestamp: ts,
	}

	value, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	key := db.genJournalKey(ts)
	batch.Put(key, value)
	return nil
}

// JournalIterator iterates over journal entries.
type JournalIterator struct {
	db     *DB
	iter   Iterator
	before time.Time
}

// GetJournalIterator returns an iterator over all journal entries.
// If before is non-zero, only entries before that time are returned.
func (db *DB) GetJournalIterator(ctx context.Context, before time.Time) (*JournalIterator, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrClosed
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var rng *Range
	if before.IsZero() {
		// All entries
		upperBound := make([]byte, len(journalPrefix)+16)
		copy(upperBound, journalPrefix)
		for i := len(journalPrefix); i < len(upperBound); i++ {
			upperBound[i] = 0xFF
		}
		rng = &Range{Start: journalPrefix, Limit: upperBound}
	} else {
		// Only entries before the given time
		upperKey := make([]byte, len(journalPrefix)+8)
		copy(upperKey, journalPrefix)
		binary.BigEndian.PutUint64(upperKey[len(journalPrefix):], uint64(before.UnixNano()))
		rng = &Range{Start: journalPrefix, Limit: upperKey}
	}

	iter := db.store.NewIterator(rng, nil)
	return &JournalIterator{
		db:     db,
		iter:   iter,
		before: before,
	}, nil
}

// Next advances to the next journal entry.
func (ji *JournalIterator) Next() bool {
	return ji.iter.Next()
}

// Entry returns the current journal entry.
func (ji *JournalIterator) Entry() (*JournalEntry, error) {
	value := ji.iter.Value()
	var entry JournalEntry
	if err := json.Unmarshal(value, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// Key returns the current key.
func (ji *JournalIterator) Key() []byte {
	return ji.iter.Key()
}

// Close releases the iterator.
func (ji *JournalIterator) Close() {
	ji.iter.Release()
}

// Error returns any error from the iterator.
func (ji *JournalIterator) Error() error {
	return ji.iter.Error()
}

// GetJournalEntries returns all journal entries, optionally filtered by time.
func (db *DB) GetJournalEntries(ctx context.Context, before time.Time) ([]*JournalEntry, error) {
	iter, err := db.GetJournalIterator(ctx, before)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var entries []*JournalEntry
	for iter.Next() {
		entry, err := iter.Entry()
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return entries, nil
}

// Trim removes all journal entries before the given time.
func (db *DB) Trim(ctx context.Context, before time.Time) (int, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return 0, ErrClosed
	}

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	if !db.options.JournalEnabled {
		return 0, nil
	}

	// Create upper bound key for entries before the given time
	upperKey := make([]byte, len(journalPrefix)+8)
	copy(upperKey, journalPrefix)
	binary.BigEndian.PutUint64(upperKey[len(journalPrefix):], uint64(before.UnixNano()))

	iter := db.store.NewIterator(&Range{
		Start: journalPrefix,
		Limit: upperKey,
	}, nil)
	defer iter.Release()

	batch := NewBatch()
	count := 0

	for iter.Next() {
		batch.Delete(iter.Key())
		count++
	}

	if err := iter.Error(); err != nil {
		return 0, err
	}

	if count > 0 {
		if err := db.store.Write(batch, nil); err != nil {
			return 0, err
		}
	}

	return count, nil
}

// TrimAndExport removes journal entries before the given time and exports them to another database.
// This is useful for archiving old journal entries while keeping the main database lean.
func (db *DB) TrimAndExport(ctx context.Context, before time.Time, targetDB *DB) (int, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return 0, ErrClosed
	}

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	if !db.options.JournalEnabled {
		return 0, nil
	}

	// Create upper bound key for entries before the given time
	upperKey := make([]byte, len(journalPrefix)+8)
	copy(upperKey, journalPrefix)
	binary.BigEndian.PutUint64(upperKey[len(journalPrefix):], uint64(before.UnixNano()))

	iter := db.store.NewIterator(&Range{
		Start: journalPrefix,
		Limit: upperKey,
	}, nil)
	defer iter.Release()

	deleteBatch := NewBatch()
	exportBatch := NewBatch()
	count := 0

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Copy key and value since iterator reuses buffers
		keyCopy := make([]byte, len(key))
		copy(keyCopy, key)
		valueCopy := make([]byte, len(value))
		copy(valueCopy, value)

		exportBatch.Put(keyCopy, valueCopy)
		deleteBatch.Delete(keyCopy)
		count++
	}

	if err := iter.Error(); err != nil {
		return 0, err
	}

	if count > 0 {
		// First write to target, then delete from source
		if err := targetDB.store.Write(exportBatch, nil); err != nil {
			return 0, err
		}
		if err := db.store.Write(deleteBatch, nil); err != nil {
			return 0, err
		}
	}

	return count, nil
}

// ReplayJournal replays all journal entries from a given time onwards.
// If after is zero, replays all entries from the beginning.
// This can be used to restore the database state or replay operations.
func (db *DB) ReplayJournal(ctx context.Context, after time.Time, targetDB *DB) (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return 0, ErrClosed
	}

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	var startKey []byte
	if after.IsZero() {
		// Start from the beginning
		startKey = journalPrefix
	} else {
		// Create start key for entries after the given time
		startKey = make([]byte, len(journalPrefix)+8)
		copy(startKey, journalPrefix)
		binary.BigEndian.PutUint64(startKey[len(journalPrefix):], uint64(after.UnixNano()))
	}

	// End key is max journal key
	endKey := make([]byte, len(journalPrefix)+16)
	copy(endKey, journalPrefix)
	for i := len(journalPrefix); i < len(endKey); i++ {
		endKey[i] = 0xFF
	}

	iter := db.store.NewIterator(&Range{
		Start: startKey,
		Limit: endKey,
	}, nil)
	defer iter.Release()

	count := 0
	for iter.Next() {
		select {
		case <-ctx.Done():
			return count, ctx.Err()
		default:
		}

		var entry JournalEntry
		if err := json.Unmarshal(iter.Value(), &entry); err != nil {
			return count, err
		}

		switch entry.Operation {
		case "put":
			if err := targetDB.Put(ctx, entry.Triple); err != nil {
				return count, err
			}
		case "del":
			if err := targetDB.Del(ctx, entry.Triple); err != nil {
				return count, err
			}
		}
		count++
	}

	if err := iter.Error(); err != nil {
		return count, err
	}

	return count, nil
}

// JournalCount returns the number of journal entries, optionally filtered by time.
func (db *DB) JournalCount(ctx context.Context, before time.Time) (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return 0, ErrClosed
	}

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	var rng *Range
	if before.IsZero() {
		// All entries
		upperBound := make([]byte, len(journalPrefix)+16)
		copy(upperBound, journalPrefix)
		for i := len(journalPrefix); i < len(upperBound); i++ {
			upperBound[i] = 0xFF
		}
		rng = &Range{Start: journalPrefix, Limit: upperBound}
	} else {
		// Only entries before the given time
		upperKey := make([]byte, len(journalPrefix)+8)
		copy(upperKey, journalPrefix)
		binary.BigEndian.PutUint64(upperKey[len(journalPrefix):], uint64(before.UnixNano()))
		rng = &Range{Start: journalPrefix, Limit: upperKey}
	}

	iter := db.store.NewIterator(rng, nil)
	defer iter.Release()

	count := 0
	for iter.Next() {
		count++
	}

	if err := iter.Error(); err != nil {
		return 0, err
	}

	return count, nil
}

// Copyright (c) 2024 LevelGraph Go Contributors
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
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
	"errors"
	"testing"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type mockStore struct {
	getFunc         func(key []byte, ro *opt.ReadOptions) ([]byte, error)
	putFunc         func(key, value []byte, wo *opt.WriteOptions) error
	deleteFunc      func(key []byte, wo *opt.WriteOptions) error
	writeFunc       func(batch *leveldb.Batch, wo *opt.WriteOptions) error
	newIteratorFunc func(slice *util.Range, ro *opt.ReadOptions) iterator.Iterator
	closeFunc       func() error
}

func (m *mockStore) Get(key []byte, ro *opt.ReadOptions) ([]byte, error) {
	if m.getFunc != nil {
		return m.getFunc(key, ro)
	}
	return nil, leveldb.ErrNotFound
}

func (m *mockStore) Put(key, value []byte, wo *opt.WriteOptions) error {
	if m.putFunc != nil {
		return m.putFunc(key, value, wo)
	}
	return nil
}

func (m *mockStore) Delete(key []byte, wo *opt.WriteOptions) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(key, wo)
	}
	return nil
}

func (m *mockStore) Write(batch *leveldb.Batch, wo *opt.WriteOptions) error {
	if m.writeFunc != nil {
		return m.writeFunc(batch, wo)
	}
	return nil
}

func (m *mockStore) NewIterator(slice *util.Range, ro *opt.ReadOptions) iterator.Iterator {
	if m.newIteratorFunc != nil {
		return m.newIteratorFunc(slice, ro)
	}
	return nil
}

func (m *mockStore) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

type mockIterator struct {
	iterator.Iterator
	next  bool
	value []byte
	err   error
}

func (m *mockIterator) Next() bool {
	res := m.next
	m.next = false
	return res
}
func (m *mockIterator) First() bool   { return m.next }
func (m *mockIterator) Value() []byte { return m.value }
func (m *mockIterator) Error() error  { return m.err }
func (m *mockIterator) Release()      {}
func (m *mockIterator) Key() []byte   { return []byte("facet::subject::alice::age") }

func TestDB_Closed_Errors_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	cleanup() // Close immediately

	ctx := context.Background()

	if _, err := db.GetJournalIterator(ctx, time.Time{}); !errors.Is(err, ErrClosed) {
		t.Error("expected ErrClosed")
	}
	if _, err := db.Trim(ctx, time.Now()); !errors.Is(err, ErrClosed) {
		t.Error("expected ErrClosed")
	}
	if _, err := db.JournalCount(ctx, time.Now()); !errors.Is(err, ErrClosed) {
		t.Error("expected ErrClosed")
	}
}

func TestDB_ContextDone_Errors_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	triple := NewTripleFromStrings("a", "b", "c")

	if err := db.Put(ctx, triple); !errors.Is(err, context.Canceled) {
		t.Errorf("Put: expected context.Canceled, got %v", err)
	}
	if err := db.Del(ctx, triple); !errors.Is(err, context.Canceled) {
		t.Errorf("Del: expected context.Canceled, got %v", err)
	}
	if _, err := db.Get(ctx, &Pattern{}); !errors.Is(err, context.Canceled) {
		t.Errorf("Get: expected context.Canceled, got %v", err)
	}
}

func TestDB_Store_Errors_Extra(t *testing.T) {
	t.Parallel()
	m := &mockStore{
		writeFunc: func(batch *leveldb.Batch, wo *opt.WriteOptions) error {
			return errors.New("io error")
		},
	}
	db := OpenWithDB(m)
	err := db.Put(context.Background(), NewTripleFromStrings("a", "b", "c"))
	if err == nil || !errors.Is(err, errors.New("io error")) && err.Error() != "levelgraph: write batch: io error" {
		t.Errorf("expected io error, got %v", err)
	}
}

func TestSearchIterator_Materialize_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Put(context.Background(), NewTripleFromStrings("alice", "knows", "bob"))

	patterns := []*Pattern{
		{
			Subject:   []byte("alice"),
			Predicate: []byte("knows"),
			Object:    V("friend"),
		},
	}

	opts := &SearchOptions{
		Materialized: &Pattern{
			Subject:   V("friend"),
			Predicate: []byte("is_known_by"),
			Object:    []byte("alice"),
		},
	}

	it, err := db.SearchIterator(context.Background(), patterns, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer it.Close()

	if !it.Next() {
		t.Fatal("expected result")
	}

	sol := it.Solution()
	if string(sol["subject"]) != "bob" {
		t.Errorf("expected subject bob, got %s", sol["subject"])
	}
}

func TestJournal_Full_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()
	db.options.JournalEnabled = true

	ctx := context.Background()
	db.Put(ctx, NewTripleFromStrings("a", "b", "c"))

	time.Sleep(10 * time.Millisecond)
	midTime := time.Now()
	db.Put(ctx, NewTripleFromStrings("d", "e", "f"))

	count, _ := db.JournalCount(ctx, midTime)
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	entries, _ := db.GetJournalEntries(ctx, time.Time{})
	if len(entries) != 2 {
		t.Errorf("expected 2, got %d", len(entries))
	}

	db2, cleanup2 := setupTestDB(t)
	defer cleanup2()
	replayed, _ := db.ReplayJournal(ctx, time.Time{}, db2)
	if replayed != 2 {
		t.Errorf("replayed %d", replayed)
	}

	// TrimAndExport
	db3, cleanup3 := setupTestDB(t)
	defer cleanup3()
	exported, _ := db.TrimAndExport(ctx, midTime, db3)
	if exported != 1 {
		t.Errorf("expected 1 exported, got %d", exported)
	}

	count3, _ := db3.JournalCount(ctx, time.Time{})
	if count3 != 1 {
		t.Errorf("expected 1 in target db, got %d", count3)
	}

	trimmed, _ := db.Trim(ctx, time.Now())
	if trimmed != 1 {
		t.Errorf("trimmed %d", trimmed)
	}
}

func TestTriple_Unmarshal_Error_Extra(t *testing.T) {
	t.Parallel()
	var tr Triple
	if err := tr.UnmarshalJSON([]byte(`{`)); err == nil {
		t.Error("expected error")
	}
	if err := tr.UnmarshalJSON([]byte(`{"subject": "!!!"}`)); err == nil {
		t.Error("expected error")
	}
	if err := tr.UnmarshalJSON([]byte(`{"subject": "YQ==", "predicate": "!!!"}`)); err == nil {
		t.Error("expected error")
	}
	if err := tr.UnmarshalJSON([]byte(`{"subject": "YQ==", "predicate": "Yg==", "object": "!!!"}`)); err == nil {
		t.Error("expected error")
	}
}

func TestIndex_PossibleIndexes_Empty_Extra(t *testing.T) {
	t.Parallel()
	res := PossibleIndexes(nil)
	if len(res) != 6 {
		t.Errorf("expected 6 indexes for empty fields, got %d", len(res))
	}
}

func TestTripleIterator_ParseError_Extra(t *testing.T) {
	t.Parallel()
	m := &mockStore{
		newIteratorFunc: func(slice *util.Range, ro *opt.ReadOptions) iterator.Iterator {
			return &mockIterator{next: true, value: []byte(`{`)}
		},
	}
	db := OpenWithDB(m)
	_, err := db.Get(context.Background(), &Pattern{})
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestJournal_Disabled_Errors_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := db.Trim(ctx, time.Now()); err != nil {
		t.Error(err)
	}
	if _, err := db.TrimAndExport(ctx, time.Now(), db); err != nil {
		t.Error(err)
	}
}

func TestNormalizeValue_Extra(t *testing.T) {
	t.Parallel()
	if normalizeValue(nil) != nil {
		t.Error("nil should be nil")
	}
	if normalizeValue([]byte{}) != nil {
		t.Error("empty bytes should be nil")
	}
	if normalizeValue("") != nil {
		t.Error("empty string should be nil")
	}
	if normalizeValue(123) != nil {
		t.Error("int should be nil")
	}
}

func TestNavigator_Count_Error_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	cleanup()
	nav := db.Nav(context.Background(), "alice").ArchOut("knows")
	_, err := nav.Count()
	if err == nil {
		t.Error("expected error")
	}
}

func TestNavigator_First_Error_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	cleanup()
	nav := db.Nav(context.Background(), "alice").ArchOut("knows")
	_, err := nav.First()
	if err == nil {
		t.Error("expected error")
	}
}

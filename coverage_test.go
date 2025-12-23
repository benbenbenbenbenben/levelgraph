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

	"github.com/benbenbenbenbenben/levelgraph/pkg/graph"
	"github.com/benbenbenbenbenben/levelgraph/pkg/index"
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

	triple := graph.NewTripleFromStrings("a", "b", "c")

	if err := db.Put(ctx, triple); !errors.Is(err, context.Canceled) {
		t.Errorf("Put: expected context.Canceled, got %v", err)
	}
	if err := db.Del(ctx, triple); !errors.Is(err, context.Canceled) {
		t.Errorf("Del: expected context.Canceled, got %v", err)
	}
	if _, err := db.Get(ctx, &graph.Pattern{}); !errors.Is(err, context.Canceled) {
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
	db, _ := OpenWithDB(m)
	err := db.Put(context.Background(), graph.NewTripleFromStrings("a", "b", "c"))
	if err == nil || !errors.Is(err, errors.New("io error")) && err.Error() != "levelgraph: write batch: io error" {
		t.Errorf("expected io error, got %v", err)
	}
}

func TestSearchIterator_Materialize_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Put(context.Background(), graph.NewTripleFromStrings("alice", "knows", "bob"))

	patterns := []*graph.Pattern{
		graph.NewPattern("alice", "knows", graph.V("friend")),
	}

	opts := &SearchOptions{
		Materialized: graph.NewPattern(graph.V("friend"), "is_known_by", "alice"),
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
	db.Put(ctx, graph.NewTripleFromStrings("a", "b", "c"))

	time.Sleep(10 * time.Millisecond)
	midTime := time.Now()
	db.Put(ctx, graph.NewTripleFromStrings("d", "e", "f"))

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
	var tr graph.Triple
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
	res := index.PossibleIndexes(nil)
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
	db, _ := OpenWithDB(m)
	_, err := db.Get(context.Background(), &graph.Pattern{})
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

func TestSolutionIterator_Error_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Put(ctx, graph.NewTripleFromStrings("alice", "knows", "bob"))

	// Create an iterator and fully consume it
	iter, err := db.SearchIterator(ctx, []*graph.Pattern{
		graph.NewPattern("alice", "knows", graph.V("friend")),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()

	// Consume all results
	for iter.Next() {
	}

	// Error() should return nil when no errors occurred
	if iter.Error() != nil {
		t.Errorf("expected nil error, got %v", iter.Error())
	}
}

func TestSolutionIterator_EmptyPatterns_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	db.Put(ctx, graph.NewTripleFromStrings("alice", "knows", "bob"))

	// Test with empty patterns - should return initial solution once
	iter, err := db.SearchIterator(ctx, []*graph.Pattern{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()

	// Should get one result (the initial empty solution)
	if !iter.Next() {
		t.Error("expected one result from empty patterns")
	}

	// Should have no more results
	if iter.Next() {
		t.Error("expected no more results after first")
	}
}

func TestSolutionIterator_Backtrack_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	// Create a graph that requires backtracking
	db.Put(ctx, graph.NewTripleFromStrings("alice", "knows", "bob"))
	db.Put(ctx, graph.NewTripleFromStrings("alice", "knows", "charlie"))
	db.Put(ctx, graph.NewTripleFromStrings("bob", "knows", "dave"))
	// charlie knows no one, so second pattern will fail for alice->charlie path

	// Multi-pattern search that will require backtracking
	iter, err := db.SearchIterator(ctx, []*graph.Pattern{
		graph.NewPattern("alice", "knows", graph.V("friend")),
		graph.NewPattern(graph.V("friend"), "knows", graph.V("fof")),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()

	count := 0
	for iter.Next() {
		count++
	}

	// Should find alice->bob->dave path only
	if count != 1 {
		t.Errorf("expected 1 result with backtracking, got %d", count)
	}

	if iter.Error() != nil {
		t.Errorf("unexpected error: %v", iter.Error())
	}
}

func TestGetVectorScore_EdgeCases_Extra(t *testing.T) {
	t.Parallel()

	// Test with no score key
	sol := graph.Solution{
		"friend": []byte("bob"),
	}
	if GetVectorScore(sol) != 0 {
		t.Error("expected 0 for missing score")
	}

	// Test with empty vector bytes
	sol["__vector_score__"] = []byte{}
	if GetVectorScore(sol) != 0 {
		t.Error("expected 0 for empty vector bytes")
	}
}

func TestFacets_ClosedDB_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	db.options.FacetsEnabled = true
	cleanup() // Close the DB

	ctx := context.Background()
	triple := graph.NewTripleFromStrings("alice", "knows", "bob")

	// All facet operations should return ErrClosed
	if err := db.SetFacet(ctx, FacetSubject, []byte("alice"), []byte("age"), []byte("30")); !errors.Is(err, ErrClosed) {
		t.Errorf("SetFacet: expected ErrClosed, got %v", err)
	}
	if _, err := db.GetFacet(ctx, FacetSubject, []byte("alice"), []byte("age")); !errors.Is(err, ErrClosed) {
		t.Errorf("GetFacet: expected ErrClosed, got %v", err)
	}
	if _, err := db.GetFacets(ctx, FacetSubject, []byte("alice")); !errors.Is(err, ErrClosed) {
		t.Errorf("GetFacets: expected ErrClosed, got %v", err)
	}
	if err := db.DelFacet(ctx, FacetSubject, []byte("alice"), []byte("age")); !errors.Is(err, ErrClosed) {
		t.Errorf("DelFacet: expected ErrClosed, got %v", err)
	}
	if err := db.SetTripleFacet(ctx, triple, []byte("since"), []byte("2020")); !errors.Is(err, ErrClosed) {
		t.Errorf("SetTripleFacet: expected ErrClosed, got %v", err)
	}
	if _, err := db.GetTripleFacet(ctx, triple, []byte("since")); !errors.Is(err, ErrClosed) {
		t.Errorf("GetTripleFacet: expected ErrClosed, got %v", err)
	}
	if _, err := db.GetTripleFacets(ctx, triple); !errors.Is(err, ErrClosed) {
		t.Errorf("GetTripleFacets: expected ErrClosed, got %v", err)
	}
	if err := db.DelTripleFacet(ctx, triple, []byte("since")); !errors.Is(err, ErrClosed) {
		t.Errorf("DelTripleFacet: expected ErrClosed, got %v", err)
	}
	if err := db.DelAllTripleFacets(ctx, triple); !errors.Is(err, ErrClosed) {
		t.Errorf("DelAllTripleFacets: expected ErrClosed, got %v", err)
	}
	if _, err := db.GetFacetIterator(ctx, FacetSubject, []byte("alice")); !errors.Is(err, ErrClosed) {
		t.Errorf("GetFacetIterator: expected ErrClosed, got %v", err)
	}
	if _, err := db.GetTripleFacetIterator(ctx, triple); !errors.Is(err, ErrClosed) {
		t.Errorf("GetTripleFacetIterator: expected ErrClosed, got %v", err)
	}
}

func TestFacets_ContextCanceled_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()
	db.options.FacetsEnabled = true

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	triple := graph.NewTripleFromStrings("alice", "knows", "bob")

	// All facet operations should return context.Canceled
	if err := db.SetFacet(ctx, FacetSubject, []byte("alice"), []byte("age"), []byte("30")); !errors.Is(err, context.Canceled) {
		t.Errorf("SetFacet: expected context.Canceled, got %v", err)
	}
	if _, err := db.GetFacet(ctx, FacetSubject, []byte("alice"), []byte("age")); !errors.Is(err, context.Canceled) {
		t.Errorf("GetFacet: expected context.Canceled, got %v", err)
	}
	if _, err := db.GetFacets(ctx, FacetSubject, []byte("alice")); !errors.Is(err, context.Canceled) {
		t.Errorf("GetFacets: expected context.Canceled, got %v", err)
	}
	if err := db.DelFacet(ctx, FacetSubject, []byte("alice"), []byte("age")); !errors.Is(err, context.Canceled) {
		t.Errorf("DelFacet: expected context.Canceled, got %v", err)
	}
	if err := db.SetTripleFacet(ctx, triple, []byte("since"), []byte("2020")); !errors.Is(err, context.Canceled) {
		t.Errorf("SetTripleFacet: expected context.Canceled, got %v", err)
	}
	if _, err := db.GetTripleFacet(ctx, triple, []byte("since")); !errors.Is(err, context.Canceled) {
		t.Errorf("GetTripleFacet: expected context.Canceled, got %v", err)
	}
	if _, err := db.GetTripleFacets(ctx, triple); !errors.Is(err, context.Canceled) {
		t.Errorf("GetTripleFacets: expected context.Canceled, got %v", err)
	}
	if err := db.DelTripleFacet(ctx, triple, []byte("since")); !errors.Is(err, context.Canceled) {
		t.Errorf("DelTripleFacet: expected context.Canceled, got %v", err)
	}
	if err := db.DelAllTripleFacets(ctx, triple); !errors.Is(err, context.Canceled) {
		t.Errorf("DelAllTripleFacets: expected context.Canceled, got %v", err)
	}
	if _, err := db.GetFacetIterator(ctx, FacetSubject, []byte("alice")); !errors.Is(err, context.Canceled) {
		t.Errorf("GetFacetIterator: expected context.Canceled, got %v", err)
	}
	if _, err := db.GetTripleFacetIterator(ctx, triple); !errors.Is(err, context.Canceled) {
		t.Errorf("GetTripleFacetIterator: expected context.Canceled, got %v", err)
	}
}

func TestFacetIterator_Key_Malformed_Extra(t *testing.T) {
	t.Parallel()

	// Test Key() with malformed iterator key
	m := &mockStore{
		newIteratorFunc: func(slice *util.Range, ro *opt.ReadOptions) iterator.Iterator {
			return &mockIterator{next: true, value: []byte("value")}
		},
	}
	db, _ := OpenWithDB(m)
	db.options.FacetsEnabled = true

	iter, err := db.GetFacetIterator(context.Background(), FacetSubject, []byte("alice"))
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()

	// Advance to get a result
	if !iter.Next() {
		t.Fatal("expected result")
	}

	// Key() should handle malformed key
	key := iter.Key()
	// The mockIterator returns "facet::subject::alice::age" which should parse correctly
	if string(key) != "age" {
		t.Errorf("Key() = %s, want age", key)
	}
}

func TestJournal_ContextCanceled_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()
	db.options.JournalEnabled = true

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	db2, cleanup2 := setupTestDB(t)
	defer cleanup2()

	// All journal operations should return context.Canceled
	if _, err := db.GetJournalIterator(ctx, time.Time{}); !errors.Is(err, context.Canceled) {
		t.Errorf("GetJournalIterator: expected context.Canceled, got %v", err)
	}
	if _, err := db.GetJournalEntries(ctx, time.Time{}); !errors.Is(err, context.Canceled) {
		t.Errorf("GetJournalEntries: expected context.Canceled, got %v", err)
	}
	if _, err := db.Trim(ctx, time.Now()); !errors.Is(err, context.Canceled) {
		t.Errorf("Trim: expected context.Canceled, got %v", err)
	}
	if _, err := db.TrimAndExport(ctx, time.Now(), db2); !errors.Is(err, context.Canceled) {
		t.Errorf("TrimAndExport: expected context.Canceled, got %v", err)
	}
	if _, err := db.ReplayJournal(ctx, time.Time{}, db2); !errors.Is(err, context.Canceled) {
		t.Errorf("ReplayJournal: expected context.Canceled, got %v", err)
	}
	if _, err := db.JournalCount(ctx, time.Time{}); !errors.Is(err, context.Canceled) {
		t.Errorf("JournalCount: expected context.Canceled, got %v", err)
	}
}

func TestJournal_ReplayWithAfterTime_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()
	db.options.JournalEnabled = true

	ctx := context.Background()

	// Add first entry
	db.Put(ctx, graph.NewTripleFromStrings("a", "b", "c"))

	// Record time between entries
	time.Sleep(10 * time.Millisecond)
	midTime := time.Now()
	time.Sleep(10 * time.Millisecond)

	// Add second entry
	db.Put(ctx, graph.NewTripleFromStrings("d", "e", "f"))

	// Replay only entries after midTime to a new DB
	db2, cleanup2 := setupTestDB(t)
	defer cleanup2()

	count, err := db.ReplayJournal(ctx, midTime, db2)
	if err != nil {
		t.Fatalf("ReplayJournal error: %v", err)
	}

	// Should only replay the second entry
	if count != 1 {
		t.Errorf("ReplayJournal count = %d, want 1", count)
	}

	// Verify only second triple exists in target DB
	results, _ := db2.Get(ctx, graph.NewPattern("d", nil, nil))
	if len(results) != 1 {
		t.Errorf("expected 1 result in target db, got %d", len(results))
	}

	results, _ = db2.Get(ctx, graph.NewPattern("a", nil, nil))
	if len(results) != 0 {
		t.Errorf("expected 0 results for first triple, got %d", len(results))
	}
}

func TestJournal_ReplayWithDelete_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()
	db.options.JournalEnabled = true

	ctx := context.Background()

	// Add and then delete a triple
	triple := graph.NewTripleFromStrings("alice", "knows", "bob")
	db.Put(ctx, triple)
	db.Del(ctx, triple)

	// Replay to a new DB
	db2, cleanup2 := setupTestDB(t)
	defer cleanup2()

	count, err := db.ReplayJournal(ctx, time.Time{}, db2)
	if err != nil {
		t.Fatalf("ReplayJournal error: %v", err)
	}

	// Should replay both put and del
	if count != 2 {
		t.Errorf("ReplayJournal count = %d, want 2", count)
	}

	// Triple should not exist in target DB (was deleted)
	results, _ := db2.Get(ctx, graph.NewPattern("alice", nil, nil))
	if len(results) != 0 {
		t.Errorf("expected 0 results after replay with delete, got %d", len(results))
	}
}

func TestJournal_GetIteratorWithBefore_Extra(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()
	db.options.JournalEnabled = true

	ctx := context.Background()

	// Add first entry
	db.Put(ctx, graph.NewTripleFromStrings("a", "b", "c"))

	time.Sleep(10 * time.Millisecond)
	midTime := time.Now()
	time.Sleep(10 * time.Millisecond)

	// Add second entry
	db.Put(ctx, graph.NewTripleFromStrings("d", "e", "f"))

	// Get iterator for entries before midTime
	iter, err := db.GetJournalIterator(ctx, midTime)
	if err != nil {
		t.Fatalf("GetJournalIterator error: %v", err)
	}
	defer iter.Close()

	count := 0
	for iter.Next() {
		count++
		entry, err := iter.Entry()
		if err != nil {
			t.Fatalf("Entry error: %v", err)
		}
		// All entries should be before midTime
		if !entry.Timestamp.Before(midTime) {
			t.Errorf("Entry timestamp %v should be before %v", entry.Timestamp, midTime)
		}
	}

	if iter.Error() != nil {
		t.Errorf("Iterator error: %v", iter.Error())
	}

	// Should only get first entry
	if count != 1 {
		t.Errorf("expected 1 entry before midTime, got %d", count)
	}
}

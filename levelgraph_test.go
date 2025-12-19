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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

func TestTriple(t *testing.T) {
	t.Run("NewTriple", func(t *testing.T) {
		triple := NewTriple([]byte("a"), []byte("b"), []byte("c"))
		if string(triple.Subject) != "a" {
			t.Errorf("expected subject 'a', got '%s'", triple.Subject)
		}
		if string(triple.Predicate) != "b" {
			t.Errorf("expected predicate 'b', got '%s'", triple.Predicate)
		}
		if string(triple.Object) != "c" {
			t.Errorf("expected object 'c', got '%s'", triple.Object)
		}
	})

	t.Run("NewTripleFromStrings", func(t *testing.T) {
		triple := NewTripleFromStrings("subject", "predicate", "object")
		if string(triple.Subject) != "subject" {
			t.Errorf("expected subject 'subject', got '%s'", triple.Subject)
		}
	})

	t.Run("Equal", func(t *testing.T) {
		t1 := NewTripleFromStrings("a", "b", "c")
		t2 := NewTripleFromStrings("a", "b", "c")
		t3 := NewTripleFromStrings("a", "b", "d")

		if !t1.Equal(t2) {
			t.Error("identical triples should be equal")
		}
		if t1.Equal(t3) {
			t.Error("different triples should not be equal")
		}
	})

	t.Run("Clone", func(t *testing.T) {
		original := NewTripleFromStrings("a", "b", "c")
		clone := original.Clone()

		if !original.Equal(clone) {
			t.Error("clone should be equal to original")
		}

		// Modify clone and ensure original is unchanged
		clone.Subject[0] = 'x'
		if original.Equal(clone) {
			t.Error("modifying clone should not affect original")
		}
	})
}

func TestVariable(t *testing.T) {
	t.Run("V constructor", func(t *testing.T) {
		v := V("x")
		if v.Name != "x" {
			t.Errorf("expected name 'x', got '%s'", v.Name)
		}
	})

	t.Run("Bind", func(t *testing.T) {
		v := V("x")
		solution := make(Solution)
		newSolution := v.Bind(solution, []byte("value"))

		if newSolution == nil {
			t.Fatal("bind should succeed")
		}
		if string(newSolution["x"]) != "value" {
			t.Errorf("expected 'value', got '%s'", newSolution["x"])
		}
	})

	t.Run("Bind conflict", func(t *testing.T) {
		v := V("x")
		solution := Solution{"x": []byte("existing")}
		newSolution := v.Bind(solution, []byte("different"))

		if newSolution != nil {
			t.Error("bind should fail when variable is already bound to different value")
		}
	})

	t.Run("Bind same value", func(t *testing.T) {
		v := V("x")
		solution := Solution{"x": []byte("value")}
		newSolution := v.Bind(solution, []byte("value"))

		if newSolution == nil {
			t.Error("bind should succeed when value matches")
		}
	})

	t.Run("IsBound", func(t *testing.T) {
		v := V("x")
		solution := Solution{"x": []byte("value")}
		emptyS := Solution{}

		if !v.IsBound(solution) {
			t.Error("should be bound")
		}
		if v.IsBound(emptyS) {
			t.Error("should not be bound")
		}
	})
}

func TestEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with:colon", "with\\:colon"},
		{"with\\backslash", "with\\\\backslash"},
		{"mixed:and\\chars", "mixed\\:and\\\\chars"},
		{"::", "\\:\\:"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Escape([]byte(tt.input))
			if string(result) != tt.expected {
				t.Errorf("Escape(%q) = %q, want %q", tt.input, result, tt.expected)
			}

			// Test round-trip
			unescaped := Unescape(result)
			if string(unescaped) != tt.input {
				t.Errorf("Unescape(Escape(%q)) = %q, want %q", tt.input, unescaped, tt.input)
			}
		})
	}
}

func TestGenKey(t *testing.T) {
	triple := NewTripleFromStrings("subject", "predicate", "object")

	t.Run("SPO index", func(t *testing.T) {
		key := GenKey(IndexSPO, triple)
		expected := "spo::subject::predicate::object"
		if string(key) != expected {
			t.Errorf("got %q, want %q", key, expected)
		}
	})

	t.Run("POS index", func(t *testing.T) {
		key := GenKey(IndexPOS, triple)
		expected := "pos::predicate::object::subject"
		if string(key) != expected {
			t.Errorf("got %q, want %q", key, expected)
		}
	})
}

func TestGenKeys(t *testing.T) {
	triple := NewTripleFromStrings("a", "b", "c")
	keys := GenKeys(triple)

	if len(keys) != 6 {
		t.Errorf("expected 6 keys, got %d", len(keys))
	}
}

func TestPattern(t *testing.T) {
	t.Run("NewPattern with strings", func(t *testing.T) {
		p := NewPattern("a", "b", "c")
		if string(p.GetConcreteValue("subject")) != "a" {
			t.Error("subject should be 'a'")
		}
	})

	t.Run("NewPattern with variable", func(t *testing.T) {
		v := V("x")
		p := NewPattern(v, "b", "c")

		if p.GetConcreteValue("subject") != nil {
			t.Error("subject should be nil (variable)")
		}
		if p.GetVariable("subject") != v {
			t.Error("subject should be variable x")
		}
	})

	t.Run("ConcreteFields", func(t *testing.T) {
		p := NewPattern("a", V("x"), "c")
		fields := p.ConcreteFields()

		if len(fields) != 2 {
			t.Errorf("expected 2 concrete fields, got %d", len(fields))
		}
	})

	t.Run("Matches", func(t *testing.T) {
		p := NewPattern("a", nil, nil)
		triple := NewTripleFromStrings("a", "b", "c")

		if !p.Matches(triple) {
			t.Error("pattern should match")
		}

		p2 := NewPattern("x", nil, nil)
		if p2.Matches(triple) {
			t.Error("pattern should not match")
		}
	})

	t.Run("BindTriple", func(t *testing.T) {
		p := NewPattern(V("x"), "b", V("y"))
		triple := NewTripleFromStrings("a", "b", "c")
		solution := p.BindTriple(nil, triple)

		if solution == nil {
			t.Fatal("binding should succeed")
		}
		if string(solution["x"]) != "a" {
			t.Errorf("x should be 'a', got '%s'", solution["x"])
		}
		if string(solution["y"]) != "c" {
			t.Errorf("y should be 'c', got '%s'", solution["y"])
		}
	})
}

func TestPossibleIndexes(t *testing.T) {
	tests := []struct {
		fields   []string
		expected int
	}{
		{[]string{"subject"}, 2},              // spo, sop
		{[]string{"predicate"}, 2},            // pos, pso
		{[]string{"object"}, 2},               // ops, osp
		{[]string{"subject", "predicate"}, 1}, // spo
		{[]string{"subject", "object"}, 1},    // sop
	}

	for _, tt := range tests {
		indexes := PossibleIndexes(tt.fields)
		if len(indexes) != tt.expected {
			t.Errorf("PossibleIndexes(%v) returned %d indexes, want %d", tt.fields, len(indexes), tt.expected)
		}
	}
}

func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "levelgraph-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(dir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(dir)
	}

	return db, cleanup
}

func TestDB_PutAndGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("Put single triple", func(t *testing.T) {
		triple := NewTripleFromStrings("a", "b", "c")
		err := db.Put(triple)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	})

	t.Run("Get by subject", func(t *testing.T) {
		results, err := db.Get(&Pattern{Subject: []byte("a")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if string(results[0].Subject) != "a" {
			t.Errorf("expected subject 'a', got '%s'", results[0].Subject)
		}
	})

	t.Run("Get by predicate", func(t *testing.T) {
		results, err := db.Get(&Pattern{Predicate: []byte("b")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("Get by object", func(t *testing.T) {
		results, err := db.Get(&Pattern{Object: []byte("c")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("Get by subject and predicate", func(t *testing.T) {
		results, err := db.Get(&Pattern{Subject: []byte("a"), Predicate: []byte("b")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("Get with no match", func(t *testing.T) {
		results, err := db.Get(&Pattern{Subject: []byte("notfound")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("expected 0 results, got %d", len(results))
		}
	})
}

func TestDB_Del(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	triple := NewTripleFromStrings("a", "b", "c")
	err := db.Put(triple)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify it exists
	results, _ := db.Get(&Pattern{Subject: []byte("a")})
	if len(results) != 1 {
		t.Fatal("triple should exist before delete")
	}

	// Delete it
	err = db.Del(triple)
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}

	// Verify it's gone
	results, _ = db.Get(&Pattern{Subject: []byte("a")})
	if len(results) != 0 {
		t.Fatal("triple should not exist after delete")
	}
}

func TestDB_MultipleTriples(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t1 := NewTripleFromStrings("a1", "b", "c")
	t2 := NewTripleFromStrings("a2", "b", "d")

	err := db.Put(t1, t2)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	t.Run("Get by shared predicate", func(t *testing.T) {
		results, err := db.Get(&Pattern{Predicate: []byte("b")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
	})

	t.Run("Get by specific subject", func(t *testing.T) {
		results, err := db.Get(&Pattern{Subject: []byte("a1")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})
}

func TestDB_Limit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t1 := NewTripleFromStrings("a1", "b", "c")
	t2 := NewTripleFromStrings("a2", "b", "d")
	db.Put(t1, t2)

	results, err := db.Get(&Pattern{Predicate: []byte("b"), Limit: 1})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result with limit, got %d", len(results))
	}
}

func TestDB_Offset(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t1 := NewTripleFromStrings("a1", "b", "c")
	t2 := NewTripleFromStrings("a2", "b", "d")
	db.Put(t1, t2)

	results, err := db.Get(&Pattern{Predicate: []byte("b"), Offset: 1})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result with offset, got %d", len(results))
	}
}

func TestDB_Filter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t1 := NewTripleFromStrings("a", "b", "c")
	t2 := NewTripleFromStrings("a", "b", "d")
	db.Put(t1, t2)

	filter := func(triple *Triple) bool {
		return string(triple.Object) == "d"
	}

	results, err := db.Get(&Pattern{Subject: []byte("a"), Filter: filter})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 filtered result, got %d", len(results))
	}
	if string(results[0].Object) != "d" {
		t.Errorf("expected object 'd', got '%s'", results[0].Object)
	}
}

func TestDB_SpecialCharacters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("String with ::", func(t *testing.T) {
		triple := NewTripleFromStrings("a::b", "predicate", "object")
		err := db.Put(triple)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		results, err := db.Get(&Pattern{Subject: []byte("a::b")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if string(results[0].Subject) != "a::b" {
			t.Errorf("expected 'a::b', got '%s'", results[0].Subject)
		}
	})

	t.Run("String with backslash", func(t *testing.T) {
		triple := NewTripleFromStrings("a\\b", "predicate", "object")
		err := db.Put(triple)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		results, err := db.Get(&Pattern{Subject: []byte("a\\b")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})
}

func TestDB_Options(t *testing.T) {
	dir, err := os.MkdirTemp("", "levelgraph-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath, WithJournal(), WithFacets(), WithBasicJoin())
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if !db.options.JournalEnabled {
		t.Error("journal should be enabled")
	}
	if !db.options.FacetsEnabled {
		t.Error("facets should be enabled")
	}
	if db.options.JoinAlgorithm != JoinAlgorithmBasic {
		t.Error("join algorithm should be basic")
	}
}

func TestGenerateBatch(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	triple := NewTripleFromStrings("a", "b", "c")
	ops, err := db.GenerateBatch(triple, "put")
	if err != nil {
		t.Fatalf("GenerateBatch failed: %v", err)
	}

	if len(ops) != 6 {
		t.Errorf("expected 6 operations, got %d", len(ops))
	}

	for _, op := range ops {
		if op.Type != "put" {
			t.Errorf("expected type 'put', got '%s'", op.Type)
		}
	}
}

// setupFOAFData sets up the FOAF test data from the JS fixtures
func setupFOAFData(db *DB) error {
	triples := []*Triple{
		NewTripleFromStrings("matteo", "friend", "daniele"),
		NewTripleFromStrings("daniele", "friend", "matteo"),
		NewTripleFromStrings("daniele", "friend", "marco"),
		NewTripleFromStrings("lucio", "friend", "matteo"),
		NewTripleFromStrings("lucio", "friend", "marco"),
		NewTripleFromStrings("marco", "friend", "davide"),
		NewTripleFromStrings("marco", "age", "32"),
		NewTripleFromStrings("daniele", "age", "25"),
		NewTripleFromStrings("lucio", "age", "15"),
		NewTripleFromStrings("davide", "age", "70"),
	}
	return db.Put(triples...)
}

func TestSearch_SinglePattern(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	t.Run("search with one result", func(t *testing.T) {
		results, err := db.Search([]*Pattern{
			{
				Subject:   V("x"),
				Predicate: []byte("friend"),
				Object:    []byte("daniele"),
			},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if string(results[0]["x"]) != "matteo" {
			t.Errorf("expected x='matteo', got '%s'", results[0]["x"])
		}
	})

	t.Run("search with multiple results", func(t *testing.T) {
		results, err := db.Search([]*Pattern{
			{
				Subject:   V("x"),
				Predicate: []byte("friend"),
				Object:    []byte("marco"),
			},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
	})
}

func TestSearch_Join(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	t.Run("two pattern join", func(t *testing.T) {
		// Find people who are friends with both marco and matteo
		results, err := db.Search([]*Pattern{
			{
				Subject:   V("x"),
				Predicate: []byte("friend"),
				Object:    []byte("marco"),
			},
			{
				Subject:   V("x"),
				Predicate: []byte("friend"),
				Object:    []byte("matteo"),
			},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results (daniele and lucio), got %d", len(results))
		}

		// Verify we got daniele and lucio
		names := make(map[string]bool)
		for _, r := range results {
			names[string(r["x"])] = true
		}
		if !names["daniele"] || !names["lucio"] {
			t.Error("expected daniele and lucio in results")
		}
	})

	t.Run("friend of friend", func(t *testing.T) {
		// Find friends of friends of matteo who are friends with davide
		results, err := db.Search([]*Pattern{
			{
				Subject:   []byte("matteo"),
				Predicate: []byte("friend"),
				Object:    V("x"),
			},
			{
				Subject:   V("x"),
				Predicate: []byte("friend"),
				Object:    V("y"),
			},
			{
				Subject:   V("y"),
				Predicate: []byte("friend"),
				Object:    []byte("davide"),
			},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if string(results[0]["x"]) != "daniele" {
			t.Errorf("expected x='daniele', got '%s'", results[0]["x"])
		}
		if string(results[0]["y"]) != "marco" {
			t.Errorf("expected y='marco', got '%s'", results[0]["y"])
		}
	})

	t.Run("mutual friends", func(t *testing.T) {
		// Find pairs where both are friends with each other
		results, err := db.Search([]*Pattern{
			{
				Subject:   V("x"),
				Predicate: []byte("friend"),
				Object:    V("y"),
			},
			{
				Subject:   V("y"),
				Predicate: []byte("friend"),
				Object:    V("x"),
			},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		// Should find matteo<->daniele and daniele<->matteo
		if len(results) != 2 {
			t.Fatalf("expected 2 mutual friend pairs, got %d", len(results))
		}
	})

	t.Run("common friends", func(t *testing.T) {
		// Find common friends of lucio and daniele
		results, err := db.Search([]*Pattern{
			{
				Subject:   []byte("lucio"),
				Predicate: []byte("friend"),
				Object:    V("x"),
			},
			{
				Subject:   []byte("daniele"),
				Predicate: []byte("friend"),
				Object:    V("x"),
			},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		// Both are friends with marco and matteo
		if len(results) != 2 {
			t.Fatalf("expected 2 common friends, got %d", len(results))
		}
	})
}

func TestSearch_Limit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	results, err := db.Search([]*Pattern{
		{
			Subject:   V("x"),
			Predicate: []byte("friend"),
			Object:    []byte("marco"),
		},
		{
			Subject:   V("x"),
			Predicate: []byte("friend"),
			Object:    []byte("matteo"),
		},
	}, &SearchOptions{Limit: 1})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result with limit, got %d", len(results))
	}
}

func TestSearch_Offset(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	results, err := db.Search([]*Pattern{
		{
			Subject:   V("x"),
			Predicate: []byte("friend"),
			Object:    []byte("marco"),
		},
		{
			Subject:   V("x"),
			Predicate: []byte("friend"),
			Object:    []byte("matteo"),
		},
	}, &SearchOptions{Offset: 1})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result with offset, got %d", len(results))
	}
}

func TestSearch_SolutionFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	// Find friends of matteo, but filter out daniele
	results, err := db.Search([]*Pattern{
		{
			Subject:   []byte("matteo"),
			Predicate: []byte("friend"),
			Object:    V("y"),
		},
		{
			Subject:   V("y"),
			Predicate: []byte("friend"),
			Object:    V("x"),
		},
	}, &SearchOptions{
		Filter: func(s Solution) bool {
			return string(s["x"]) != "matteo"
		},
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 filtered result, got %d", len(results))
	}
	if string(results[0]["x"]) != "marco" {
		t.Errorf("expected x='marco', got '%s'", results[0]["x"])
	}
}

func TestSearch_Materialized(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	results, err := db.Search([]*Pattern{
		{
			Subject:   V("x"),
			Predicate: []byte("friend"),
			Object:    []byte("marco"),
		},
		{
			Subject:   V("x"),
			Predicate: []byte("friend"),
			Object:    []byte("matteo"),
		},
	}, &SearchOptions{
		Materialized: &Pattern{
			Subject:   V("x"),
			Predicate: []byte("newpredicate"),
			Object:    []byte("abcde"),
		},
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 materialized results, got %d", len(results))
	}

	for _, r := range results {
		if string(r["predicate"]) != "newpredicate" {
			t.Errorf("expected predicate='newpredicate', got '%s'", r["predicate"])
		}
		if string(r["object"]) != "abcde" {
			t.Errorf("expected object='abcde', got '%s'", r["object"])
		}
	}
}

func TestSearch_PatternFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	// Find friends of daniele but filter at pattern level
	results, err := db.Search([]*Pattern{
		{
			Subject:   V("x"),
			Predicate: []byte("friend"),
			Object:    []byte("daniele"),
			Filter: func(t *Triple) bool {
				return string(t.Subject) != "matteo"
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results after filter, got %d", len(results))
	}
}

func TestSearch_EmptyPatterns(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	results, err := db.Search([]*Pattern{}, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results for empty patterns, got %d", len(results))
	}
}

func TestSearchIterator(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	iter, err := db.SearchIterator([]*Pattern{
		{
			Subject:   V("x"),
			Predicate: []byte("friend"),
			Object:    []byte("marco"),
		},
	}, nil)
	if err != nil {
		t.Fatalf("SearchIterator failed: %v", err)
	}
	defer iter.Close()

	count := 0
	for iter.Next() {
		sol := iter.Solution()
		if sol == nil {
			t.Error("solution should not be nil")
		}
		count++
	}

	if count != 2 {
		t.Errorf("expected 2 iterations, got %d", count)
	}
}

// Navigator tests - ported from JS navigator_spec.js

func TestNavigator_SingleVertex(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	values, err := db.Nav("matteo").Values()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if string(values[0]) != "matteo" {
		t.Errorf("expected 'matteo', got '%s'", values[0])
	}
}

func TestNavigator_ArchOut(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	values, err := db.Nav("matteo").ArchOut("friend").Values()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if string(values[0]) != "daniele" {
		t.Errorf("expected 'daniele', got '%s'", values[0])
	}
}

func TestNavigator_ArchIn(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	values, err := db.Nav("davide").ArchIn("friend").Values()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if string(values[0]) != "marco" {
		t.Errorf("expected 'marco', got '%s'", values[0])
	}
}

func TestNavigator_MultipleArchs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	// Follow path: davide <- friend <- friend -> friend
	values, err := db.Nav("davide").
		ArchIn("friend").
		ArchIn("friend").
		ArchOut("friend").
		Values()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}

	// Should contain marco and matteo
	found := make(map[string]bool)
	for _, v := range values {
		found[string(v)] = true
	}
	if !found["marco"] || !found["matteo"] {
		t.Errorf("expected marco and matteo, got %v", found)
	}
}

func TestNavigator_As(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	// marco <- friend as 'a' -> friend -> friend as 'a'
	// This should find cases where the same person appears at both positions
	values, err := db.Nav("marco").
		ArchIn("friend").
		As("a").
		ArchOut("friend").
		ArchOut("friend").
		As("a").
		Values()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if string(values[0]) != "daniele" {
		t.Errorf("expected 'daniele', got '%s'", values[0])
	}
}

func TestNavigator_Bind(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	// matteo <- friend.bind('lucio') -> friend.bind('marco')
	values, err := db.Nav("matteo").
		ArchIn("friend").
		Bind("lucio").
		ArchOut("friend").
		Bind("marco").
		Values()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if string(values[0]) != "marco" {
		t.Errorf("expected 'marco', got '%s'", values[0])
	}
}

func TestNavigator_StartFromVariable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	// Start from variable -> friend.bind('matteo') -> friend
	values, err := db.Nav(nil).
		ArchOut("friend").
		Bind("matteo").
		ArchOut("friend").
		Values()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if string(values[0]) != "daniele" {
		t.Errorf("expected 'daniele', got '%s'", values[0])
	}
}

func TestNavigator_Solutions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	solutions, err := db.Nav("daniele").
		ArchOut("friend").
		As("a").
		Solutions()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if len(solutions) != 2 {
		t.Fatalf("expected 2 solutions, got %d", len(solutions))
	}

	// Should have marco and matteo as 'a'
	found := make(map[string]bool)
	for _, sol := range solutions {
		if val, ok := sol["a"]; ok {
			found[string(val)] = true
		}
	}
	if !found["marco"] || !found["matteo"] {
		t.Errorf("expected marco and matteo as 'a', got %v", found)
	}
}

func TestNavigator_NoConditions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	// No conditions should return initial solution (empty)
	solutions, err := db.Nav("daniele").Solutions()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if len(solutions) != 1 {
		t.Fatalf("expected 1 empty solution, got %d", len(solutions))
	}
}

func TestNavigator_Go(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	// marco <- friend as 'a', go to matteo -> friend as 'b'
	solutions, err := db.Nav("marco").
		ArchIn("friend").
		As("a").
		Go("matteo").
		ArchOut("friend").
		As("b").
		Solutions()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if len(solutions) != 2 {
		t.Fatalf("expected 2 solutions, got %d", len(solutions))
	}

	// Both should have b='daniele', and a should be 'daniele' or 'lucio'
	for _, sol := range solutions {
		if string(sol["b"]) != "daniele" {
			t.Errorf("expected b='daniele', got '%s'", sol["b"])
		}
	}
}

func TestNavigator_GoToVariable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	// marco, go to var as 'a' -> friend as 'b'.bind('matteo')
	solutions, err := db.Nav("marco").
		Go(nil).
		As("a").
		ArchOut("friend").
		As("b").
		Bind("matteo").
		Solutions()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if len(solutions) != 2 {
		t.Fatalf("expected 2 solutions, got %d", len(solutions))
	}

	// Both should have b='matteo', and a should be 'daniele' or 'lucio'
	for _, sol := range solutions {
		if string(sol["b"]) != "matteo" {
			t.Errorf("expected b='matteo', got '%s'", sol["b"])
		}
	}
}

func TestNavigator_Exists(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	exists, err := db.Nav("matteo").ArchOut("friend").Exists()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if !exists {
		t.Error("expected to find friends of matteo")
	}

	exists, err = db.Nav("nobody").ArchOut("friend").Exists()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if exists {
		t.Error("expected not to find friends of nobody")
	}
}

func TestNavigator_Count(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	count, err := db.Nav("daniele").ArchOut("friend").Count()
	if err != nil {
		t.Fatalf("Navigator failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 friends, got %d", count)
	}
}

func TestNavigator_Clone(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := setupFOAFData(db); err != nil {
		t.Fatalf("failed to setup data: %v", err)
	}

	nav1 := db.Nav("matteo").ArchOut("friend")
	nav2 := nav1.Clone().ArchOut("friend")

	// nav1 should have 1 condition, nav2 should have 2
	vals1, _ := nav1.Values()
	vals2, _ := nav2.Values()

	if len(vals1) != 1 {
		t.Errorf("nav1 expected 1 value, got %d", len(vals1))
	}
	if len(vals2) != 2 {
		t.Errorf("nav2 expected 2 values, got %d", len(vals2))
	}
}

// Journal tests

func setupJournalDB(t *testing.T) (*DB, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "levelgraph-journal-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(dir, "test.db")
	db, err := Open(dbPath, WithJournal())
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(dir)
	}

	return db, cleanup
}

func TestJournal_RecordsOperations(t *testing.T) {
	db, cleanup := setupJournalDB(t)
	defer cleanup()

	// Put a triple
	t1 := NewTripleFromStrings("a", "b", "c")
	if err := db.Put(t1); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Check journal has an entry
	count, err := db.JournalCount(time.Time{})
	if err != nil {
		t.Fatalf("JournalCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 journal entry, got %d", count)
	}

	// Get the entry
	entries, err := db.GetJournalEntries(time.Time{})
	if err != nil {
		t.Fatalf("GetJournalEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Operation != "put" {
		t.Errorf("expected op 'put', got '%s'", entries[0].Operation)
	}
	if !entries[0].Triple.Equal(t1) {
		t.Errorf("triple mismatch")
	}

	// Delete the triple
	if err := db.Del(t1); err != nil {
		t.Fatalf("Del failed: %v", err)
	}

	count, _ = db.JournalCount(time.Time{})
	if count != 2 {
		t.Errorf("expected 2 journal entries, got %d", count)
	}

	entries, _ = db.GetJournalEntries(time.Time{})
	if entries[1].Operation != "del" {
		t.Errorf("expected op 'del', got '%s'", entries[1].Operation)
	}
}

func TestJournal_Trim(t *testing.T) {
	db, cleanup := setupJournalDB(t)
	defer cleanup()

	// Put some triples
	t1 := NewTripleFromStrings("a", "b", "c")
	t2 := NewTripleFromStrings("d", "e", "f")
	db.Put(t1)

	// Wait a tiny bit to ensure different timestamps
	time.Sleep(10 * time.Millisecond)
	trimTime := time.Now()
	time.Sleep(10 * time.Millisecond)

	db.Put(t2)

	// Should have 2 entries
	count, _ := db.JournalCount(time.Time{})
	if count != 2 {
		t.Errorf("expected 2 journal entries before trim, got %d", count)
	}

	// Trim entries before trimTime
	trimmed, err := db.Trim(trimTime)
	if err != nil {
		t.Fatalf("Trim failed: %v", err)
	}
	if trimmed != 1 {
		t.Errorf("expected to trim 1 entry, trimmed %d", trimmed)
	}

	// Should have 1 entry remaining
	count, _ = db.JournalCount(time.Time{})
	if count != 1 {
		t.Errorf("expected 1 journal entry after trim, got %d", count)
	}
}

func TestJournal_TrimAndExport(t *testing.T) {
	db, cleanup := setupJournalDB(t)
	defer cleanup()

	// Create export target database
	dir, _ := os.MkdirTemp("", "levelgraph-export-*")
	defer os.RemoveAll(dir)

	exportPath := filepath.Join(dir, "export.db")
	exportDB, err := Open(exportPath, WithJournal())
	if err != nil {
		t.Fatalf("failed to open export database: %v", err)
	}
	defer exportDB.Close()

	// Put some triples
	t1 := NewTripleFromStrings("a", "b", "c")
	t2 := NewTripleFromStrings("d", "e", "f")
	db.Put(t1)

	time.Sleep(10 * time.Millisecond)
	trimTime := time.Now()
	time.Sleep(10 * time.Millisecond)

	db.Put(t2)

	// Export old entries
	exported, err := db.TrimAndExport(trimTime, exportDB)
	if err != nil {
		t.Fatalf("TrimAndExport failed: %v", err)
	}
	if exported != 1 {
		t.Errorf("expected to export 1 entry, exported %d", exported)
	}

	// Main DB should have 1 entry
	mainCount, _ := db.JournalCount(time.Time{})
	if mainCount != 1 {
		t.Errorf("expected 1 entry in main db, got %d", mainCount)
	}

	// Export DB should have 1 entry
	exportCount, _ := exportDB.JournalCount(time.Time{})
	if exportCount != 1 {
		t.Errorf("expected 1 entry in export db, got %d", exportCount)
	}
}

func TestJournal_Replay(t *testing.T) {
	db, cleanup := setupJournalDB(t)
	defer cleanup()

	// Create replay target database
	dir, _ := os.MkdirTemp("", "levelgraph-replay-*")
	defer os.RemoveAll(dir)

	replayPath := filepath.Join(dir, "replay.db")
	replayDB, err := Open(replayPath)
	if err != nil {
		t.Fatalf("failed to open replay database: %v", err)
	}
	defer replayDB.Close()

	// Put some triples
	t1 := NewTripleFromStrings("a", "b", "c")
	t2 := NewTripleFromStrings("d", "e", "f")
	db.Put(t1)
	db.Put(t2)
	db.Del(t1)

	// Replay to target
	replayed, err := db.ReplayJournal(time.Time{}, replayDB)
	if err != nil {
		t.Fatalf("ReplayJournal failed: %v", err)
	}
	if replayed != 3 {
		t.Errorf("expected to replay 3 operations, replayed %d", replayed)
	}

	// Check replay result - should only have t2
	results1, _ := replayDB.Get(&Pattern{Subject: []byte("a")})
	if len(results1) != 0 {
		t.Error("expected triple 'a' to be deleted")
	}

	results2, _ := replayDB.Get(&Pattern{Subject: []byte("d")})
	if len(results2) != 1 {
		t.Error("expected triple 'd' to exist")
	}
}

func TestJournal_DisabledByDefault(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Put a triple (journal should be disabled)
	t1 := NewTripleFromStrings("a", "b", "c")
	db.Put(t1)

	// Journal should be empty
	count, _ := db.JournalCount(time.Time{})
	if count != 0 {
		t.Errorf("expected 0 journal entries (disabled), got %d", count)
	}
}

func TestJournal_Iterator(t *testing.T) {
	db, cleanup := setupJournalDB(t)
	defer cleanup()

	// Put some triples
	db.Put(NewTripleFromStrings("a", "b", "c"))
	db.Put(NewTripleFromStrings("d", "e", "f"))
	db.Put(NewTripleFromStrings("g", "h", "i"))

	iter, err := db.GetJournalIterator(time.Time{})
	if err != nil {
		t.Fatalf("GetJournalIterator failed: %v", err)
	}
	defer iter.Close()

	count := 0
	for iter.Next() {
		entry, err := iter.Entry()
		if err != nil {
			t.Fatalf("Entry() failed: %v", err)
		}
		if entry.Operation != "put" {
			t.Errorf("expected 'put' operation")
		}
		count++
	}

	if err := iter.Error(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 entries, got %d", count)
	}
}

// Facet tests

func setupFacetDB(t *testing.T) (*DB, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "levelgraph-facet-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(dir, "test.db")
	db, err := Open(dbPath, WithFacets())
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(dir)
	}

	return db, cleanup
}

func TestFacet_ComponentFacets(t *testing.T) {
	db, cleanup := setupFacetDB(t)
	defer cleanup()

	// Set a facet on a subject
	err := db.SetFacet(FacetSubject, []byte("alice"), []byte("age"), []byte("30"))
	if err != nil {
		t.Fatalf("SetFacet failed: %v", err)
	}

	// Get the facet
	value, err := db.GetFacet(FacetSubject, []byte("alice"), []byte("age"))
	if err != nil {
		t.Fatalf("GetFacet failed: %v", err)
	}
	if string(value) != "30" {
		t.Errorf("expected '30', got '%s'", value)
	}

	// Set another facet
	err = db.SetFacet(FacetSubject, []byte("alice"), []byte("city"), []byte("NYC"))
	if err != nil {
		t.Fatalf("SetFacet failed: %v", err)
	}

	// Get all facets
	facets, err := db.GetFacets(FacetSubject, []byte("alice"))
	if err != nil {
		t.Fatalf("GetFacets failed: %v", err)
	}
	if len(facets) != 2 {
		t.Errorf("expected 2 facets, got %d", len(facets))
	}
	if string(facets["age"]) != "30" {
		t.Errorf("expected age='30'")
	}
	if string(facets["city"]) != "NYC" {
		t.Errorf("expected city='NYC'")
	}

	// Delete a facet
	err = db.DelFacet(FacetSubject, []byte("alice"), []byte("age"))
	if err != nil {
		t.Fatalf("DelFacet failed: %v", err)
	}

	// Verify deletion
	value, err = db.GetFacet(FacetSubject, []byte("alice"), []byte("age"))
	if err != nil {
		t.Fatalf("GetFacet failed: %v", err)
	}
	if value != nil {
		t.Errorf("expected nil after deletion, got '%s'", value)
	}
}

func TestFacet_TripleFacets(t *testing.T) {
	db, cleanup := setupFacetDB(t)
	defer cleanup()

	triple := NewTripleFromStrings("alice", "knows", "bob")

	// Put the triple
	db.Put(triple)

	// Set a facet on the triple
	err := db.SetTripleFacet(triple, []byte("since"), []byte("2020"))
	if err != nil {
		t.Fatalf("SetTripleFacet failed: %v", err)
	}

	// Get the facet
	value, err := db.GetTripleFacet(triple, []byte("since"))
	if err != nil {
		t.Fatalf("GetTripleFacet failed: %v", err)
	}
	if string(value) != "2020" {
		t.Errorf("expected '2020', got '%s'", value)
	}

	// Set another facet
	err = db.SetTripleFacet(triple, []byte("weight"), []byte("0.9"))
	if err != nil {
		t.Fatalf("SetTripleFacet failed: %v", err)
	}

	// Get all facets
	facets, err := db.GetTripleFacets(triple)
	if err != nil {
		t.Fatalf("GetTripleFacets failed: %v", err)
	}
	if len(facets) != 2 {
		t.Errorf("expected 2 facets, got %d", len(facets))
	}

	// Delete a facet
	err = db.DelTripleFacet(triple, []byte("since"))
	if err != nil {
		t.Fatalf("DelTripleFacet failed: %v", err)
	}

	facets, _ = db.GetTripleFacets(triple)
	if len(facets) != 1 {
		t.Errorf("expected 1 facet after deletion, got %d", len(facets))
	}

	// Delete all facets
	err = db.DelAllTripleFacets(triple)
	if err != nil {
		t.Fatalf("DelAllTripleFacets failed: %v", err)
	}

	facets, _ = db.GetTripleFacets(triple)
	if len(facets) != 0 {
		t.Errorf("expected 0 facets after delete all, got %d", len(facets))
	}
}

func TestFacet_DisabledByDefault(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.SetFacet(FacetSubject, []byte("alice"), []byte("age"), []byte("30"))
	if err != ErrFacetsDisabled {
		t.Errorf("expected ErrFacetsDisabled, got %v", err)
	}
}

func TestFacet_Iterator(t *testing.T) {
	db, cleanup := setupFacetDB(t)
	defer cleanup()

	// Set multiple facets
	db.SetFacet(FacetSubject, []byte("alice"), []byte("age"), []byte("30"))
	db.SetFacet(FacetSubject, []byte("alice"), []byte("city"), []byte("NYC"))
	db.SetFacet(FacetSubject, []byte("alice"), []byte("email"), []byte("alice@example.com"))

	iter, err := db.GetFacetIterator(FacetSubject, []byte("alice"))
	if err != nil {
		t.Fatalf("GetFacetIterator failed: %v", err)
	}
	defer iter.Close()

	count := 0
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		if key == nil || value == nil {
			t.Error("key and value should not be nil")
		}
		count++
	}

	if err := iter.Error(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 facets, got %d", count)
	}
}

func TestFacet_DifferentTypes(t *testing.T) {
	db, cleanup := setupFacetDB(t)
	defer cleanup()

	// Set facets on different component types
	db.SetFacet(FacetSubject, []byte("value1"), []byte("key"), []byte("subject_facet"))
	db.SetFacet(FacetPredicate, []byte("value1"), []byte("key"), []byte("predicate_facet"))
	db.SetFacet(FacetObject, []byte("value1"), []byte("key"), []byte("object_facet"))

	// Verify they are stored separately
	v1, _ := db.GetFacet(FacetSubject, []byte("value1"), []byte("key"))
	v2, _ := db.GetFacet(FacetPredicate, []byte("value1"), []byte("key"))
	v3, _ := db.GetFacet(FacetObject, []byte("value1"), []byte("key"))

	if string(v1) != "subject_facet" {
		t.Errorf("expected 'subject_facet', got '%s'", v1)
	}
	if string(v2) != "predicate_facet" {
		t.Errorf("expected 'predicate_facet', got '%s'", v2)
	}
	if string(v3) != "object_facet" {
		t.Errorf("expected 'object_facet', got '%s'", v3)
	}
}

func TestFacet_SpecialCharacters(t *testing.T) {
	db, cleanup := setupFacetDB(t)
	defer cleanup()

	// Test with special characters in values
	err := db.SetFacet(FacetSubject, []byte("alice::bob"), []byte("key::with::colons"), []byte("value\\with\\backslash"))
	if err != nil {
		t.Fatalf("SetFacet failed: %v", err)
	}

	value, err := db.GetFacet(FacetSubject, []byte("alice::bob"), []byte("key::with::colons"))
	if err != nil {
		t.Fatalf("GetFacet failed: %v", err)
	}
	if string(value) != "value\\with\\backslash" {
		t.Errorf("expected 'value\\with\\backslash', got '%s'", value)
	}
}

// Unicode tests - ported from triple_unicode_store_spec.js

func TestUnicode_BasicTriple(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Chinese characters
	triple := NewTriple([]byte("车"), []byte("是"), []byte("交通工具"))
	err := db.Put(triple)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	results, err := db.Get(&Pattern{Subject: []byte("车")})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if string(results[0].Subject) != "车" {
		t.Errorf("expected '车', got '%s'", results[0].Subject)
	}
}

func TestUnicode_Emoji(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Emoji and special unicode characters from JS test
	triple := NewTriple([]byte("􀃿"), []byte("🜁"), []byte("🚃"))
	err := db.Put(triple)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	t.Run("Get by subject", func(t *testing.T) {
		results, err := db.Get(&Pattern{Subject: []byte("􀃿")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("Get by object", func(t *testing.T) {
		results, err := db.Get(&Pattern{Object: []byte("🚃")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("Get by predicate", func(t *testing.T) {
		results, err := db.Get(&Pattern{Predicate: []byte("🜁")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("Get by subject and predicate", func(t *testing.T) {
		results, err := db.Get(&Pattern{Subject: []byte("􀃿"), Predicate: []byte("🜁")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})
}

func TestUnicode_ExactMatch(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Chinese characters - test exact matching
	t1 := NewTriple([]byte("飞机"), []byte("是"), []byte("交通工具"))
	t2 := NewTriple([]byte("车"), []byte("是"), []byte("动物"))
	db.Put(t1, t2)

	results, err := db.Get(&Pattern{Subject: []byte("车")})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for exact match, got %d", len(results))
	}
	if string(results[0].Object) != "动物" {
		t.Errorf("expected object '动物', got '%s'", results[0].Object)
	}
}

func TestUnicode_MultipleTriples(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t1 := NewTriple([]byte("飞机"), []byte("是"), []byte("交通工具"))
	t2 := NewTriple([]byte("狗熊"), []byte("是"), []byte("动物"))
	db.Put(t1, t2)

	t.Run("Get by shared predicate", func(t *testing.T) {
		results, err := db.Get(&Pattern{Predicate: []byte("是")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
	})

	t.Run("Delete and verify", func(t *testing.T) {
		db.Del(t2)
		results, err := db.Get(&Pattern{Predicate: []byte("是")})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result after delete, got %d", len(results))
		}
	})
}

func TestUnicode_Filter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t1 := NewTriple([]byte("车"), []byte("是"), []byte("动物"))
	t2 := NewTriple([]byte("车"), []byte("是"), []byte("交通工具"))
	db.Put(t1, t2)

	filter := func(triple *Triple) bool {
		return string(triple.Object) == "动物"
	}

	results, err := db.Get(&Pattern{Subject: []byte("车"), Predicate: []byte("是"), Filter: filter})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 filtered result, got %d", len(results))
	}
	if string(results[0].Object) != "动物" {
		t.Errorf("expected '动物', got '%s'", results[0].Object)
	}
}

// Binary data tests - test arbitrary byte sequences

func TestBinary_ArbitraryBytes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test with binary data containing null bytes and other special bytes
	subject := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	predicate := []byte{0x10, 0x20, 0x30}
	object := []byte{0xAA, 0xBB, 0xCC, 0x00, 0xDD}

	triple := NewTriple(subject, predicate, object)
	err := db.Put(triple)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	results, err := db.Get(&Pattern{Subject: subject})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Verify the binary data is preserved exactly
	if string(results[0].Subject) != string(subject) {
		t.Errorf("subject mismatch")
	}
	if string(results[0].Predicate) != string(predicate) {
		t.Errorf("predicate mismatch")
	}
	if string(results[0].Object) != string(object) {
		t.Errorf("object mismatch")
	}
}

func TestBinary_MixedContent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Mix of text and binary
	subject := []byte("user:123")
	predicate := []byte("avatar")
	object := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A} // PNG magic bytes

	triple := NewTriple(subject, predicate, object)
	db.Put(triple)

	results, err := db.Get(&Pattern{Subject: subject, Predicate: predicate})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Check PNG magic bytes preserved
	if results[0].Object[0] != 0x89 || results[0].Object[1] != 0x50 {
		t.Error("binary data not preserved correctly")
	}
}

// Additional edge case tests from JS spec

func TestEdgeCase_PrefixMatching(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Ensure 'a' doesn't match 'a1'
	t1 := NewTripleFromStrings("a1", "b", "c")
	t2 := NewTripleFromStrings("a", "b", "d")
	db.Put(t1, t2)

	results, err := db.Get(&Pattern{Subject: []byte("a")})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (exact match only), got %d", len(results))
	}
	if string(results[0].Object) != "d" {
		t.Errorf("expected object 'd', got '%s'", results[0].Object)
	}
}

func TestEdgeCase_StringEndingWithColon(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t1 := NewTripleFromStrings("a", "b", "c")
	t2 := NewTripleFromStrings("a:", "b", "c")
	db.Put(t1, t2)

	results, err := db.Get(&Pattern{Subject: []byte("a:")})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if string(results[0].Subject) != "a:" {
		t.Errorf("expected 'a:', got '%s'", results[0].Subject)
	}
}

func TestEdgeCase_StringEndingWithBackslash(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t1 := NewTripleFromStrings("a", "b", "c")
	t2 := NewTripleFromStrings("a\\", "b", "c")
	db.Put(t1, t2)

	results, err := db.Get(&Pattern{Subject: []byte("a\\")})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if string(results[0].Subject) != "a\\" {
		t.Errorf("expected 'a\\', got '%s'", results[0].Subject)
	}
}

func TestEdgeCase_StringWithEscapedSeparator(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t1 := NewTripleFromStrings("a", "b", "c")
	t2 := NewTripleFromStrings("a\\::a", "b", "c")
	db.Put(t1, t2)

	results, err := db.Get(&Pattern{Subject: []byte("a")})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestEdgeCase_EmptyPattern(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t1 := NewTripleFromStrings("a", "b", "c")
	t2 := NewTripleFromStrings("d", "e", "f")
	db.Put(t1, t2)

	// Empty pattern should return all triples
	results, err := db.Get(&Pattern{})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for empty pattern, got %d", len(results))
	}
}

func TestEdgeCase_Reverse(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t1 := NewTripleFromStrings("a1", "b", "c")
	t2 := NewTripleFromStrings("a2", "b", "d")
	db.Put(t1, t2)

	results, err := db.Get(&Pattern{Predicate: []byte("b"), Reverse: true})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// In reverse order, a2 should come before a1
	if string(results[0].Subject) != "a2" {
		t.Errorf("expected first result 'a2' in reverse, got '%s'", results[0].Subject)
	}
	if string(results[1].Subject) != "a1" {
		t.Errorf("expected second result 'a1' in reverse, got '%s'", results[1].Subject)
	}
}

func TestEdgeCase_ReverseLimitOffset(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t1 := NewTripleFromStrings("a1", "b", "c")
	t2 := NewTripleFromStrings("a2", "b", "d")
	t3 := NewTripleFromStrings("a3", "b", "e")
	db.Put(t1, t2, t3)

	results, err := db.Get(&Pattern{Predicate: []byte("b"), Reverse: true, Limit: 1})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if string(results[0].Subject) != "a3" {
		t.Errorf("expected 'a3' with reverse and limit, got '%s'", results[0].Subject)
	}
}

// ============================================================================
// Additional coverage tests for uncovered utility functions
// ============================================================================

func TestTriple_String(t *testing.T) {
	triple := NewTripleFromStrings("alice", "knows", "bob")
	str := triple.String()
	expected := "alice knows bob"
	if str != expected {
		t.Errorf("expected '%s', got '%s'", expected, str)
	}
}

func TestTriple_Set(t *testing.T) {
	triple := NewTripleFromStrings("a", "b", "c")

	triple.Set("subject", []byte("x"))
	if string(triple.Subject) != "x" {
		t.Errorf("expected subject 'x', got '%s'", triple.Subject)
	}

	triple.Set("predicate", []byte("y"))
	if string(triple.Predicate) != "y" {
		t.Errorf("expected predicate 'y', got '%s'", triple.Predicate)
	}

	triple.Set("object", []byte("z"))
	if string(triple.Object) != "z" {
		t.Errorf("expected object 'z', got '%s'", triple.Object)
	}

	// Invalid field should be a no-op
	triple.Set("invalid", []byte("test"))
	if string(triple.Subject) != "x" || string(triple.Predicate) != "y" || string(triple.Object) != "z" {
		t.Error("invalid field should not change triple")
	}
}

func TestVariable_GetValue(t *testing.T) {
	v := V("x")
	sol := Solution{"x": []byte("value")}

	result := v.GetValue(sol)
	if string(result) != "value" {
		t.Errorf("expected 'value', got '%s'", result)
	}

	// Unbound variable
	emptySol := Solution{}
	result = v.GetValue(emptySol)
	if result != nil {
		t.Error("unbound variable should return nil")
	}
}

func TestVariable_Equal(t *testing.T) {
	sol1 := Solution{"x": []byte("a"), "y": []byte("b")}
	sol2 := Solution{"x": []byte("a"), "y": []byte("b")}
	sol3 := Solution{"x": []byte("a"), "y": []byte("c")}
	sol4 := Solution{"x": []byte("a")}

	if !sol1.Equal(sol2) {
		t.Error("identical solutions should be equal")
	}
	if sol1.Equal(sol3) {
		t.Error("different values should not be equal")
	}
	if sol1.Equal(sol4) {
		t.Error("different lengths should not be equal")
	}
}

func TestIsVariable(t *testing.T) {
	v := V("x")
	if !IsVariable(v) {
		t.Error("V() result should be a Variable")
	}
	if IsVariable([]byte("test")) {
		t.Error("[]byte should not be a Variable")
	}
	if IsVariable("test") {
		t.Error("string should not be a Variable")
	}
	if IsVariable(nil) {
		t.Error("nil should not be a Variable")
	}
}

func TestAsVariable(t *testing.T) {
	v := V("x")
	result, ok := AsVariable(v)
	if !ok {
		t.Error("AsVariable should succeed for Variable")
	}
	if result.Name != "x" {
		t.Errorf("expected name 'x', got '%s'", result.Name)
	}

	result, ok = AsVariable([]byte("test"))
	if ok {
		t.Error("AsVariable should fail for []byte")
	}
	if result != nil {
		t.Error("AsVariable should return nil for non-Variable")
	}
}

func TestPattern_HasVariable(t *testing.T) {
	p1 := &Pattern{Subject: []byte("a"), Predicate: []byte("b"), Object: []byte("c")}
	if p1.HasVariable() {
		t.Error("pattern without variables should return false")
	}

	p2 := &Pattern{Subject: V("x"), Predicate: []byte("b"), Object: []byte("c")}
	if !p2.HasVariable() {
		t.Error("pattern with subject variable should return true")
	}

	p3 := &Pattern{Subject: []byte("a"), Predicate: V("p"), Object: []byte("c")}
	if !p3.HasVariable() {
		t.Error("pattern with predicate variable should return true")
	}

	p4 := &Pattern{Subject: []byte("a"), Predicate: []byte("b"), Object: V("o")}
	if !p4.HasVariable() {
		t.Error("pattern with object variable should return true")
	}
}

func TestPattern_VariableFields(t *testing.T) {
	p := &Pattern{Subject: V("s"), Predicate: []byte("p"), Object: V("o")}
	fields := p.VariableFields()

	if len(fields) != 2 {
		t.Errorf("expected 2 variable fields, got %d", len(fields))
	}
	if fields["subject"] == nil || fields["subject"].Name != "s" {
		t.Error("subject variable not found")
	}
	if fields["object"] == nil || fields["object"].Name != "o" {
		t.Error("object variable not found")
	}
	if fields["predicate"] != nil {
		t.Error("predicate should not be in variable fields")
	}
}

func TestPattern_ToTriple(t *testing.T) {
	// Pattern with all concrete values
	p1 := &Pattern{Subject: []byte("a"), Predicate: []byte("b"), Object: []byte("c")}
	triple := p1.ToTriple()
	if triple == nil {
		t.Fatal("expected triple, got nil")
	}
	if string(triple.Subject) != "a" || string(triple.Predicate) != "b" || string(triple.Object) != "c" {
		t.Error("triple values don't match pattern")
	}

	// Pattern with nil field
	p2 := &Pattern{Subject: []byte("a"), Predicate: nil, Object: []byte("c")}
	if p2.ToTriple() != nil {
		t.Error("pattern with nil field should return nil")
	}

	// Pattern with variable
	p3 := &Pattern{Subject: []byte("a"), Predicate: V("p"), Object: []byte("c")}
	if p3.ToTriple() != nil {
		t.Error("pattern with variable should return nil")
	}
}

func TestDB_V(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	v := db.V("test")
	if v == nil {
		t.Fatal("V should return a Variable")
	}
	if v.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", v.Name)
	}
}

func TestDB_IsOpen(t *testing.T) {
	db, cleanup := setupTestDB(t)

	if !db.IsOpen() {
		t.Error("database should be open")
	}

	cleanup()

	if db.IsOpen() {
		t.Error("database should be closed after cleanup")
	}
}

func TestDB_GetIterator(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t1 := NewTripleFromStrings("a", "b", "c")
	t2 := NewTripleFromStrings("a", "b", "d")
	db.Put(t1, t2)

	iter, err := db.GetIterator(&Pattern{Subject: []byte("a")})
	if err != nil {
		t.Fatalf("GetIterator failed: %v", err)
	}
	defer iter.Release()

	count := 0
	for iter.Next() {
		_, err := iter.Triple()
		if err != nil {
			t.Fatalf("Triple failed: %v", err)
		}
		count++
	}
	if err := iter.Error(); err != nil {
		t.Fatalf("Iterator error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 triples, got %d", count)
	}
}

func TestParseKey(t *testing.T) {
	// Generate a key and parse it back
	triple := NewTripleFromStrings("alice", "knows", "bob")
	key := GenKey(IndexSPO, triple)

	indexName, values := ParseKey(key)
	if indexName != IndexSPO {
		t.Errorf("expected index %s, got %s", IndexSPO, indexName)
	}
	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(values))
	}
	if string(values[0]) != "alice" {
		t.Errorf("expected 'alice', got '%s'", values[0])
	}
	if string(values[1]) != "knows" {
		t.Errorf("expected 'knows', got '%s'", values[1])
	}
	if string(values[2]) != "bob" {
		t.Errorf("expected 'bob', got '%s'", values[2])
	}

	// Test empty key
	indexName, values = ParseKey([]byte{})
	if indexName != "" {
		t.Error("empty key should return empty index name")
	}
	if len(values) != 0 {
		t.Error("empty key should return empty values")
	}
}

func TestNavigator_Triples(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Put(
		NewTripleFromStrings("alice", "knows", "bob"),
		NewTripleFromStrings("bob", "knows", "charlie"),
	)

	// Use Where to add a pattern that binds subject, predicate, object
	nav := db.Nav(nil).Where(&Pattern{
		Subject:   V("s"),
		Predicate: V("p"),
		Object:    V("o"),
	})
	// Materialize the solutions into triples using the variable names
	triples, err := nav.Triples(&Pattern{
		Subject:   V("s"),
		Predicate: V("p"),
		Object:    V("o"),
	})
	if err != nil {
		t.Fatalf("Triples failed: %v", err)
	}
	if len(triples) != 2 {
		t.Errorf("expected 2 triples, got %d", len(triples))
	}

	// Empty navigator
	emptyNav := &Navigator{db: db}
	triples, err = emptyNav.Triples(&Pattern{})
	if err != nil {
		t.Fatalf("empty Triples failed: %v", err)
	}
	if triples != nil {
		t.Error("empty navigator should return nil triples")
	}
}

func TestNavigator_Filter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Put(
		NewTripleFromStrings("alice", "age", "30"),
		NewTripleFromStrings("bob", "age", "25"),
	)

	nav := db.Nav(nil)
	nav.conditions = append(nav.conditions, &Pattern{Predicate: []byte("age"), Object: V("age")})
	nav = nav.Filter(func(t *Triple) bool {
		return string(t.Subject) == "alice"
	})

	solutions, err := nav.Solutions()
	if err != nil {
		t.Fatalf("Solutions failed: %v", err)
	}
	if len(solutions) != 1 {
		t.Errorf("expected 1 solution after filter, got %d", len(solutions))
	}
}

func TestNavigator_Where(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Put(NewTripleFromStrings("alice", "knows", "bob"))

	nav := db.Nav(nil).Where(&Pattern{
		Subject:   V("s"),
		Predicate: []byte("knows"),
		Object:    V("o"),
	})

	solutions, err := nav.Solutions()
	if err != nil {
		t.Fatalf("Solutions failed: %v", err)
	}
	if len(solutions) != 1 {
		t.Errorf("expected 1 solution, got %d", len(solutions))
	}
	if string(solutions[0]["s"]) != "alice" {
		t.Errorf("expected s='alice', got '%s'", solutions[0]["s"])
	}
	if string(solutions[0]["o"]) != "bob" {
		t.Errorf("expected o='bob', got '%s'", solutions[0]["o"])
	}
}

func TestWithSortJoin(t *testing.T) {
	dir, err := os.MkdirTemp("", "levelgraph-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	db, err := Open(filepath.Join(dir, "test.db"), WithSortJoin())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// SortJoin option should be set
	if db.options.JoinAlgorithm != "sort" {
		t.Errorf("expected 'sort' join algorithm, got '%s'", db.options.JoinAlgorithm)
	}
}

func TestJournalIterator_Key(t *testing.T) {
	dir, err := os.MkdirTemp("", "levelgraph-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	db, err := Open(filepath.Join(dir, "test.db"), WithJournal())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Add a triple to create journal entry
	db.Put(NewTripleFromStrings("a", "b", "c"))

	iter, err := db.GetJournalIterator(time.Time{})
	if err != nil {
		t.Fatalf("GetJournalIterator failed: %v", err)
	}
	defer iter.Close()

	if !iter.Next() {
		t.Fatal("expected at least one journal entry")
	}

	key := iter.Key()
	if key == nil {
		t.Error("Key should not be nil")
	}
	if len(key) == 0 {
		t.Error("Key should not be empty")
	}
}

func TestGetTripleFacetIterator(t *testing.T) {
	dir, err := os.MkdirTemp("", "levelgraph-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	db, err := Open(filepath.Join(dir, "test.db"), WithFacets())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	triple := NewTripleFromStrings("alice", "knows", "bob")
	db.Put(triple)
	db.SetTripleFacet(triple, []byte("since"), []byte("2020"))
	db.SetTripleFacet(triple, []byte("strength"), []byte("strong"))

	iter, err := db.GetTripleFacetIterator(triple)
	if err != nil {
		t.Fatalf("GetTripleFacetIterator failed: %v", err)
	}
	defer iter.Close()

	count := 0
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		if len(key) == 0 || value == nil {
			t.Error("key and value should not be empty")
		}
		count++
	}
	if err := iter.Error(); err != nil {
		t.Fatalf("Iterator error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 facets, got %d", count)
	}
}

func TestOpenWithDB(t *testing.T) {
	dir, err := os.MkdirTemp("", "levelgraph-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Open LevelDB directly
	ldb, err := leveldb.OpenFile(filepath.Join(dir, "test.db"), nil)
	if err != nil {
		t.Fatalf("failed to open LevelDB: %v", err)
	}

	// Wrap with LevelGraph
	db := OpenWithDB(ldb, WithJournal())
	if db == nil {
		t.Fatal("OpenWithDB should return a DB")
	}
	if !db.options.JournalEnabled {
		t.Error("options should be applied")
	}

	// Verify it works
	triple := NewTripleFromStrings("a", "b", "c")
	if err := db.Put(triple); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	results, err := db.Get(&Pattern{Subject: []byte("a")})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	db.Close()
}

func TestValidateTriple_EdgeCases(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Nil triple
	err := db.Put(nil)
	if err != ErrInvalidTriple {
		t.Errorf("expected ErrInvalidTriple for nil triple, got %v", err)
	}

	// Triple with nil subject
	err = db.Put(&Triple{Subject: nil, Predicate: []byte("p"), Object: []byte("o")})
	if err != ErrInvalidTriple {
		t.Errorf("expected ErrInvalidTriple for nil subject, got %v", err)
	}

	// Triple with nil predicate
	err = db.Put(&Triple{Subject: []byte("s"), Predicate: nil, Object: []byte("o")})
	if err != ErrInvalidTriple {
		t.Errorf("expected ErrInvalidTriple for nil predicate, got %v", err)
	}

	// Triple with nil object
	err = db.Put(&Triple{Subject: []byte("s"), Predicate: []byte("p"), Object: nil})
	if err != ErrInvalidTriple {
		t.Errorf("expected ErrInvalidTriple for nil object, got %v", err)
	}
}

func TestNewPattern_EdgeCases(t *testing.T) {
	// Empty byte slice treated as nil
	p := NewPattern([]byte{}, "pred", "obj")
	if p.Subject != nil {
		t.Error("empty []byte should be treated as nil")
	}

	// Empty string treated as nil
	p = NewPattern("", "pred", "obj")
	if p.Subject != nil {
		t.Error("empty string should be treated as nil")
	}

	// Bool values
	p = NewPattern(true, false, "obj")
	if string(p.Subject.([]byte)) != "true" {
		t.Errorf("expected 'true', got '%s'", p.Subject)
	}
	if string(p.Predicate.([]byte)) != "false" {
		t.Errorf("expected 'false', got '%s'", p.Predicate)
	}

	// Unknown type defaults to nil
	p = NewPattern(123, "pred", "obj")
	if p.Subject != nil {
		t.Error("unknown type should be treated as nil")
	}
}

func TestTriple_Equal_NilCases(t *testing.T) {
	triple := NewTripleFromStrings("a", "b", "c")

	// Comparing with nil
	if triple.Equal(nil) {
		t.Error("triple should not equal nil")
	}
}

func TestPattern_Matches_EdgeCases(t *testing.T) {
	triple := NewTripleFromStrings("alice", "knows", "bob")

	// Pattern with no concrete values matches everything
	p := &Pattern{}
	if !p.Matches(triple) {
		t.Error("empty pattern should match any triple")
	}

	// Pattern with subject only
	p = &Pattern{Subject: []byte("alice")}
	if !p.Matches(triple) {
		t.Error("pattern with matching subject should match")
	}

	p = &Pattern{Subject: []byte("charlie")}
	if p.Matches(triple) {
		t.Error("pattern with non-matching subject should not match")
	}

	// Pattern with predicate only
	p = &Pattern{Predicate: []byte("knows")}
	if !p.Matches(triple) {
		t.Error("pattern with matching predicate should match")
	}

	// Pattern with object only
	p = &Pattern{Object: []byte("bob")}
	if !p.Matches(triple) {
		t.Error("pattern with matching object should match")
	}

	p = &Pattern{Object: []byte("charlie")}
	if p.Matches(triple) {
		t.Error("pattern with non-matching object should not match")
	}
}

func TestNavigator_Go_EdgeCases(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Put(NewTripleFromStrings("alice", "knows", "bob"))

	// Go with nil creates a new variable
	nav := db.Nav(nil).Go(nil)
	if nav.lastElement == nil {
		t.Error("Go(nil) should create a new variable")
	}
	if _, ok := nav.lastElement.(*Variable); !ok {
		t.Error("Go(nil) should create a Variable")
	}

	// Go with []byte
	nav = db.Nav(nil).Go([]byte("test"))
	if b, ok := nav.lastElement.([]byte); !ok || string(b) != "test" {
		t.Error("Go([]byte) should set lastElement to that []byte")
	}

	// Go with string
	nav = db.Nav(nil).Go("test")
	if b, ok := nav.lastElement.([]byte); !ok || string(b) != "test" {
		t.Error("Go(string) should convert to []byte")
	}

	// Go with *Variable
	v := V("myvar")
	nav = db.Nav(nil).Go(v)
	if nav.lastElement != v {
		t.Error("Go(*Variable) should set lastElement to that Variable")
	}

	// Go with unknown type creates a new variable
	nav = db.Nav(nil).Go(123)
	if _, ok := nav.lastElement.(*Variable); !ok {
		t.Error("Go(unknown type) should create a new Variable")
	}
}

func TestNavigator_normalizeValue_EdgeCases(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.Put(NewTripleFromStrings("alice", "knows", "bob"))

	// Test ArchOut with empty []byte predicate - treated as nil/wildcard
	nav := db.Nav([]byte("alice")).ArchOut([]byte{})
	solutions, err := nav.Solutions()
	if err != nil {
		t.Fatalf("Solutions failed: %v", err)
	}
	// Empty predicate is normalized to nil, which means "any predicate"
	if len(solutions) != 1 {
		t.Errorf("expected 1 solution with nil predicate, got %d", len(solutions))
	}

	// Test ArchOut with empty string predicate
	nav = db.Nav([]byte("alice")).ArchOut("")
	solutions, err = nav.Solutions()
	if err != nil {
		t.Fatalf("Solutions failed: %v", err)
	}
	if len(solutions) != 1 {
		t.Errorf("expected 1 solution with empty string predicate, got %d", len(solutions))
	}
}

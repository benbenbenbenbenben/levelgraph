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

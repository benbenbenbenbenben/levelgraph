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
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func setupBenchDB(b *testing.B) (*DB, func()) {
	b.Helper()

	dir, err := os.MkdirTemp("", "levelgraph-bench-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(dir, "bench.db")
	db, err := Open(dbPath)
	if err != nil {
		os.RemoveAll(dir)
		b.Fatalf("failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(dir)
	}

	return db, cleanup
}

// BenchmarkPut measures single triple insertion performance.
func BenchmarkPut(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		triple := NewTripleFromStrings(
			fmt.Sprintf("subject%d", i),
			"predicate",
			fmt.Sprintf("object%d", i),
		)
		if err := db.Put(context.Background(), triple); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPutBatch measures batch triple insertion performance.
func BenchmarkPutBatch(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	// Prepare triples
	triples := make([]*Triple, 100)
	for i := 0; i < 100; i++ {
		triples[i] = NewTripleFromStrings(
			fmt.Sprintf("subject%d", i),
			"predicate",
			fmt.Sprintf("object%d", i),
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := db.Put(context.Background(), triples...); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGet measures query performance by subject.
func BenchmarkGet(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	// Insert some data
	for i := 0; i < 1000; i++ {
		triple := NewTripleFromStrings(
			fmt.Sprintf("subject%d", i%100),
			"predicate",
			fmt.Sprintf("object%d", i),
		)
		db.Put(context.Background(), triple)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subject := fmt.Sprintf("subject%d", i%100)
		_, err := db.Get(context.Background(), &Pattern{Subject: []byte(subject)})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetByPredicate measures query performance by predicate.
func BenchmarkGetByPredicate(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	// Insert some data
	predicates := []string{"knows", "likes", "follows", "works_with"}
	for i := 0; i < 1000; i++ {
		triple := NewTripleFromStrings(
			fmt.Sprintf("subject%d", i),
			predicates[i%len(predicates)],
			fmt.Sprintf("object%d", i),
		)
		db.Put(context.Background(), triple)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pred := predicates[i%len(predicates)]
		_, err := db.Get(context.Background(), &Pattern{Predicate: []byte(pred)})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSearch measures search/join performance.
func BenchmarkSearch(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	// Setup FOAF-like data
	for i := 0; i < 100; i++ {
		for j := 0; j < 5; j++ {
			triple := NewTripleFromStrings(
				fmt.Sprintf("person%d", i),
				"friend",
				fmt.Sprintf("person%d", (i+j+1)%100),
			)
			db.Put(context.Background(), triple)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.Search(context.Background(), []*Pattern{
			{
				Subject:   V("x"),
				Predicate: []byte("friend"),
				Object:    []byte("person50"),
			},
		}, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSearchJoin measures multi-pattern join performance.
func BenchmarkSearchJoin(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	// Setup FOAF-like data
	for i := 0; i < 100; i++ {
		for j := 0; j < 5; j++ {
			triple := NewTripleFromStrings(
				fmt.Sprintf("person%d", i),
				"friend",
				fmt.Sprintf("person%d", (i+j+1)%100),
			)
			db.Put(context.Background(), triple)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Find friends of friends of person0
		_, err := db.Search(context.Background(), []*Pattern{
			{
				Subject:   []byte("person0"),
				Predicate: []byte("friend"),
				Object:    V("x"),
			},
			{
				Subject:   V("x"),
				Predicate: []byte("friend"),
				Object:    V("y"),
			},
		}, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNavigator measures Navigator API performance.
func BenchmarkNavigator(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	// Setup FOAF-like data
	for i := 0; i < 100; i++ {
		for j := 0; j < 5; j++ {
			triple := NewTripleFromStrings(
				fmt.Sprintf("person%d", i),
				"friend",
				fmt.Sprintf("person%d", (i+j+1)%100),
			)
			db.Put(context.Background(), triple)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.Nav(context.Background(), "person0").
			ArchOut("friend").
			ArchOut("friend").
			Values()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDel measures deletion performance.
func BenchmarkDel(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	// Pre-insert triples
	triples := make([]*Triple, b.N)
	for i := 0; i < b.N; i++ {
		triples[i] = NewTripleFromStrings(
			fmt.Sprintf("subject%d", i),
			"predicate",
			fmt.Sprintf("object%d", i),
		)
		db.Put(context.Background(), triples[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := db.Del(context.Background(), triples[i]); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIterator measures iterator performance.
func BenchmarkIterator(b *testing.B) {
	db, cleanup := setupBenchDB(b)
	defer cleanup()

	// Insert some data
	for i := 0; i < 1000; i++ {
		triple := NewTripleFromStrings(
			"subject",
			"predicate",
			fmt.Sprintf("object%d", i),
		)
		db.Put(context.Background(), triple)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter, err := db.GetIterator(context.Background(), &Pattern{Subject: []byte("subject")})
		if err != nil {
			b.Fatal(err)
		}
		count := 0
		for iter.Next() {
			count++
		}
		iter.Release()
	}
}

// BenchmarkJournalPut measures Put performance with journal enabled.
func BenchmarkJournalPut(b *testing.B) {
	dir, err := os.MkdirTemp("", "levelgraph-bench-journal-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "bench.db")
	db, err := Open(dbPath, WithJournal())
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		triple := NewTripleFromStrings(
			fmt.Sprintf("subject%d", i),
			"predicate",
			fmt.Sprintf("object%d", i),
		)
		if err := db.Put(context.Background(), triple); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFacetSet measures facet set performance.
func BenchmarkFacetSet(b *testing.B) {
	dir, err := os.MkdirTemp("", "levelgraph-bench-facet-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "bench.db")
	db, err := Open(dbPath, WithFacets())
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := db.SetFacet(context.Background(), FacetSubject, []byte(fmt.Sprintf("subject%d", i)), []byte("key"), []byte("value"))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFacetGet measures facet get performance.
func BenchmarkFacetGet(b *testing.B) {
	dir, err := os.MkdirTemp("", "levelgraph-bench-facet-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "bench.db")
	db, err := Open(dbPath, WithFacets())
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Pre-set facets
	for i := 0; i < 1000; i++ {
		db.SetFacet(context.Background(), FacetSubject, []byte(fmt.Sprintf("subject%d", i)), []byte("key"), []byte("value"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetFacet(context.Background(), FacetSubject, []byte(fmt.Sprintf("subject%d", i%1000)), []byte("key"))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEscape measures escape function performance.
func BenchmarkEscape(b *testing.B) {
	value := []byte("subject::with::many::colons::and\\backslashes")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Escape(value)
	}
}

// BenchmarkGenKey measures key generation performance.
func BenchmarkGenKey(b *testing.B) {
	triple := NewTripleFromStrings("subject", "predicate", "object")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenKey(IndexSPO, triple)
	}
}

// BenchmarkGenKeys measures full key generation performance (all 6 indexes).
func BenchmarkGenKeys(b *testing.B) {
	triple := NewTripleFromStrings("subject", "predicate", "object")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenKeys(triple)
	}
}

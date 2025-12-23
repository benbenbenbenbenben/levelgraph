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

package levelgraph_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/benbenbenbenbenben/levelgraph"
	"github.com/benbenbenbenbenben/levelgraph/pkg/graph"
	"github.com/benbenbenbenbenben/levelgraph/vector"
)

// Example demonstrates basic LevelGraph usage: opening a database,
// storing triples, and querying them.
func Example() {
	// Create a temporary directory for the database
	dir, err := os.MkdirTemp("", "levelgraph-example")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer os.RemoveAll(dir)

	// Open the database
	db, err := levelgraph.Open(filepath.Join(dir, "example.db"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer db.Close()

	// Insert triples
	err = db.Put(context.Background(),
		graph.NewTripleFromStrings("alice", "knows", "bob"),
		graph.NewTripleFromStrings("bob", "knows", "charlie"),
	)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Query by subject
	triples, err := db.Get(context.Background(), &graph.Pattern{Subject: graph.ExactString("alice")})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, t := range triples {
		fmt.Printf("%s %s %s\n", t.Subject, t.Predicate, t.Object)
	}
	// Output: alice knows bob
}

// Example_search demonstrates using Search with variables to find patterns.
func Example_search() {
	dir, err := os.MkdirTemp("", "levelgraph-search")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer os.RemoveAll(dir)

	db, err := levelgraph.Open(filepath.Join(dir, "search.db"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer db.Close()

	// Build a social graph
	db.Put(context.Background(),
		graph.NewTripleFromStrings("alice", "knows", "bob"),
		graph.NewTripleFromStrings("bob", "knows", "charlie"),
		graph.NewTripleFromStrings("alice", "knows", "dave"),
	)

	// Find everyone alice knows
	results, err := db.Search(context.Background(), []*graph.Pattern{
		{
			Subject:   graph.ExactString("alice"),
			Predicate: graph.ExactString("knows"),
			Object:    graph.Binding("friend"),
		},
	}, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Alice knows %d people\n", len(results))
	// Output: Alice knows 2 people
}

// Example_navigator demonstrates the fluent Navigator API for graph traversal.
func Example_navigator() {
	dir, err := os.MkdirTemp("", "levelgraph-nav")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer os.RemoveAll(dir)

	db, err := levelgraph.Open(filepath.Join(dir, "nav.db"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer db.Close()

	// Build a graph
	db.Put(context.Background(),
		graph.NewTripleFromStrings("alice", "knows", "bob"),
		graph.NewTripleFromStrings("bob", "knows", "charlie"),
	)

	// Navigate: find friends of friends of alice
	solutions, err := db.Nav(context.Background(), "alice").
		ArchOut("knows").As("friend").
		ArchOut("knows").As("fof").
		Solutions()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if len(solutions) > 0 {
		fmt.Printf("Friend of friend: %s\n", solutions[0]["fof"])
	}
	// Output: Friend of friend: charlie
}

// Example_facets demonstrates attaching metadata to triples and their components.
func Example_facets() {
	dir, err := os.MkdirTemp("", "levelgraph-facets")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer os.RemoveAll(dir)

	// Open with facets enabled
	db, err := levelgraph.Open(filepath.Join(dir, "facets.db"), levelgraph.WithFacets())
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer db.Close()

	ctx := context.Background()

	// Add a triple
	triple := graph.NewTripleFromStrings("alice", "knows", "bob")
	db.Put(ctx, triple)

	// Add facet to the triple itself (relationship metadata)
	err = db.SetTripleFacet(ctx, triple, []byte("since"), []byte("2020"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Add facet to a subject (entity metadata)
	err = db.SetFacet(ctx, levelgraph.FacetSubject, []byte("alice"), []byte("age"), []byte("30"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Retrieve the facets
	since, err := db.GetTripleFacet(ctx, triple, []byte("since"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	age, err := db.GetFacet(ctx, levelgraph.FacetSubject, []byte("alice"), []byte("age"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Alice (age %s) knows Bob since %s\n", age, since)
	// Output: Alice (age 30) knows Bob since 2020
}

// Example_journal demonstrates journaling for audit trails.
func Example_journal() {
	dir, err := os.MkdirTemp("", "levelgraph-journal")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer os.RemoveAll(dir)

	// Open with journaling enabled
	db, err := levelgraph.Open(filepath.Join(dir, "journal.db"), levelgraph.WithJournal())
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer db.Close()

	ctx := context.Background()

	// Perform some operations
	db.Put(ctx, graph.NewTripleFromStrings("alice", "knows", "bob"))
	db.Put(ctx, graph.NewTripleFromStrings("bob", "knows", "charlie"))
	db.Del(ctx, graph.NewTripleFromStrings("alice", "knows", "bob"))

	// Get all journal entries (use zero time to get all)
	count, err := db.JournalCount(ctx, time.Time{})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Journal has %d entries\n", count)
	// Output: Journal has 3 entries
}

// Example_searchJoin demonstrates multi-pattern joins to find complex relationships.
func Example_searchJoin() {
	dir, err := os.MkdirTemp("", "levelgraph-join")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer os.RemoveAll(dir)

	db, err := levelgraph.Open(filepath.Join(dir, "join.db"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer db.Close()

	ctx := context.Background()

	// Build a graph of people and their interests
	db.Put(ctx,
		graph.NewTripleFromStrings("alice", "likes", "tennis"),
		graph.NewTripleFromStrings("alice", "likes", "programming"),
		graph.NewTripleFromStrings("bob", "likes", "tennis"),
		graph.NewTripleFromStrings("bob", "likes", "chess"),
		graph.NewTripleFromStrings("charlie", "likes", "programming"),
	)

	// Find what alice and bob have in common (join on shared interest)
	results, err := db.Search(ctx, []*graph.Pattern{
		{
			Subject:   graph.ExactString("alice"),
			Predicate: graph.ExactString("likes"),
			Object:    graph.Binding("interest"),
		},
		{
			Subject:   graph.ExactString("bob"),
			Predicate: graph.ExactString("likes"),
			Object:    graph.Binding("interest"), // Same variable binds shared value
		},
	}, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if len(results) > 0 {
		fmt.Printf("Alice and Bob both like: %s\n", results[0]["interest"])
	}
	// Output: Alice and Bob both like: tennis
}

// Example_vectorSearch demonstrates semantic similarity search using vector embeddings.
// This enables "fuzzy" queries that find results based on meaning rather than exact matches.
func Example_vectorSearch() {
	dir, err := os.MkdirTemp("", "levelgraph-example-vector")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer os.RemoveAll(dir)

	// Create a vector index (3 dimensions for this simple example)
	vectorIndex := vector.NewFlatIndex(3)

	db, err := levelgraph.Open(filepath.Join(dir, "vector.db"),
		levelgraph.WithVectors(vectorIndex),
	)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer db.Close()

	ctx := context.Background()

	// Add triples about sports
	db.Put(ctx,
		graph.NewTripleFromStrings("alice", "likes", "tennis"),
		graph.NewTripleFromStrings("bob", "likes", "badminton"),
		graph.NewTripleFromStrings("charlie", "likes", "football"),
	)

	// Associate vectors with sport objects (simulated embeddings)
	// In practice, these would come from an embedding model
	// Tennis and badminton are similar (both racket sports)
	db.SetObjectVector(ctx, []byte("tennis"), []float32{0.9, 0.1, 0.0})
	db.SetObjectVector(ctx, []byte("badminton"), []float32{0.85, 0.15, 0.0})
	db.SetObjectVector(ctx, []byte("football"), []float32{0.1, 0.9, 0.0})

	// Search for sports similar to tennis
	results, err := db.SearchSimilarObjects(ctx, []float32{0.9, 0.1, 0.0}, 3)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Sports similar to tennis:")
	for _, match := range results {
		fmt.Printf("  %s (score: %.2f)\n", match.Parts[0], match.Score)
	}
	// Output:
	// Sports similar to tennis:
	//   tennis (score: 1.00)
	//   badminton (score: 1.00)
	//   football (score: 0.61)
}

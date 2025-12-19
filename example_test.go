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
	"fmt"
	"os"
	"path/filepath"

	"github.com/levelgraph/levelgraph"
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
	err = db.Put(
		levelgraph.NewTripleFromStrings("alice", "knows", "bob"),
		levelgraph.NewTripleFromStrings("bob", "knows", "charlie"),
	)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Query by subject
	triples, err := db.Get(&levelgraph.Pattern{Subject: []byte("alice")})
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
	db.Put(
		levelgraph.NewTripleFromStrings("alice", "knows", "bob"),
		levelgraph.NewTripleFromStrings("bob", "knows", "charlie"),
		levelgraph.NewTripleFromStrings("alice", "knows", "dave"),
	)

	// Find everyone alice knows
	results, err := db.Search([]*levelgraph.Pattern{
		{
			Subject:   []byte("alice"),
			Predicate: []byte("knows"),
			Object:    levelgraph.V("friend"),
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
	db.Put(
		levelgraph.NewTripleFromStrings("alice", "knows", "bob"),
		levelgraph.NewTripleFromStrings("bob", "knows", "charlie"),
	)

	// Navigate: find friends of friends of alice
	solutions, err := db.Nav("alice").
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

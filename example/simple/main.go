// Package main demonstrates basic LevelGraph features.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/benbenbenbenbenben/levelgraph"
)

func main() {
	// Create a temporary directory for our database
	dbPath := "./simple.db"
	defer os.RemoveAll(dbPath)

	// Open a LevelGraph database with journaling and facets enabled
	db, err := levelgraph.Open(dbPath,
		levelgraph.WithJournal(),
		levelgraph.WithFacets(),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== Basic Triple Operations ===")

	// Add some simple triples
	err = db.Put(context.Background(),
		levelgraph.NewTripleFromStrings("alice", "knows", "bob"),
		levelgraph.NewTripleFromStrings("bob", "knows", "alice"),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Added: alice knows bob")
	fmt.Println("Added: bob knows alice")

	// Query by subject
	results, err := db.Get(context.Background(), &levelgraph.Pattern{
		Subject: []byte("alice"),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nAlice knows: ")
	for _, t := range results {
		fmt.Printf("%s ", t.Object)
	}
	fmt.Println()

	fmt.Println("\n=== Journal Feature ===")

	// Query the journal for recent operations
	oneHourAgo := time.Now().Add(-time.Hour)
	entries, err := db.GetJournalEntries(context.Background(), oneHourAgo)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d journal entries from the last hour:\n", len(entries))
	for _, entry := range entries {
		fmt.Printf("  [%s] %s: %s %s %s\n",
			entry.Timestamp.Format("15:04:05"),
			entry.Operation,
			entry.Triple.Subject,
			entry.Triple.Predicate,
			entry.Triple.Object,
		)
	}

	fmt.Println("\n=== Facets Feature ===")

	// Create a triple with facets (metadata)
	triple := levelgraph.NewTripleFromStrings("alice", "knows", "charlie")
	err = db.Put(context.Background(), triple)
	if err != nil {
		log.Fatal(err)
	}

	// Add facets to the triple
	err = db.SetTripleFacet(context.Background(), triple, []byte("since"), []byte("2023"))
	if err != nil {
		log.Fatal(err)
	}
	err = db.SetTripleFacet(context.Background(), triple, []byte("trust"), []byte("high"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Added: alice knows charlie (with facets: since=2023, trust=high)")

	// Retrieve facets
	facets, err := db.GetTripleFacets(context.Background(), triple)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Facets on alice-knows-charlie:")
	for k, v := range facets {
		fmt.Printf("  %s = %s\n", k, v)
	}

	fmt.Println("\n=== Navigator API ===")

	// Add more data for navigation
	db.Put(context.Background(),
		levelgraph.NewTripleFromStrings("bob", "knows", "charlie"),
		levelgraph.NewTripleFromStrings("charlie", "knows", "diana"),
	)

	// Use the Navigator API for graph traversal
	// Find all people that alice knows, and who they know
	solutions, err := db.Nav(context.Background(), []byte("alice")).
		ArchOut([]byte("knows")).As("friend").
		ArchOut([]byte("knows")).As("friendOfFriend").
		Solutions()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Friends of Alice's friends:")
	for _, sol := range solutions {
		fmt.Printf("  alice → %s → %s\n", sol["friend"], sol["friendOfFriend"])
	}

	fmt.Println("\n=== Search with Variables ===")

	// Search for all "knows" relationships using variables
	x := levelgraph.V("x")
	y := levelgraph.V("y")
	searchResults, err := db.Search(context.Background(), []*levelgraph.Pattern{
		{Subject: x, Predicate: []byte("knows"), Object: y},
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("All 'knows' relationships:")
	for _, sol := range searchResults {
		fmt.Printf("  %s knows %s\n", sol["x"], sol["y"])
	}

	fmt.Println("\nDone!")
}

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
	"testing"

	"github.com/benbenbenbenbenben/levelgraph/pkg/graph"
)

// setupSocialGraph creates a social network graph for testing complex queries
func setupSocialGraph(t *testing.T) (*DB, func()) {
	t.Helper()
	db, cleanup := setupTestDB(t)

	triples := []*graph.Triple{
		// People and their types
		graph.NewTripleFromStrings("alice", "type", "Person"),
		graph.NewTripleFromStrings("bob", "type", "Person"),
		graph.NewTripleFromStrings("charlie", "type", "Person"),
		graph.NewTripleFromStrings("diana", "type", "Person"),
		graph.NewTripleFromStrings("eve", "type", "Person"),

		// Friendships (directed - alice knows bob doesn't mean bob knows alice)
		graph.NewTripleFromStrings("alice", "knows", "bob"),
		graph.NewTripleFromStrings("alice", "knows", "charlie"),
		graph.NewTripleFromStrings("bob", "knows", "charlie"),
		graph.NewTripleFromStrings("bob", "knows", "diana"),
		graph.NewTripleFromStrings("charlie", "knows", "diana"),
		graph.NewTripleFromStrings("diana", "knows", "eve"),

		// Interests
		graph.NewTripleFromStrings("alice", "likes", "hiking"),
		graph.NewTripleFromStrings("alice", "likes", "photography"),
		graph.NewTripleFromStrings("bob", "likes", "hiking"),
		graph.NewTripleFromStrings("bob", "likes", "coding"),
		graph.NewTripleFromStrings("charlie", "likes", "music"),
		graph.NewTripleFromStrings("charlie", "likes", "photography"),
		graph.NewTripleFromStrings("diana", "likes", "hiking"),
		graph.NewTripleFromStrings("diana", "likes", "music"),
		graph.NewTripleFromStrings("eve", "likes", "coding"),

		// Ages
		graph.NewTripleFromStrings("alice", "age", "30"),
		graph.NewTripleFromStrings("bob", "age", "25"),
		graph.NewTripleFromStrings("charlie", "age", "35"),
		graph.NewTripleFromStrings("diana", "age", "28"),
		graph.NewTripleFromStrings("eve", "age", "22"),

		// Locations
		graph.NewTripleFromStrings("alice", "livesIn", "NYC"),
		graph.NewTripleFromStrings("bob", "livesIn", "NYC"),
		graph.NewTripleFromStrings("charlie", "livesIn", "LA"),
		graph.NewTripleFromStrings("diana", "livesIn", "LA"),
		graph.NewTripleFromStrings("eve", "livesIn", "NYC"),
	}

	if err := db.Put(context.Background(), triples...); err != nil {
		cleanup()
		t.Fatalf("failed to setup social graph: %v", err)
	}

	return db, cleanup
}

// TestComplexQuery_SharedInterests finds people who share interests
func TestComplexQuery_SharedInterests(t *testing.T) {
	t.Parallel()
	db, cleanup := setupSocialGraph(t)
	defer cleanup()

	t.Run("find all pairs sharing an interest", func(t *testing.T) {
		// Find all pairs of people who share at least one interest
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("person1"), Predicate: graph.ExactString("likes"), Object: graph.Binding("interest")},
			{Subject: graph.Binding("person2"), Predicate: graph.ExactString("likes"), Object: graph.Binding("interest")},
		}, &SearchOptions{
			Filter: func(s graph.Solution) bool {
				// Exclude self-pairs
				return string(s["person1"]) != string(s["person2"])
			},
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Count unique pairs (person1, person2, interest)
		// Expected: alice-bob (hiking), alice-charlie (photography), alice-diana (hiking),
		// bob-diana (hiking), charlie-diana (music), bob-eve (coding)
		// Plus reverse pairs
		if len(results) < 6 {
			t.Errorf("expected at least 6 shared interest pairs, got %d", len(results))
		}

		// graph.Verify structure
		for _, sol := range results {
			if sol["person1"] == nil || sol["person2"] == nil || sol["interest"] == nil {
				t.Error("solution should have person1, person2, and interest")
			}
		}
	})

	t.Run("find people who share alice's interests", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.ExactString("alice"), Predicate: graph.ExactString("likes"), Object: graph.Binding("interest")},
			{Subject: graph.Binding("other"), Predicate: graph.ExactString("likes"), Object: graph.Binding("interest")},
		}, &SearchOptions{
			Filter: func(s graph.Solution) bool {
				return string(s["other"]) != "alice"
			},
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Alice likes hiking and photography
		// Others who like hiking: bob, diana
		// Others who like photography: charlie
		others := make(map[string]bool)
		for _, sol := range results {
			others[string(sol["other"])] = true
		}

		if !others["bob"] || !others["charlie"] || !others["diana"] {
			t.Errorf("expected bob, charlie, diana to share interests with alice, got %v", others)
		}
		if others["eve"] {
			t.Error("eve should not share interests with alice")
		}
	})
}

// TestComplexQuery_FriendsOfFriends finds friends at various depths
func TestComplexQuery_FriendsOfFriends(t *testing.T) {
	t.Parallel()
	db, cleanup := setupSocialGraph(t)
	defer cleanup()

	t.Run("direct friends of alice", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.ExactString("alice"), Predicate: graph.ExactString("knows"), Object: graph.Binding("friend")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		friends := make(map[string]bool)
		for _, sol := range results {
			friends[string(sol["friend"])] = true
		}

		if !friends["bob"] || !friends["charlie"] {
			t.Errorf("expected bob and charlie as direct friends, got %v", friends)
		}
		if len(friends) != 2 {
			t.Errorf("expected 2 direct friends, got %d", len(friends))
		}
	})

	t.Run("friends of friends of alice", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.ExactString("alice"), Predicate: graph.ExactString("knows"), Object: graph.Binding("friend")},
			{Subject: graph.Binding("friend"), Predicate: graph.ExactString("knows"), Object: graph.Binding("fof")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		fofs := make(map[string]bool)
		for _, sol := range results {
			fofs[string(sol["fof"])] = true
		}

		// alice -> bob -> charlie, diana
		// alice -> charlie -> diana
		if !fofs["charlie"] || !fofs["diana"] {
			t.Errorf("expected charlie and diana as friends-of-friends, got %v", fofs)
		}
	})

	t.Run("friends of friends excluding direct friends", func(t *testing.T) {
		// First get direct friends
		directResults, _ := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.ExactString("alice"), Predicate: graph.ExactString("knows"), Object: graph.Binding("friend")},
		}, nil)
		directFriends := make(map[string]bool)
		for _, sol := range directResults {
			directFriends[string(sol["friend"])] = true
		}

		// Get friends of friends
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.ExactString("alice"), Predicate: graph.ExactString("knows"), Object: graph.Binding("friend")},
			{Subject: graph.Binding("friend"), Predicate: graph.ExactString("knows"), Object: graph.Binding("fof")},
		}, &SearchOptions{
			Filter: func(s graph.Solution) bool {
				fof := string(s["fof"])
				// Exclude alice and direct friends
				return fof != "alice" && !directFriends[fof]
			},
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		fofs := make(map[string]bool)
		for _, sol := range results {
			fofs[string(sol["fof"])] = true
		}

		// diana is a friend-of-friend but not a direct friend
		if !fofs["diana"] {
			t.Error("diana should be in friends-of-friends")
		}
		if fofs["bob"] || fofs["charlie"] {
			t.Error("direct friends should be excluded")
		}
	})

	t.Run("three degrees of separation", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.ExactString("alice"), Predicate: graph.ExactString("knows"), Object: graph.Binding("f1")},
			{Subject: graph.Binding("f1"), Predicate: graph.ExactString("knows"), Object: graph.Binding("f2")},
			{Subject: graph.Binding("f2"), Predicate: graph.ExactString("knows"), Object: graph.Binding("f3")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Path: alice -> bob -> diana -> eve
		// Path: alice -> charlie -> diana -> eve
		// Path: alice -> bob -> charlie -> diana
		found := make(map[string]bool)
		for _, sol := range results {
			found[string(sol["f3"])] = true
		}

		if !found["eve"] {
			t.Error("should find eve at 3 degrees of separation")
		}
	})
}

// TestComplexQuery_TriangleFinding finds triangles (cycles of length 3)
func TestComplexQuery_TriangleFinding(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a graph with known triangles
	triples := []*graph.Triple{
		// Triangle: a-b-c-a
		graph.NewTripleFromStrings("a", "connected", "b"),
		graph.NewTripleFromStrings("b", "connected", "c"),
		graph.NewTripleFromStrings("c", "connected", "a"),
		// Additional edges
		graph.NewTripleFromStrings("a", "connected", "d"),
		graph.NewTripleFromStrings("d", "connected", "e"),
	}
	db.Put(context.Background(), triples...)

	results, err := db.Search(context.Background(), []*graph.Pattern{
		{Subject: graph.Binding("x"), Predicate: graph.ExactString("connected"), Object: graph.Binding("y")},
		{Subject: graph.Binding("y"), Predicate: graph.ExactString("connected"), Object: graph.Binding("z")},
		{Subject: graph.Binding("z"), Predicate: graph.ExactString("connected"), Object: graph.Binding("x")},
	}, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// The search finds all 3 rotations of the triangle: (a,b,c), (b,c,a), (c,a,b)
	// This is correct behavior - each starting point is a valid match
	if len(results) != 3 {
		t.Errorf("expected 3 triangle rotations, got %d", len(results))
	}

	// graph.Verify all results form valid triangles
	triangleNodes := map[string]bool{"a": true, "b": true, "c": true}
	for i, r := range results {
		x := string(r["x"])
		y := string(r["y"])
		z := string(r["z"])
		if !triangleNodes[x] || !triangleNodes[y] || !triangleNodes[z] {
			t.Errorf("result %d: unexpected nodes %s-%s-%s", i, x, y, z)
		}
		// graph.Verify they're all different (no duplicates in the same match)
		if x == y || y == z || x == z {
			t.Errorf("result %d: duplicate nodes in triangle %s-%s-%s", i, x, y, z)
		}
	}
}

// TestComplexQuery_PathExistence checks if paths exist between nodes
func TestComplexQuery_PathExistence(t *testing.T) {
	t.Parallel()
	db, cleanup := setupSocialGraph(t)
	defer cleanup()

	t.Run("direct path exists", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.ExactString("alice"), Predicate: graph.ExactString("knows"), Object: graph.ExactString("bob")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) != 1 {
			t.Error("direct path alice->bob should exist")
		}
	})

	t.Run("two-hop path exists", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.ExactString("alice"), Predicate: graph.ExactString("knows"), Object: graph.Binding("mid")},
			{Subject: graph.Binding("mid"), Predicate: graph.ExactString("knows"), Object: graph.ExactString("diana")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		// alice -> bob -> diana and alice -> charlie -> diana
		if len(results) != 2 {
			t.Errorf("expected 2 paths alice->?->diana, got %d", len(results))
		}
	})

	t.Run("path does not exist", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.ExactString("eve"), Predicate: graph.ExactString("knows"), Object: graph.Binding("anyone")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		// eve doesn't know anyone in our graph
		if len(results) != 0 {
			t.Errorf("eve should not know anyone, got %d results", len(results))
		}
	})
}

// TestComplexQuery_MultiplePredicates queries involving multiple predicate types
func TestComplexQuery_MultiplePredicates(t *testing.T) {
	t.Parallel()
	db, cleanup := setupSocialGraph(t)
	defer cleanup()

	t.Run("friends in same city", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("person1"), Predicate: graph.ExactString("knows"), Object: graph.Binding("person2")},
			{Subject: graph.Binding("person1"), Predicate: graph.ExactString("livesIn"), Object: graph.Binding("city")},
			{Subject: graph.Binding("person2"), Predicate: graph.ExactString("livesIn"), Object: graph.Binding("city")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// alice knows bob, both in NYC
		// charlie knows diana, both in LA
		found := false
		for _, sol := range results {
			p1, p2 := string(sol["person1"]), string(sol["person2"])
			if (p1 == "alice" && p2 == "bob") || (p1 == "charlie" && p2 == "diana") {
				found = true
			}
		}
		if !found {
			t.Error("should find friends in same city")
		}
	})

	t.Run("people who like what their friends like", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("person"), Predicate: graph.ExactString("knows"), Object: graph.Binding("friend")},
			{Subject: graph.Binding("person"), Predicate: graph.ExactString("likes"), Object: graph.Binding("interest")},
			{Subject: graph.Binding("friend"), Predicate: graph.ExactString("likes"), Object: graph.Binding("interest")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Find cases where person and friend share an interest
		if len(results) == 0 {
			t.Error("should find people who share interests with friends")
		}

		// graph.Verify: alice knows bob, both like hiking
		found := false
		for _, sol := range results {
			if string(sol["person"]) == "alice" && string(sol["friend"]) == "bob" && string(sol["interest"]) == "hiking" {
				found = true
			}
		}
		if !found {
			t.Error("should find alice-bob-hiking")
		}
	})
}

// TestComplexQuery_Aggregation simulates aggregation by counting in Go
func TestComplexQuery_Aggregation(t *testing.T) {
	t.Parallel()
	db, cleanup := setupSocialGraph(t)
	defer cleanup()

	t.Run("count friends per person", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("person"), Predicate: graph.ExactString("knows"), Object: graph.Binding("friend")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Count friends per person
		friendCount := make(map[string]int)
		for _, sol := range results {
			person := string(sol["person"])
			friendCount[person]++
		}

		// graph.Verify counts
		if friendCount["alice"] != 2 {
			t.Errorf("alice should have 2 friends, got %d", friendCount["alice"])
		}
		if friendCount["bob"] != 2 {
			t.Errorf("bob should have 2 friends, got %d", friendCount["bob"])
		}
		if friendCount["diana"] != 1 {
			t.Errorf("diana should have 1 friend, got %d", friendCount["diana"])
		}
	})

	t.Run("count interests per person", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("person"), Predicate: graph.ExactString("likes"), Object: graph.Binding("interest")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		interestCount := make(map[string]int)
		for _, sol := range results {
			person := string(sol["person"])
			interestCount[person]++
		}

		// All people have 2 interests except eve (1)
		for _, person := range []string{"alice", "bob", "charlie", "diana"} {
			if interestCount[person] != 2 {
				t.Errorf("%s should have 2 interests, got %d", person, interestCount[person])
			}
		}
		if interestCount["eve"] != 1 {
			t.Errorf("eve should have 1 interest, got %d", interestCount["eve"])
		}
	})
}

// TestComplexQuery_OptionalPatterns simulates OPTIONAL patterns using union
func TestComplexQuery_OptionalPatterns(t *testing.T) {
	t.Parallel()
	db, cleanup := setupSocialGraph(t)
	defer cleanup()

	t.Run("find people with optional age", func(t *testing.T) {
		// Get all people
		peopleResults, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("person"), Predicate: graph.ExactString("type"), Object: graph.ExactString("Person")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// For each person, try to get their age
		type PersonWithAge struct {
			Name string
			Age  string
		}
		var results []PersonWithAge

		for _, sol := range peopleResults {
			person := string(sol["person"])
			ageResults, _ := db.Search(context.Background(), []*graph.Pattern{
				{Subject: graph.Exact([]byte(person)), Predicate: graph.ExactString("age"), Object: graph.Binding("age")},
			}, nil)

			pwa := PersonWithAge{Name: person}
			if len(ageResults) > 0 {
				pwa.Age = string(ageResults[0]["age"])
			}
			results = append(results, pwa)
		}

		if len(results) != 5 {
			t.Errorf("expected 5 people, got %d", len(results))
		}

		// All should have ages in our test data
		for _, r := range results {
			if r.Age == "" {
				t.Errorf("%s should have an age", r.Name)
			}
		}
	})
}

// TestComplexQuery_Negation simulates NOT EXISTS patterns
func TestComplexQuery_Negation(t *testing.T) {
	t.Parallel()
	db, cleanup := setupSocialGraph(t)
	defer cleanup()

	t.Run("find people who don't like hiking", func(t *testing.T) {
		// Get all people
		allPeople, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("person"), Predicate: graph.ExactString("type"), Object: graph.ExactString("Person")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Get people who like hiking
		hikingLovers, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("person"), Predicate: graph.ExactString("likes"), Object: graph.ExactString("hiking")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		hikingSet := make(map[string]bool)
		for _, sol := range hikingLovers {
			hikingSet[string(sol["person"])] = true
		}

		// Find people NOT in hikingSet
		var nonHikers []string
		for _, sol := range allPeople {
			person := string(sol["person"])
			if !hikingSet[person] {
				nonHikers = append(nonHikers, person)
			}
		}

		// charlie and eve don't like hiking
		if len(nonHikers) != 2 {
			t.Errorf("expected 2 non-hikers, got %d: %v", len(nonHikers), nonHikers)
		}
	})

	t.Run("find people with no friends", func(t *testing.T) {
		// Get all people
		allPeople, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("person"), Predicate: graph.ExactString("type"), Object: graph.ExactString("Person")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Get people who know someone
		peopleWithFriends, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("person"), Predicate: graph.ExactString("knows"), Object: graph.Binding("_")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		friendlySet := make(map[string]bool)
		for _, sol := range peopleWithFriends {
			friendlySet[string(sol["person"])] = true
		}

		var loners []string
		for _, sol := range allPeople {
			person := string(sol["person"])
			if !friendlySet[person] {
				loners = append(loners, person)
			}
		}

		// eve knows no one
		if len(loners) != 1 || loners[0] != "eve" {
			t.Errorf("expected only eve as loner, got %v", loners)
		}
	})
}

// TestComplexQuery_SelfJoin tests joining a pattern with itself
func TestComplexQuery_SelfJoin(t *testing.T) {
	t.Parallel()
	db, cleanup := setupSocialGraph(t)
	defer cleanup()

	t.Run("find mutual friends (both know each other)", func(t *testing.T) {
		// Add some mutual friendships
		db.Put(context.Background(),
			graph.NewTripleFromStrings("bob", "knows", "alice"), // alice already knows bob
		)

		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("a"), Predicate: graph.ExactString("knows"), Object: graph.Binding("b")},
			{Subject: graph.Binding("b"), Predicate: graph.ExactString("knows"), Object: graph.Binding("a")},
		}, &SearchOptions{
			Filter: func(s graph.Solution) bool {
				// Avoid duplicates (a,b) and (b,a)
				return string(s["a"]) < string(s["b"])
			},
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// alice and bob are mutual friends
		if len(results) != 1 {
			t.Errorf("expected 1 mutual friendship, got %d", len(results))
		}
	})
}

// TestComplexQuery_PropertyPaths tests property path-like queries
func TestComplexQuery_PropertyPaths(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a taxonomy
	triples := []*graph.Triple{
		graph.NewTripleFromStrings("dog", "subclassOf", "mammal"),
		graph.NewTripleFromStrings("cat", "subclassOf", "mammal"),
		graph.NewTripleFromStrings("mammal", "subclassOf", "animal"),
		graph.NewTripleFromStrings("animal", "subclassOf", "livingThing"),
		graph.NewTripleFromStrings("plant", "subclassOf", "livingThing"),
	}
	db.Put(context.Background(), triples...)

	t.Run("direct subclass", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("x"), Predicate: graph.ExactString("subclassOf"), Object: graph.ExactString("mammal")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		classes := make(map[string]bool)
		for _, sol := range results {
			classes[string(sol["x"])] = true
		}

		if !classes["dog"] || !classes["cat"] {
			t.Error("dog and cat should be direct subclasses of mammal")
		}
		if len(classes) != 2 {
			t.Errorf("expected 2 direct subclasses, got %d", len(classes))
		}
	})

	t.Run("two-level subclass", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("x"), Predicate: graph.ExactString("subclassOf"), Object: graph.Binding("y")},
			{Subject: graph.Binding("y"), Predicate: graph.ExactString("subclassOf"), Object: graph.ExactString("animal")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// dog and cat are 2 levels below animal
		classes := make(map[string]bool)
		for _, sol := range results {
			classes[string(sol["x"])] = true
		}

		if !classes["dog"] || !classes["cat"] {
			t.Error("dog and cat should be 2-level subclasses of animal")
		}
	})
}

// TestComplexQuery_DiamondPattern tests the diamond join pattern
func TestComplexQuery_DiamondPattern(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a diamond pattern: a -> b -> d, a -> c -> d
	triples := []*graph.Triple{
		graph.NewTripleFromStrings("a", "edge", "b"),
		graph.NewTripleFromStrings("a", "edge", "c"),
		graph.NewTripleFromStrings("b", "edge", "d"),
		graph.NewTripleFromStrings("c", "edge", "d"),
	}
	db.Put(context.Background(), triples...)

	// Find paths from a to d through different intermediaries
	results, err := db.Search(context.Background(), []*graph.Pattern{
		{Subject: graph.ExactString("a"), Predicate: graph.ExactString("edge"), Object: graph.Binding("mid")},
		{Subject: graph.Binding("mid"), Predicate: graph.ExactString("edge"), Object: graph.ExactString("d")},
	}, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should find two paths: a->b->d and a->c->d
	if len(results) != 2 {
		t.Errorf("expected 2 paths through diamond, got %d", len(results))
	}

	mids := make(map[string]bool)
	for _, sol := range results {
		mids[string(sol["mid"])] = true
	}
	if !mids["b"] || !mids["c"] {
		t.Error("should find both b and c as intermediaries")
	}
}

// TestComplexQuery_StarPattern tests queries with star-shaped patterns
func TestComplexQuery_StarPattern(t *testing.T) {
	t.Parallel()
	db, cleanup := setupSocialGraph(t)
	defer cleanup()

	t.Run("get all attributes of a person", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.ExactString("alice"), Predicate: graph.Binding("prop"), Object: graph.Binding("value")},
		}, nil)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// alice has: type, knows (x2), likes (x2), age, livesIn
		if len(results) < 7 {
			t.Errorf("expected at least 7 properties for alice, got %d", len(results))
		}

		props := make(map[string][]string)
		for _, sol := range results {
			prop := string(sol["prop"])
			value := string(sol["value"])
			props[prop] = append(props[prop], value)
		}

		if len(props["knows"]) != 2 {
			t.Errorf("alice should know 2 people, got %d", len(props["knows"]))
		}
		if len(props["likes"]) != 2 {
			t.Errorf("alice should like 2 things, got %d", len(props["likes"]))
		}
	})
}

// TestComplexQuery_LargeJoin tests performance with larger join cardinality
func TestComplexQuery_LargeJoin(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a larger graph
	var triples []*graph.Triple
	for i := 0; i < 100; i++ {
		triples = append(triples,
			graph.NewTripleFromStrings("node", "connected", string(rune('a'+i%26))+string(rune('0'+i/26))),
		)
	}
	for i := 0; i < 26; i++ {
		for j := 0; j < 4; j++ {
			triples = append(triples,
				graph.NewTripleFromStrings(string(rune('a'+i))+string(rune('0'+j)), "has", "property"),
			)
		}
	}
	db.Put(context.Background(), triples...)

	// Query that produces a large intermediate result
	results, err := db.Search(context.Background(), []*graph.Pattern{
		{Subject: graph.ExactString("node"), Predicate: graph.ExactString("connected"), Object: graph.Binding("x")},
		{Subject: graph.Binding("x"), Predicate: graph.ExactString("has"), Object: graph.ExactString("property")},
	}, &SearchOptions{Limit: 50})

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 50 {
		t.Errorf("expected 50 results with limit, got %d", len(results))
	}
}

// TestComplexQuery_FilterChaining tests multiple filter conditions
func TestComplexQuery_FilterChaining(t *testing.T) {
	t.Parallel()
	db, cleanup := setupSocialGraph(t)
	defer cleanup()

	t.Run("multiple filter conditions", func(t *testing.T) {
		results, err := db.Search(context.Background(), []*graph.Pattern{
			{Subject: graph.Binding("person"), Predicate: graph.ExactString("type"), Object: graph.ExactString("Person")},
			{Subject: graph.Binding("person"), Predicate: graph.ExactString("livesIn"), Object: graph.Binding("city")},
			{Subject: graph.Binding("person"), Predicate: graph.ExactString("likes"), Object: graph.Binding("interest")},
		}, &SearchOptions{
			Filter: func(s graph.Solution) bool {
				city := string(s["city"])
				interest := string(s["interest"])
				// Only NYC people who like hiking
				return city == "NYC" && interest == "hiking"
			},
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// alice and bob live in NYC, both like hiking
		people := make(map[string]bool)
		for _, sol := range results {
			people[string(sol["person"])] = true
		}

		if !people["alice"] || !people["bob"] {
			t.Errorf("expected alice and bob, got %v", people)
		}
		if len(people) != 2 {
			t.Errorf("expected 2 people, got %d", len(people))
		}
	})
}

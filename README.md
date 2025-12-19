# LevelGraph

![Logo](https://raw.githubusercontent.com/levelgraph/levelgraph/master/logo.png)

**LevelGraph** is a Graph Database built on the ultra-fast key-value store
[LevelDB](https://github.com/google/leveldb). This is a Go port of the original
[JavaScript LevelGraph](https://github.com/levelgraph/levelgraph).

LevelGraph uses the **Hexastore** approach as presented in the article:
[Hexastore: sextuple indexing for semantic web data management](https://sci-hub.se/10.14778/1453856.1453965)
(C. Weiss, P. Karras, A. Bernstein - Proceedings of the VLDB Endowment, 2008).
Following this approach, LevelGraph uses six indices for every triple, enabling
extremely fast pattern matching queries.

## Installation

```bash
go get github.com/levelgraph/levelgraph
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/levelgraph/levelgraph"
)

func main() {
    // Open a database
    db, err := levelgraph.Open("./mydb")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Insert a triple
    triple := levelgraph.NewTripleFromStrings("alice", "knows", "bob")
    if err := db.Put(triple); err != nil {
        log.Fatal(err)
    }

    // Query by subject
    results, err := db.Get(&levelgraph.Pattern{
        Subject: []byte("alice"),
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, t := range results {
        fmt.Printf("%s %s %s\n", t.Subject, t.Predicate, t.Object)
    }
}
```

## Features

- **Hexastore Indexing**: Six indexes for every triple enable fast lookups by any combination of subject, predicate, and object
- **Pattern Matching**: Query triples using flexible patterns with variables
- **Search/Join**: Multi-pattern joins for complex graph queries
- **Navigator API**: Fluent API for graph traversal
- **Journalling**: Record all write operations for audit trails and replication
- **Facets**: Attach properties to subjects, predicates, objects, or entire triples
- **Binary Data Support**: Store arbitrary `[]byte` data in triples

## API Reference

### Opening a Database

```go
// Basic open
db, err := levelgraph.Open("/path/to/db")

// With options
db, err := levelgraph.Open("/path/to/db",
    levelgraph.WithJournal(),   // Enable journalling
    levelgraph.WithFacets(),    // Enable facets
)
```

### Triples

Triples are the fundamental unit of data:

```go
// Create from byte slices
triple := levelgraph.NewTriple([]byte("subject"), []byte("predicate"), []byte("object"))

// Create from strings (convenience)
triple := levelgraph.NewTripleFromStrings("alice", "knows", "bob")
```

### Put and Delete

```go
// Insert single triple
err := db.Put(triple)

// Insert multiple triples
err := db.Put(t1, t2, t3)

// Delete triple
err := db.Del(triple)
```

### Get (Query)

Query triples using patterns:

```go
// Get by subject
results, err := db.Get(&levelgraph.Pattern{
    Subject: []byte("alice"),
})

// Get by predicate and object
results, err := db.Get(&levelgraph.Pattern{
    Predicate: []byte("knows"),
    Object:    []byte("bob"),
})

// With limit and offset
results, err := db.Get(&levelgraph.Pattern{
    Subject: []byte("alice"),
    Limit:   10,
    Offset:  5,
})

// With filter
results, err := db.Get(&levelgraph.Pattern{
    Subject: []byte("alice"),
    Filter: func(t *levelgraph.Triple) bool {
        return string(t.Object) != "eve"
    },
})

// Reverse order
results, err := db.Get(&levelgraph.Pattern{
    Subject: []byte("alice"),
    Reverse: true,
})
```

### Search (Join)

Perform multi-pattern joins using variables:

```go
// Find friends of friends
results, err := db.Search([]*levelgraph.Pattern{
    {
        Subject:   []byte("alice"),
        Predicate: []byte("knows"),
        Object:    levelgraph.V("x"),  // Variable
    },
    {
        Subject:   levelgraph.V("x"),
        Predicate: []byte("knows"),
        Object:    levelgraph.V("y"),
    },
}, nil)

// Each result is a Solution map[string][]byte
for _, sol := range results {
    fmt.Printf("x=%s, y=%s\n", sol["x"], sol["y"])
}

// With options
results, err := db.Search(patterns, &levelgraph.SearchOptions{
    Limit:  10,
    Offset: 0,
    Filter: func(s levelgraph.Solution) bool {
        return string(s["x"]) != "eve"
    },
})
```

### Navigator API

Fluent API for graph traversal:

```go
// Find friends of alice
values, err := db.Nav("alice").
    ArchOut("knows").
    Values()

// Find who knows alice
values, err := db.Nav("alice").
    ArchIn("knows").
    Values()

// Chain multiple traversals
values, err := db.Nav("alice").
    ArchOut("knows").       // alice -> knows -> ?
    ArchOut("knows").       // ? -> knows -> ?
    Values()

// Name intermediate vertices
solutions, err := db.Nav("alice").
    ArchOut("knows").
    As("friend").
    ArchOut("likes").
    As("liked").
    Solutions()

// Bind to specific value
values, err := db.Nav("alice").
    ArchOut("knows").
    Bind("bob").            // Must match "bob"
    ArchOut("knows").
    Values()

// Check existence
exists, err := db.Nav("alice").ArchOut("knows").Exists()

// Count results
count, err := db.Nav("alice").ArchOut("knows").Count()

// Clone navigator for branching
nav := db.Nav("alice").ArchOut("knows")
nav1 := nav.Clone().ArchOut("likes")
nav2 := nav.Clone().ArchOut("follows")
```

### Iterators

For large result sets, use iterators:

```go
// Triple iterator
iter, err := db.GetIterator(&levelgraph.Pattern{Subject: []byte("alice")})
if err != nil {
    log.Fatal(err)
}
defer iter.Release()

for iter.Next() {
    triple, err := iter.Triple()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(triple)
}

// Search iterator
iter, err := db.SearchIterator(patterns, nil)
if err != nil {
    log.Fatal(err)
}
defer iter.Close()

for iter.Next() {
    solution := iter.Solution()
    fmt.Println(solution)
}
```

### Journalling

When enabled, all write operations are recorded:

```go
// Open with journalling
db, err := levelgraph.Open("/path/to/db", levelgraph.WithJournal())

// Get journal entries
entries, err := db.GetJournalEntries(time.Time{})  // All entries
entries, err := db.GetJournalEntries(since)        // Entries after timestamp

// Count entries
count, err := db.JournalCount(time.Time{})

// Trim old entries
trimmed, err := db.Trim(before)

// Export and trim
exported, err := db.TrimAndExport(before, archiveDB)

// Replay journal to another database
replayed, err := db.ReplayJournal(after, targetDB)
```

### Facets

Attach properties to graph components:

```go
// Open with facets
db, err := levelgraph.Open("/path/to/db", levelgraph.WithFacets())

// Component facets (on subjects, predicates, or objects)
err = db.SetFacet(levelgraph.FacetSubject, []byte("alice"), []byte("age"), []byte("30"))
value, err := db.GetFacet(levelgraph.FacetSubject, []byte("alice"), []byte("age"))
facets, err := db.GetFacets(levelgraph.FacetSubject, []byte("alice"))
err = db.DelFacet(levelgraph.FacetSubject, []byte("alice"), []byte("age"))

// Triple facets (on entire triples)
triple := levelgraph.NewTripleFromStrings("alice", "knows", "bob")
err = db.SetTripleFacet(triple, []byte("since"), []byte("2020"))
value, err := db.GetTripleFacet(triple, []byte("since"))
facets, err := db.GetTripleFacets(triple)
err = db.DelTripleFacet(triple, []byte("since"))
err = db.DelAllTripleFacets(triple)
```

## Binary Data Support

LevelGraph stores all data as `[]byte`, supporting arbitrary binary data:

```go
// Binary data in triples
subject := []byte{0x00, 0x01, 0x02}
predicate := []byte("hasData")
object := []byte{0xAA, 0xBB, 0xCC}

triple := levelgraph.NewTriple(subject, predicate, object)
db.Put(triple)
```

## Benchmarks

Run benchmarks:

```bash
go test -bench=. -benchmem
```

Example results:

```
BenchmarkPut-24              107331    10462 ns/op    4261 B/op    33 allocs/op
BenchmarkGet-24               49762    24005 ns/op    7467 B/op   171 allocs/op
BenchmarkSearch-24            79240    15024 ns/op    6696 B/op   122 allocs/op
BenchmarkSearchJoin-24        10000   103245 ns/op   58649 B/op   792 allocs/op
BenchmarkNavigator-24          9805   106973 ns/op   60318 B/op   831 allocs/op
```

## Testing

```bash
go test ./...
```

## Credits

This Go port builds on the excellent work of the original JavaScript LevelGraph
by Matteo Collina and contributors.

LevelGraph builds on LevelDB from Google, accessed via
[goleveldb](https://github.com/syndtr/goleveldb).

## License

MIT License - see [LICENSE](LICENSE) file.

Copyright (c) 2013-2025 Matteo Collina and LevelGraph Contributors
Copyright (c) 2025 Benjamin Babik and LevelGraph Go Contributors

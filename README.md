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
go get github.com/benbenbenbenbenben/levelgraph
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/benbenbenbenbenben/levelgraph"
)

func main() {
    ctx := context.Background()

    // Open a database
    db, err := levelgraph.Open("./mydb")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Insert a triple
    triple := levelgraph.NewTripleFromStrings("alice", "knows", "bob")
    if err := db.Put(ctx, triple); err != nil {
        log.Fatal(err)
    }

    // Query by subject
    results, err := db.Get(ctx, &levelgraph.Pattern{
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
- **Vector Search**: Semantic similarity search using vector embeddings (HNSW)
- **Hybrid Search**: Combine graph traversal with vector similarity

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
ctx := context.Background()

// Insert single triple
err := db.Put(ctx, triple)

// Insert multiple triples
err := db.Put(ctx, t1, t2, t3)

// Delete triple
err := db.Del(ctx, triple)
```

### Get (Query)

Query triples using patterns:

```go
ctx := context.Background()

// Get by subject
results, err := db.Get(ctx, &levelgraph.Pattern{
    Subject: []byte("alice"),
})

// Get by predicate and object
results, err := db.Get(ctx, &levelgraph.Pattern{
    Predicate: []byte("knows"),
    Object:    []byte("bob"),
})

// With limit and offset
results, err := db.Get(ctx, &levelgraph.Pattern{
    Subject: []byte("alice"),
    Limit:   10,
    Offset:  5,
})

// With filter
results, err := db.Get(ctx, &levelgraph.Pattern{
    Subject: []byte("alice"),
    Filter: func(t *levelgraph.Triple) bool {
        return string(t.Object) != "eve"
    },
})

// Reverse order
results, err := db.Get(ctx, &levelgraph.Pattern{
    Subject: []byte("alice"),
    Reverse: true,
})
```

### Search (Join)

Perform multi-pattern joins using variables:

```go
ctx := context.Background()

// Find friends of friends
results, err := db.Search(ctx, []*levelgraph.Pattern{
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
results, err := db.Search(ctx, patterns, &levelgraph.SearchOptions{
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
ctx := context.Background()

// Triple iterator
iter, err := db.GetIterator(ctx, &levelgraph.Pattern{Subject: []byte("alice")})
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
iter, err := db.SearchIterator(ctx, patterns, nil)
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

### Vector Search

LevelGraph supports semantic similarity search using vector embeddings. This enables "fuzzy" queries based on meaning rather than exact matches.

#### Basic Setup

```go
import (
    "github.com/benbenbenbenbenben/levelgraph"
    "github.com/benbenbenbenbenben/levelgraph/vector"
    "github.com/benbenbenbenbenben/levelgraph/vector/luxical"
)

// Load a text embedding model (Luxical produces 192-dim embeddings)
embedder, err := luxical.NewEmbedder("./models/luxical")
if err != nil {
    log.Fatal(err)
}
defer embedder.Close()

// Create a vector index matching the embedder dimensions
index := vector.NewHNSWIndex(embedder.Dimensions())

// Open database with vector support and auto-embedding
db, err := levelgraph.Open("/path/to/db",
    levelgraph.WithVectors(index),
    levelgraph.WithAutoEmbed(embedder, levelgraph.AutoEmbedObjects),
)
```

#### Manual Vector Operations

```go
ctx := context.Background()

// Set a vector manually
vec := []float32{0.1, 0.2, 0.3, ...} // 192 dimensions
id := vector.MakeID(vector.IDTypeObject, []byte("tennis"))
db.SetVector(ctx, id, vec)

// Get a vector
vec, err := db.GetVector(ctx, id)

// Search for similar vectors
results, err := db.SearchVectors(ctx, queryVec, 10)
for _, match := range results {
    fmt.Printf("ID: %s, Score: %.3f\n", match.ID, match.Score)
}

// Search by text (requires embedder)
results, err := db.SearchVectorsByText(ctx, "racket sports", 10)
```

#### Hybrid Search (Graph + Vectors)

Combine graph pattern matching with vector similarity:

```go
// Find people who like topics similar to "machine learning"
solutions, err := db.Search(ctx, []*levelgraph.Pattern{
    {Subject: levelgraph.V("person"), Predicate: []byte("likes"), Object: levelgraph.V("topic")},
}, &levelgraph.SearchOptions{
    VectorFilter: &levelgraph.VectorFilter{
        Variable:  "topic",
        QueryText: "machine learning",  // Will be embedded
        TopK:      10,                   // Top 10 similar topics
        MinScore:  0.7,                  // Filter low similarity
        IDType:    vector.IDTypeObject,
    },
})

for _, sol := range solutions {
    score := levelgraph.GetVectorScore(sol)
    fmt.Printf("%s likes %s (score: %.3f)\n", sol["person"], sol["topic"], score)
}
```

#### Async Auto-Embedding

For better performance with real embedding models, enable async embedding:

```go
db, err := levelgraph.Open("/path/to/db",
    levelgraph.WithVectors(index),
    levelgraph.WithAutoEmbed(embedder, levelgraph.AutoEmbedObjects),
    levelgraph.WithAsyncAutoEmbed(100),  // Buffer size
)

// Add triples (embedding happens in background)
for _, triple := range triples {
    db.Put(ctx, triple)
}

// Wait for all embeddings before searching
err = db.WaitForEmbeddings(ctx)
```

#### HNSW Parameter Tuning

```go
// High-speed, lower recall (~95%)
index := vector.NewHNSWIndex(192,
    vector.WithM(12),
    vector.WithEfConstruction(100),
    vector.WithEfSearch(30),
)

// Balanced (default, ~98% recall)
index := vector.NewHNSWIndex(192,
    vector.WithM(16),
    vector.WithEfConstruction(200),
    vector.WithEfSearch(50),
)

// High-recall (~99.5%)
index := vector.NewHNSWIndex(192,
    vector.WithM(32),
    vector.WithEfConstruction(400),
    vector.WithEfSearch(200),
)
```

#### Score Interpretation

- **1.0**: Identical vectors (perfect match)
- **0.7-0.9**: Highly similar (typically good matches)
- **0.5-0.7**: Moderately similar
- **0.0-0.5**: Dissimilar

## Web Playground (WASM)

LevelGraph can be compiled to WebAssembly and run directly in the browser. A playground is included for interactive experimentation.

### Building the Playground

```bash
# Build WASM module and start local server (standard Go, 3.8MB)
make serve

# Build with TinyGo for smaller binary (1.5MB, ~60% smaller)
make serve-tinygo
```

Then open http://localhost:8080 in your browser.

### Makefile Targets

```bash
make wasm             # Build WASM module (standard Go)
make wasm-tinygo      # Build WASM module (TinyGo, smaller)
make playground       # Build WASM + copy wasm_exec.js
make playground-tinygo # Build TinyGo WASM + copy wasm_exec_tinygo.js
make serve            # Build and start local server
make serve-tinygo     # Build TinyGo version and start server
```

The playground UI allows switching between builds via a dropdown menu.

### WASM API

When loaded in a browser, the following JavaScript API is available:

```javascript
// Insert triples
levelgraph.put([
    { subject: "alice", predicate: "knows", object: "bob" },
    { subject: "bob", predicate: "knows", object: "charlie" }
]);

// Delete triples
levelgraph.del([
    { subject: "alice", predicate: "knows", object: "bob" }
]);

// Query by pattern (use null for wildcards)
const results = levelgraph.get({ subject: "alice", predicate: null, object: null });

// Search with variables (prefix with ?)
const friends = levelgraph.search([
    { subject: "alice", predicate: "knows", object: "?friend" },
    { subject: "?friend", predicate: "knows", object: "?fof" }
]);

// Search with filters
const results = levelgraph.search([
    { subject: "?person", predicate: "knows", object: "?other" }
], {
    notEqual: [{ var: "person", var2: "other" }]  // person != other
});

// Navigation API
const nav = levelgraph.nav({
    start: "alice",
    steps: [
        { direction: "out", predicate: "knows", as: "friend" },
        { direction: "out", predicate: "likes", as: "liked" }
    ]
});

// Reset database
levelgraph.reset();

// Check if ready
if (levelgraph.isReady()) { /* ... */ }
```

The playground includes several example presets demonstrating these features.

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

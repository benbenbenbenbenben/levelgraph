# Simple LevelGraph Example

This example demonstrates the core features of LevelGraph:

- Basic triple operations (Put, Get, Del)
- Journaling (operation tracking)
- Facets (metadata on triples)
- Navigator API (graph traversal)
- Search with variables

## Running

```bash
cd example/simple
go run main.go
```

## Features Demonstrated

### 1. Basic Triple Operations

```go
db.Put(
    levelgraph.NewTripleFromStrings("alice", "knows", "bob"),
    levelgraph.NewTripleFromStrings("bob", "knows", "alice"),
)

results, _ := db.Get(levelgraph.NewPattern("alice", nil, nil))
```

### 2. Journaling

Track all write operations for audit trails:

```go
db, _ := levelgraph.Open(path, levelgraph.WithJournal())

// Get all journal entries (pass zero time for no filter)
entries, _ := db.GetJournalEntries(context.Background(), time.Time{})
for _, entry := range entries {
    fmt.Printf("%s: %s\n", entry.Operation, entry.Triple)
}

// Or filter entries before a specific time
before := time.Now()
entries, _ = db.GetJournalEntries(context.Background(), before)
```

### 3. Facets

Attach metadata to triples:

```go
db, _ := levelgraph.Open(path, levelgraph.WithFacets())

triple := levelgraph.NewTripleFromStrings("alice", "knows", "charlie")
db.Put(triple)
db.SetTripleFacet(triple, []byte("since"), []byte("2023"))
db.SetTripleFacet(triple, []byte("trust"), []byte("high"))

facets, _ := db.GetTripleFacets(triple)
```

### 4. Navigator API

Fluent graph traversal:

```go
solutions, _ := db.Nav([]byte("alice")).
    ArchOut([]byte("knows")).As("friend").
    ArchOut([]byte("knows")).As("friendOfFriend").
    Solutions()
```

### 5. Search with Variables

Pattern matching with variables:

```go
x := levelgraph.V("x")
y := levelgraph.V("y")
results, _ := db.Search([]*levelgraph.Pattern{
    levelgraph.NewPattern(x, "knows", y),
}, nil)
```

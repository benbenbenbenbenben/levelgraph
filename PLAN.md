# LevelGraph Go Port - Implementation Plan

## Overview

This document outlines the plan to port LevelGraph from JavaScript to Go, while adding new features (journalling and facets) and changing the core data type from strings to `[]byte`.

## Original JavaScript Library Summary

LevelGraph is a graph database built on top of LevelDB that uses the **Hexastore** indexing approach. It stores triples (subject, predicate, object) with six indexes for fast access:

- `spo` - Subject, Predicate, Object
- `sop` - Subject, Object, Predicate  
- `pos` - Predicate, Object, Subject
- `pso` - Predicate, Subject, Object
- `ops` - Object, Predicate, Subject
- `osp` - Object, Subject, Predicate

### Core Features to Port

1. **Triple Store Operations**
   - `Put` - Insert single or multiple triples
   - `Get` - Query triples by pattern matching (subject, predicate, object)
   - `Del` - Delete triples
   - Batch operations support

2. **Query Features**
   - Pattern matching with partial triple specification
   - Variables for binding (`db.v("name")`)
   - Limit and offset pagination
   - Reverse ordering
   - Filtering (sync and async callbacks)

3. **Search/Join Operations**
   - Search with multiple conditions (join queries)
   - Two join algorithms:
     - Basic join (JoinStream)
     - Sort-merge join (SortJoinStream)
   - Query planning with cost estimation
   - Materialized views (transform solutions to new triples)

4. **Navigator API**
   - Fluent API for graph traversal
   - `archOut(predicate)` - follow outgoing edges
   - `archIn(predicate)` - follow incoming edges
   - `as(name)` - name current position as variable
   - `bind(value)` - bind variable to value
   - `go(vertex)` - jump to another vertex
   - `values()` - get unique values
   - `solutions()` - get all variable bindings
   - `triples(pattern)` - materialize results

5. **Streaming Interface**
   - `GetStream` - stream results from get queries
   - `PutStream` - stream writes
   - `DelStream` - stream deletes
   - `SearchStream` - stream search results

## New Features

### 1. Journalling (Option-enabled)

A journal system that records all write operations for audit, replay, and export capabilities.

**Features:**
- Journal entries stored in the same database with a dedicated prefix
- Each entry contains: timestamp, operation type, triple data
- `Trim(before time.Time)` - remove journal entries before timestamp
- `TrimAndExport(before time.Time, targetDB)` - move old journal entries to another database file

**Schema:**
```
journal::<timestamp>::<operation_id> -> {op: "put"|"del", triple: {...}, ts: timestamp}
```

### 2. Facets/Properties (Option-enabled)

Allow attaching arbitrary key-value properties to any triple component (subject, predicate, object) or to the triple itself.

**Two approaches:**
1. **Component Facets** - Properties on individual S/P/O values
2. **Triple Facets** - Properties on the entire triple relationship

**Schema approach:**
```
facet::<component_type>::<component_value>::<property_key> -> property_value
triple_facet::<spo_key>::<property_key> -> property_value
```

### 3. Data Type Change: `[]byte`

All subject, predicate, and object values will be `[]byte` instead of strings. This provides:
- Better compatibility with binary data
- More portable across systems
- Explicit encoding/decoding at application level

## Go Architecture

### Package Structure

```
levelgraph/
├── levelgraph.go       # Main DB interface, Open(), options
├── triple.go           # Triple type and related methods
├── variable.go         # Variable type for query binding
├── index.go            # Index definitions and key generation
├── query.go            # Query building and execution
├── search.go           # Search/join implementation
├── join.go             # Basic join algorithm
├── sortjoin.go         # Sort-merge join algorithm
├── planner.go          # Query planner
├── filter.go           # Filter implementation
├── materializer.go     # Result materialization
├── navigator.go        # Navigator fluent API
├── iterator.go         # Iterator interfaces for streaming
├── journal.go          # Journalling feature
├── facets.go           # Facets/properties feature
├── options.go          # Option pattern configuration
├── encoding.go         # Key encoding/escaping utilities
└── levelgraph_test.go  # Tests
```

### Core Types

```go
// Triple represents a subject-predicate-object triple
type Triple struct {
    Subject   []byte
    Predicate []byte
    Object    []byte
}

// Variable represents a query variable for binding
type Variable struct {
    Name string
}

// DB is the main database interface
type DB struct {
    ldb     *leveldb.DB  // or appropriate Go LevelDB binding
    options *Options
}

// Options for database configuration
type Options struct {
    JournalEnabled bool
    FacetsEnabled  bool
    JoinAlgorithm  string // "basic" or "sort"
}

// Pattern for querying
type Pattern struct {
    Subject   interface{} // []byte or *Variable
    Predicate interface{} // []byte or *Variable
    Object    interface{} // []byte or *Variable
    Filter    func(*Triple) bool
    Limit     int
    Offset    int
    Reverse   bool
}

// Solution represents bound variables from a search
type Solution map[string][]byte
```

### Option Pattern

```go
type Option func(*Options)

func WithJournal() Option {
    return func(o *Options) {
        o.JournalEnabled = true
    }
}

func WithFacets() Option {
    return func(o *Options) {
        o.FacetsEnabled = true
    }
}

func WithJoinAlgorithm(algo string) Option {
    return func(o *Options) {
        o.JoinAlgorithm = algo
    }
}

func Open(path string, opts ...Option) (*DB, error)
```

## Implementation Phases

### Phase 1: Core Infrastructure
- [ ] Set up Go module and dependencies
- [ ] Implement Triple type with `[]byte` fields
- [ ] Implement Variable type
- [ ] Implement index key generation (6 indexes)
- [ ] Implement key escaping for special characters (`:`, `\`)
- [ ] Basic DB open/close with options

### Phase 2: Basic CRUD Operations
- [ ] Implement `Put` for single and batch triples
- [ ] Implement `Get` with pattern matching
- [ ] Implement `Del` for single and batch triples
- [ ] Implement query creation with index selection
- [ ] Support limit, offset, reverse
- [ ] Support basic filtering

### Phase 3: Search/Join Operations
- [ ] Implement basic join algorithm (JoinStream equivalent)
- [ ] Implement query planner with size estimation
- [ ] Implement sort-merge join algorithm (SortJoinStream equivalent)
- [ ] Implement solution filtering
- [ ] Implement materialization

### Phase 4: Navigator API
- [ ] Implement Navigator struct
- [ ] Implement `archOut`, `archIn` methods
- [ ] Implement `as`, `bind`, `go` methods
- [ ] Implement `Values`, `Solutions`, `Triples` methods

### Phase 5: Iterator/Streaming Interface
- [ ] Define Iterator interface
- [ ] Implement GetIterator
- [ ] Implement SearchIterator
- [ ] Implement PutIterator, DelIterator (or channel-based)

### Phase 6: Journalling Feature
- [ ] Implement journal entry type
- [ ] Hook into Put/Del to record journal entries
- [ ] Implement `Trim(before time.Time)`
- [ ] Implement `TrimAndExport(before time.Time, targetDB)`

### Phase 7: Facets Feature
- [ ] Implement facet storage schema
- [ ] Implement `SetFacet(component, key, value)`
- [ ] Implement `GetFacet(component, key)`
- [ ] Implement `GetFacets(component)`
- [ ] Implement triple-level facets

### Phase 8: Testing & Polish
- [ ] Port all JavaScript tests to Go
- [ ] Add benchmarks
- [ ] Documentation
- [ ] Example code

### Phase 9: Cleanup
- [ ] Remove JavaScript files
- [ ] Remove Node.js artifacts (package.json, etc.)
- [ ] Update README for Go usage

## Key Differences from JavaScript

1. **No streams** - Go uses iterators/channels instead of Node streams
2. **No callbacks** - Go uses return values and errors
3. **Explicit types** - Using `[]byte` and interfaces vs dynamic JS types
4. **Concurrency** - Can leverage goroutines for parallel operations
5. **Error handling** - Explicit error returns vs exceptions

## Dependencies

- `github.com/syndtr/goleveldb/leveldb` - LevelDB binding for Go
  (or `github.com/cockroachdb/pebble` as modern alternative)

## Testing Strategy

1. Unit tests for each component
2. Integration tests matching JS test cases
3. Benchmark tests for performance comparison
4. Fuzz tests for encoding/decoding

## Success Criteria

- [ ] All original LevelGraph features working
- [ ] Journalling feature complete with trim/export
- [ ] Facets feature complete
- [ ] All data stored as `[]byte`
- [ ] Comprehensive test coverage
- [ ] Clean removal of all JS artifacts
- [ ] Documentation updated for Go usage

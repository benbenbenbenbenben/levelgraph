# Architectural Review: LevelGraph

## Executive Summary

LevelGraph is a graph database built on top of LevelDB, implementing a hexastore approach for fast triple pattern matching. It appears to include recent additions for vector similarity search ("hybrid search") and journaling. The codebase is relatively compact but exhibits a mix of Go idioms and patterns that suggest a port from a dynamic language (likely the original Node.js LevelGraph) or rapid iteration.

## 1. File and Folder Structure

- **Flat Root**: The root directory is quite crowded. Core components like `search.go`, `journal.go`, `facets.go`, `navigator.go` reside alongside `levelgraph.go`.
- **Commonality**: It is common in Go libraries to have a flat structure, but as the project grows (adding vectors, facets), grouping related functionality into subpackages (e.g., `pkg/search`, `pkg/journal`) would improve navigability.
- **`cmd` Directory**: The `cmd` directory currently only contains `wasm`. There is no standard CLI tool entry point exposed, which is unusual for a database project.
- **`vector` and `memstore`**: These are correctly separated into subpackages, which is a good pattern.

## 2. Code Patterns & Design

- **Base Types**: The core `Triple` type uses `[]byte` for Subject, Predicate, and Object. This is efficient for LevelDB storage but requires frequent conversion/casting when working with strings or other types.
- **Composition**: The `DB` struct composes `KVStore`, which is good for testing (allowing `MemStore` or `LevelDB`).
- **Dependencies**:
  - The project uses `github.com/syndtr/goleveldb` effectively.
  - **Crucial Issue**: The `go.mod` file contains a local `replace` directive (`replace github.com/benbenbenbenbenben/luxical-one-go => /home/ben/luxical-one/go/luxical`). This makes the repository non-buildable for anyone other than the original author. This **must** be resolved.
- **Logging**: There is no unified logging strategy visible. Errors are returned, which is good, but debug/info logging seems absent or ad-hoc (e.g., relying on `fmt` or `log` in main applications).

## 3. Type Safety & Go Idioms

- **`interface{}` Usage**: The `Pattern` struct and `Variable` handling rely heavily on `interface{}`.
  - `Pattern` fields (`Subject`, `Predicate`, `Object`) are explicit `interface{}`.
  - While this allows flexibility (variables vs constants), it bypasses Go's type safety.
  - The existence of `pattern_typed.go` suggests an awareness of this, but the core `search.go` still relies on the loose typing.
- **Magic Comments**: No significant overuse of magic comments found, though `//go:build !js` in `storage.go` is appropriate for WASM support.

## 4. Components Analysis

### Search & Vectors

- **Hybrid Search**: The `VectorFilter` implementation in `search.go` applies vector similarity scoring _after_ retrieving graph solutions.
- **Performance Risk**: `applyVectorFilter` iterates over _all_ intermediate solutions to compute vector scores. If the graph query returns thousands of results, this will be a severe bottleneck, especially since it calculates cosine similarity (dot product + norms) for every result.
- **Memory**: High constraint on memory if result sets are large, as `sort.Slice` is used on the full result set.

### Journaling

- **JSON Encoding**: `journal.go` uses `encoding/json` to store journal entries.
  - JSON is verbose and slow to marshal/unmarshal compared to binary formats (like Gob or Protobuf).
  - Since `Triple` fields are `[]byte`, `encoding/json` will base64 encode them, adding 33% overhead to storage size.

### Facets

- **Implementation**: Facets are stored as separate keys. This is a reasonable approach for a KV store implementation of a property graph, keeping the core hexastore indices clean.

## 5. Testing

- Tests exist (`levelgraph_test.go`, `vector_test.go`), which is good.
- `complex_queries_test.go` suggests an attempt to test realistic scenarios.

## Recommendations

1.  **Remove Local Replace**: Immediately remove the `replace` directive in `go.mod` or point it to a published tag. (editor: Defer this, depends on another repo doing work!)
2.  **Refactor Root**: Move core components into a `pkg/` structure or group them logically (e.g., `pkg/core`, `pkg/search`).
3.  **Strict Typing**: Migrate `Pattern` to use a algebraic data type approach (e.g., an interface implemented by `Variable` and `Constant` types) rather than `interface{}` to gain compile-time safety. (editor: Agreed, do this and test test test!)
4.  **Optimize Journaling**: Switch from JSON to a binary format (e.g., `encoding/gob` or a custom binary packer) to save space and CPU. (editor: Again agreed, a good idea. Test it well, we don't want to make assumptions about portability and find out we break something.)
5.  **Vector Optimization**: Consider an index look-up strategy that doesn't require scoring every single graph match if `TopK` is small, or at least parallelize the scoring. (editor: Explore the options here, we'll need benchmarks etc in all cases to make a decision.)
6.  **CLI**: Add a root-level CLI in `cmd/levelgraph` for basic database operations (put, get, query) to assist with debugging and administration. (editor: Fair, that would be practical, let's do that.)

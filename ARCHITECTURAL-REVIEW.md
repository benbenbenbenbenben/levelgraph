# Architectural Review: LevelGraph

## Executive Summary

LevelGraph is a graph database built on top of LevelDB, implementing a hexastore approach for fast triple pattern matching. It appears to include recent additions for vector similarity search ("hybrid search") and journaling. The codebase is relatively compact but exhibits a mix of Go idioms and patterns that suggest a port from a dynamic language (likely the original Node.js LevelGraph) or rapid iteration.

## 1. File and Folder Structure

- **Refactored Root**: Core graph types (`Triple`, `Pattern`, `Variable`, etc.) have been moved to `pkg/graph`, significantly decluttering the root directory.
- **`cmd` Directory**: Now contains `cmd/levelgraph` with a functional CLI tool (`put`, `get`, `dump`, `load`), in addition to `wasm`.
- **`vector` and `memstore`**: These remain correctly separated.

## 2. Code Patterns & Design

- **Storage Format**: The storage format has been refactored to use compact binary encoding (varints) for Triples and Journal entries, replacing the previous JSON+Base64 approach. This improves storage efficiency and performance.
- **Base Types**: `Triple` still uses `[]byte`, but the move to `pkg/graph` provides a cleaner API surface. `pkg/index` exports needed helpers.
- **Composition**: The `DB` struct composes `KVStore`.
- **Dependencies**:
  - The project uses `github.com/syndtr/goleveldb` effectively.
  - **Crucial Issue**: The `go.mod` file contains a local `replace` directive (`replace github.com/benbenbenbenbenben/luxical-one-go => /home/ben/luxical-one/go/luxical`). This makes the repository non-buildable for anyone other than the original author. This **must** be resolved.
- **Logging**: There is no unified logging strategy visible.

## 3. Type Safety & Go Idioms

- **`interface{}` Usage**: The `Pattern` struct and `Variable` handling rely heavily on `interface{}`.
- **Magic Comments**: No significant overuse.

## 4. Components Analysis

### Search & Vectors

- **Hybrid Search**: `VectorFilter` applies similarity scoring.
- **Performance Risk**: `applyVectorFilter` still iterates over intermediate solutions. This remains a potential bottleneck for large result sets.

### Facets

- **Implementation**: Facets are stored as separate keys.

## 5. Testing

- Tests exist and have been verified to pass after the refactoring (`levelgraph_test.go`, `vectors_test.go`, `pattern_typed_test.go`).

## Recommendations

1.  **Remove Local Replace**: Immediately remove the `replace` directive in `go.mod` or point it to a published tag. (editor: Defer this, depends on another repo doing work!)
2.  **Refactor Root**: [DONE] Core types moved to `pkg/graph`.
3.  **Strict Typing**: Migrate `Pattern` to use a algebraic data type approach. (editor: `TypedPattern` exists, further adoption encouraged.)
4.  **Vector Optimization**: Consider an index look-up strategy. (editor: Explore the options here.)
5.  **CLI**: [DONE] Added `cmd/levelgraph` for basic operations.
6.  **Binary Encoding**: [DONE] Implemented efficient binary storage format.

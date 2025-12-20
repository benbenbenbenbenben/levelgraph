---
created: 2025-12-20T11:50:24.801Z
---

# 10 Add HNSW graph structure persistence

## Problem

Currently only raw vectors are persisted, not the HNSW graph structure (levels, connections). On reload via `LoadVectors()`, the entire HNSW index must be rebuilt from scratch by re-inserting each vector.

## Impact
**Medium** - Slow startup for large indexes. Rebuilding is O(n log n) which can take minutes for millions of vectors.

## Current Behavior
```go
// On startup:
db.LoadVectors(ctx)  // Re-inserts each vector, rebuilding HNSW from scratch
```

## Solution
1. Serialize HNSW metadata: entry point, max level, node levels
2. Serialize connections for each node at each level
3. Store in KVStore with `hnsw::` prefix
4. Add `SaveHNSW()` / `LoadHNSW()` methods
5. Consider: Auto-save on Close(), auto-load on Open()

## Acceptance Criteria
- [ ] HNSW graph structure can be saved and loaded
- [ ] Loaded index has same search quality as before save
- [ ] Startup time significantly reduced for large indexes
- [ ] Backward compatible with existing databases (fallback to rebuild)
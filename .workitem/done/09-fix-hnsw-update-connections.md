---
created: 2025-12-20T11:50:19.179Z
---

# 09 Fix HNSW update to rebuild connections when vector changes

## Problem

When updating an existing vector in HNSW, only the vector data is replaced - the graph connections are not rebuilt.

**Location**: `hnsw.go:159-164`

```go
if existing, exists := h.nodes[idStr]; exists {
    existing.vector = v
    // TODO: Could rebuild connections, but for simplicity we just update the vector
    return nil
}
```

## Impact
**Low-Medium** - If vector changes significantly, connections become suboptimal, reducing recall.

## Solution Options
1. Simple: Delete and re-add the node
2. Better: Rebuild connections for the node using the new vector
3. Lazy: Only rebuild if vector changed significantly (cosine distance > threshold)

## Acceptance Criteria
- [ ] Updated vectors have appropriate connections
- [ ] Search recall maintained after updates
- [ ] Add test that updates vectors and verifies search quality
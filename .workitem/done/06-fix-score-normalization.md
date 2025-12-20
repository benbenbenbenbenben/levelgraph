---
created: 2025-12-20T11:50:09.127Z
---

# 06 Fix score normalization inconsistency

## Problem

Score ranges are inconsistent between different code paths:

In `search.go:525-527` (VectorFilter):
```go
// Normalize to [0, 1] range (cosine similarity is [-1, 1])
normalizedScore := (similarity + 1) / 2
```

But in Index implementations (`flat.go:159`, `hnsw.go:325`):
```go
Score: 1 - dist,  // For cosine distance, this is cosine similarity in [-1,1]
```

## Impact
**Medium** - Confusing API where scores mean different things in different contexts. VectorFilter MinScore threshold may not work as expected.

## Solution
Standardize on one score range throughout:
- Option A: Always use cosine similarity [-1, 1]
- Option B: Always normalize to [0, 1]

Document the chosen convention clearly.

## Acceptance Criteria
- [ ] Consistent score range across all vector search methods
- [ ] Document score range in godoc
- [ ] Update MinScore documentation to reflect actual range
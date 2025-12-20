---
created: 2025-12-20T11:50:01.234Z
---

# 04 Add dimension validation for Embedder and VectorIndex

## Problem

No validation that Embedder dimensions match VectorIndex dimensions when both are configured:

```go
db, _ := Open(path,
    WithVectors(vector.NewHNSWIndex(192)),
    WithAutoEmbed(embedder, AutoEmbedObjects), // What if embedder produces 384-dim?
)
```

This will cause runtime errors on first auto-embed attempt.

## Impact
**Medium** - Runtime errors instead of clear configuration error at startup.

## Solution
Add validation in `Open()` or `applyOptions()` that checks:
1. If both VectorIndex and Embedder are set
2. Compare `VectorIndex.Dimensions()` with `Embedder.Dimensions()`
3. Return error if they don't match

## Acceptance Criteria
- [ ] Open() returns error when dimensions don't match
- [ ] Clear error message indicating the mismatch
- [ ] Add test for dimension mismatch detection
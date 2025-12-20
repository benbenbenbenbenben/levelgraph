---
created: 2025-12-20T11:49:56.495Z
---

# 03 Fix misleading lock code in SearchVectorsByText

## Problem

The lock handling in `SearchVectorsByText` has misleading/dead code:

**Location**: `vectors.go:260-264`

```go
// Release lock and search (search will re-acquire)
db.mu.RUnlock()
defer db.mu.RLock()  // This defer executes AFTER function returns - meaningless

return db.SearchVectors(ctx, queryVec, k)
```

The `defer db.mu.RLock()` does nothing useful - it would re-acquire the lock after the function returns, which is pointless.

## Impact
**Medium** - No correctness issue, but confusing and misleading code.

## Solution
Remove the misleading defer statement. The function should simply unlock before calling SearchVectors (which will acquire its own lock).

## Acceptance Criteria
- [ ] Remove the misleading defer statement
- [ ] Add comment explaining the lock release pattern
- [ ] Verify SearchVectorsByText still works correctly
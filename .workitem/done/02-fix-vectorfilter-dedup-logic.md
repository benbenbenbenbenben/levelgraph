---
created: 2025-12-20T11:49:52.922Z
---

# 02 Fix VectorFilter deduplication logic bug

## Problem

The deduplication logic in `applyVectorFilter` tries to reuse a cached score for duplicate variable values, but the inner loop breaks immediately without finding the correct score.

**Location**: `search.go:498-510`

```go
for _, s := range scored {
    if seen[string(vector.MakeID(idType, s.solution[vf.Variable]))] {
        scored = append(scored, scoredSolution{...})
        break  // This breaks immediately without finding the matching score
    }
}
```

## Impact
**Critical** - Duplicate variable values get score 0 instead of the correct cached score, causing incorrect ranking in hybrid search results.

## Solution
Build a proper score cache map: `map[string]float32` keyed by vector ID string, populated on first encounter, looked up on duplicates.

## Acceptance Criteria
- [ ] Duplicate variable values get the same score as the original
- [ ] Add test case with multiple solutions having the same variable value
- [ ] Verify ranking is correct with duplicates
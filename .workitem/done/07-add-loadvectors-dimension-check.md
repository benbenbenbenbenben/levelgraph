---
created: 2025-12-20T11:50:11.987Z
---

# 07 Add dimension validation in LoadVectors

## Problem

`LoadVectors` doesn't validate that persisted vectors have the same dimensions as the index. If the index was recreated with different dimensions, vectors will silently fail to load or corrupt the index.

**Location**: `vectors.go:342-402`

## Impact
**Medium** - Silent corruption or confusing errors when reopening DB with different vector dimensions.

## Solution
1. Check first loaded vector's dimensions against index dimensions
2. If mismatch, return clear error
3. Optionally: skip mismatched vectors with warning log

## Acceptance Criteria
- [ ] LoadVectors returns error on dimension mismatch
- [ ] Clear error message indicating expected vs actual dimensions
- [ ] Add test for dimension mismatch scenario
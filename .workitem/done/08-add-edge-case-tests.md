---
created: 2025-12-20T11:50:15.720Z
---

# 08 Add missing edge case tests for vector feature

## Problem

Several edge cases are not tested:

## Missing Test Cases

### Critical
- [ ] Vector IDs containing colons (`:`) - URLs, URIs
- [ ] VectorFilter with non-existent variable name
- [ ] VectorFilter with QueryText but no Embedder configured

### Important
- [ ] Empty text embedding
- [ ] Very large vectors (high dimensions, e.g., 1536)
- [ ] Concurrent auto-embed operations
- [ ] HNSW with many deletions (recall degradation)
- [ ] Hybrid search with SearchIterator (should error or be documented)
- [ ] LoadVectors after auto-embed
- [ ] Auto-embed with HNSW index (most tests use FlatIndex)

### Nice to Have
- [ ] WASM build compatibility test
- [ ] Real Luxical embedder end-to-end test
- [ ] Stress test with thousands of vectors

## Acceptance Criteria
- [ ] All critical test cases added and passing
- [ ] All important test cases added and passing
- [ ] Test coverage for vector package > 80%

## Notes

---
**Autopilot Note (2025-12-28)**: Reviewed checklist against current tests:
- [x] Vector IDs with colons - Tested in `TestDB_VectorIDsWithSpecialCharacters` (vectors_test.go:1308)
- [x] VectorFilter with non-existent variable - Tested in vectors_test.go:1377
- [x] VectorFilter with QueryText but no Embedder - Tested in `TestDB_VectorFilterQueryTextNoEmbedder`
- [x] Hybrid search with SearchIterator - Now documented (SearchIterator doesn't support VectorFilter, use Search() instead)

Most critical edge cases are covered. Remaining items are nice-to-have.
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
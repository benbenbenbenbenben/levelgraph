---
created: 2025-12-20T11:50:35.888Z
---

# 12 Add comprehensive documentation for vector feature

## Problem

Vector feature documentation is incomplete:

## Missing Documentation

### Godoc
- [ ] Examples for hybrid search usage
- [ ] Explanation of score ranges and what they mean
- [ ] HNSW parameter tuning guidance (M, efConstruction, efSearch)
- [ ] When to use FlatIndex vs HNSWIndex

### README
- [ ] Vector search feature overview
- [ ] Quick start example with Luxical
- [ ] Performance characteristics

### Luxical Package
- [ ] Model file requirements (which files needed)
- [ ] Model download/setup instructions
- [ ] Int8 vs Float16 trade-offs

## Acceptance Criteria
- [ ] All public APIs have godoc with examples
- [ ] README includes vector search section
- [ ] Luxical package has setup instructions
- [ ] Parameter tuning guide available
---
created: 2025-12-20T17:24:58.002Z
---

# Optimize Navigator/Search Performance

The Navigator fluent API (`nav.ArchOut().ArchOut()`) does significant looping and filtering in search.go.

Tasks:
1. Add benchmarks for Navigator operations (before)
2. Profile to identify bottlenecks
3. Consider optimizations:
   - Lazy evaluation / iterator-based approach
   - Caching intermediate results
   - Batch lookups instead of per-solution iteration
4. Run benchmarks (after) to measure improvement
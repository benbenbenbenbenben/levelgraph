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

## Notes

### Completed 2025-12-20

#### Profiling Results
Identified top allocation sources:
1. `Variable.Bind` - 786MB (33%) - Creates new Solution map on every bind
2. `Solution.Clone` - 353MB (15%) - Deep copies byte slices unnecessarily
3. `TripleIterator.parseCurrentValue` - 170MB (7%) - Parsing from LevelDB

#### Optimizations Implemented
1. **`Variable.Bind`** - Returns same solution when value already bound (avoids allocation)
2. **`Solution.ShallowClone`** - New method for fast cloning when values won't be mutated
3. **`Variable.BindInPlace`** - New method for in-place binding without map allocation
4. **`Pattern.BindTripleFast`** - Uses shallow clone + in-place binding
5. **Pre-allocate slices** in Search loop

#### Results
```
Before: 44.5μs/op, 48KB, 560 allocs
After:  32.3μs/op, 27KB, 424 allocs
```
- ~27% faster
- ~43% less memory
- ~24% fewer allocations
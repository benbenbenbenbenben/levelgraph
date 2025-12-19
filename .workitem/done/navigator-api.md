---
created: 2025-12-19T18:29:14.420Z
---

# Phase 4: Navigator Fluent API

Implement the Navigator fluent API for graph traversal:
- Navigator struct with builder pattern
- archOut(predicate) - follow outgoing edges
- archIn(predicate) - follow incoming edges
- as(name) - name current position as variable
- bind(value) - bind variable to specific value
- go(vertex) - jump to another vertex
- Values() - get unique values at current position
- Solutions() - get all variable bindings
- Triples(pattern) - materialize results as triples
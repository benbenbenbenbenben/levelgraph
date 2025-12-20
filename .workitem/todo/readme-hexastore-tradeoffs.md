---
created: 2025-12-20T17:24:51.286Z
---

# Document Hexastore Trade-offs in README

The README mentions Hexastore but doesn't explain the design trade-off:
- 6x write amplification (one entry per index: SPO, SOP, POS, PSO, OPS, OSP)
- Benefit: O(1) lookups by any combination of subject, predicate, object
- This is a deliberate design choice optimizing for read speed over write speed/storage

Also explain why []byte is the core type:
- Supports arbitrary binary data (not just strings)
- Avoids encoding overhead
- Consistent with LevelDB's native key/value types
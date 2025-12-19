---
created: 2025-12-19T18:29:08.988Z
---

# Phase 1: Core Infrastructure Setup

Set up Go module and basic infrastructure:
- Initialize go.mod with module path
- Add LevelDB dependency (goleveldb or pebble)
- Implement Triple type with []byte fields for Subject, Predicate, Object
- Implement Variable type for query binding
- Implement index key generation for all 6 indexes (spo, sop, pos, pso, ops, osp)
- Implement key escaping for special characters (: and \)
- Implement Options struct and option pattern
- Basic DB Open/Close functions
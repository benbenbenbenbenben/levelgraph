---
created: 2025-12-24T01:00:09.070Z
---

# Improve Core Function Test Coverage

Several core functions have coverage below 85%:

- `journal.go:MarshalBinary` (81.8%) - error paths for binary.Write
- `journal.go:recordJournalEntry` (80.0%) - error paths
- `journal.go:ReplayJournal` (80.6%) - complex error scenarios
- `levelgraph.go:Put` (77.8%) - store.Write errors, journal errors
- `levelgraph.go:Del` (78.3%) - similar to Put
- `levelgraph.go:Open` (84.6%) - LevelDB open errors

These require mocking store internals to test error paths. Consider:
1. Creating a mockable store interface for testing
2. Adding error injection capabilities for testing
3. Focusing on the most impactful functions first

**Priority**: Low - current coverage (88.1%) is good, these are diminishing returns.
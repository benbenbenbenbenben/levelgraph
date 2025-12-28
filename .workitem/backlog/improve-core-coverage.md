# Improve Core Function Test Coverage

Current Status: 90.5% Main Package Coverage

Several functions have coverage below 85%:

Highest Impact (below 85%):
- journal.go:recordJournalEntry (80.0%) - MarshalBinary error path
- journal.go:MarshalBinary (81.8%) - binary.Write errors (hard to trigger)
- levelgraph.go:Put (81.5%) - store.Write errors, journal errors
- search.go:advance (81.8%) - iterator edge cases
- levelgraph.go:OpenWithDB (83.3%) - error paths
- journal.go:GetJournalEntries (84.6%)
- levelgraph.go:getUnlocked (84.6%)

Notes:
- Most uncovered lines are error paths requiring mocked store internals
- MarshalBinary errors are impossible to trigger (binary.Write to bytes.Buffer doesn't fail)
- Current coverage is excellent for production use

Priority: Low - diminishing returns beyond 90%

## Notes



---
**Autopilot Note (2025-12-24)**: Session improved coverage from 88.5% to 90.5%:
- Navigator.Clone: 80% → 100%
- applyVectorFilter: 65.8% → 96.1%
- ReplayJournal: 80.6% → 86.1%

Added tests for VectorFilter embedder error and optimization path. Remaining uncovered lines are error paths requiring complex mocking (binary.Write failures, iterator errors, etc.) - diminishing returns.

---
**Autopilot Note (2025-12-24)**: CLI coverage improved: 86.9% → 89.7%. Current coverage summary:
- levelgraph (main): 90.5%
- cmd/levelgraph: 89.7%
- vector: 97.8%
- pkg/graph: 95.3%
- pkg/index: 95.0%
- memstore: 94.7%
- vector/luxical: 89.5%

Remaining uncovered lines are mostly error paths requiring complex mocking (binary.Write failures, iterator errors, etc.) - diminishing returns.

---
**Autopilot Note (2025-12-28)**: Go version upgraded to 1.25.5 LTS. All tests pass, build succeeds. Staticcheck updated and passes. Coverage remains excellent:
- levelgraph (main): 90.5%
- cmd/levelgraph: 89.7%
- vector: 97.8%
- vector/luxical: 89.5%
- pkg/graph: 95.3%
- pkg/index: 95.0%
- memstore: 94.7%
---
created: 2025-12-19T18:29:17.817Z
---

# Phase 6: Journalling Feature

Implement optional journalling system:
- Journal entry type with timestamp, operation, triple data
- Journal storage schema with dedicated key prefix
- Hook into Put/Del to record journal entries when enabled
- Trim(before time.Time) - remove old journal entries
- TrimAndExport(before time.Time, targetDB) - move old entries to another DB
- Enable via WithJournal() option
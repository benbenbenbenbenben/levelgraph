---
created: 2025-12-20T11:50:05.175Z
---

# 05 Fix HNSW delete to repair neighbor connections

## Problem

When a node is deleted from HNSW, the algorithm removes connections TO the deleted node but doesn't add replacement connections between the orphaned neighbors.

**Location**: `hnsw.go:234-269`

If node B connects A--B--C, deleting B leaves A and C disconnected even if they should be connected.

## Impact
**Medium** - Graph quality degrades after deletions, recall may decrease over time with many delete operations.

## Solution Options
1. Simple: Connect each pair of orphaned neighbors (may exceed mMax)
2. Better: For each orphaned neighbor, search for new connections to maintain connectivity
3. Best: Implement proper HNSW delete with neighbor reconnection as per paper

## Acceptance Criteria
- [ ] Deleted node's neighbors maintain graph connectivity
- [ ] Recall doesn't degrade significantly after many deletions
- [ ] Add test that deletes nodes and verifies search quality
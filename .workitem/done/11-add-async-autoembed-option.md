---
created: 2025-12-20T11:50:32.301Z
---

# 11 Add async auto-embed option to not block Put

## Problem

Currently auto-embed runs synchronously during `Put()`, which blocks on embedding computation. For real embedders this can be slow (~7ms per embedding with Luxical, more for larger models).

## Impact
**Low** - Acceptable for small batches, but could slow down bulk imports significantly.

## Solution
Add option for async embedding:

```go
WithAutoEmbed(embedder, AutoEmbedObjects, AsyncEmbed(true))
```

Implementation:
1. Queue embedding requests to a background worker
2. Return from Put() immediately after graph write
3. Worker processes embedding queue asynchronously
4. Optional: WaitForEmbeddings() method to block until queue is drained

## Considerations
- What happens if embedding fails? Log warning? Retry queue?
- Memory pressure from large queues
- Shutdown handling - wait for queue to drain?

## Acceptance Criteria
- [ ] AsyncEmbed option available
- [ ] Put() returns immediately when async enabled
- [ ] Embeddings eventually complete in background
- [ ] Graceful handling of embedding failures
- [ ] WaitForEmbeddings() or similar sync mechanism
---
created: 2025-12-23T22:16:35.691Z
---

# Add unit tests for luxical package that don't require model files

The vector/luxical package has 78.9% coverage, but all tests skip when model files are unavailable.

## Suggested Tests

1. Test `CosineSimilarity` with known vectors:
   - Identical vectors should return 1.0
   - Orthogonal vectors should return 0.0
   - Opposite vectors should return -1.0

2. Test error handling:
   - `NewEmbedder` with non-existent directory
   - `NewEmbedderInt8` with non-existent directory

3. Test interface compliance (already exists)

## Notes
The CosineSimilarity function is a wrapper around lux.CosineSimilarity, so unit tests would verify the wrapper works correctly.
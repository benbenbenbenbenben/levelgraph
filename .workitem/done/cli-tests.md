---
created: 2025-12-23T22:16:32.000Z
---

# Add tests for CLI tool (cmd/levelgraph)

The cmd/levelgraph CLI tool currently has 0% test coverage.

## Suggested Approach

1. Refactor main.go to extract testable functions:
   - Move command logic from `runPut`, `runGet`, `runDump`, `runLoad` to accept io.Writer for output
   - Create a `CLI` struct that can be instantiated with mock dependencies
   - Make database path configurable via dependency injection

2. Add unit tests for:
   - `parseFlags` argument parsing
   - `runPut` triple insertion
   - `runGet` pattern matching with wildcards
   - `runDump` output formatting
   - `runLoad` N-Triples parsing

3. Add integration tests using temp directories

## Priority
Low - The CLI is a thin wrapper around the well-tested core library.
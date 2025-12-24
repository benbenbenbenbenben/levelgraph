# Remove Local Replace Directive from go.mod

The go.mod file contains:
```
replace github.com/benbenbenbenbenben/luxical-one-go => /home/ben/luxical-one/go/luxical
```

This makes the repo non-buildable for anyone except the original author.

**BLOCKED**: The remote `luxical-one-go` repo doesn't have the Go package at the root level. When we remove the replace directive, `go mod tidy` fails:
```
module github.com/benbenbenbenbenben/luxical-one-go@latest found (v0.0.0-20251220142615-20bf408f3499), but does not contain package github.com/benbenbenbenbenben/luxical-one-go
```

**Required action**: Push the Go package files (embedder.go, dense.go, tokenizer.go, etc.) to the root of the `luxical-one-go` GitHub repo, then:
1. Remove the replace directive
2. Run `go mod tidy`
3. Test builds

## Notes


---
**Autopilot Note (2025-12-23)**: This item is BLOCKED and requires manual intervention. The external `luxical-one-go` GitHub repo needs Go package files pushed to its root level. This cannot be done from within this repository.

---
**Autopilot Note (2025-12-24)**: Session completed with 49 commits. This workitem remains BLOCKED pending external action (push Go package files to root of luxical-one-go GitHub repo). All other improvements have been made:
- Test coverage: 88.1%
- CI updated to Go 1.23
- All lints/tests pass
- 8 new test functions for edge cases and error paths
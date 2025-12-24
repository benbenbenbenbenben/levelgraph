---
created: 2025-12-24T01:31:16.740Z
---

# Fix Go Version in go.mod

The go.mod file specifies `go 1.25.1` which is a development/future version.

The minimum supported version should be `go 1.21` to match:
1. CI matrix (tests 1.21, 1.22, 1.23)
2. Usage of `log/slog` which was added in Go 1.21

To fix:
```bash
go mod edit -go=1.21
go mod tidy
```

Priority: Low - builds work on current Go versions
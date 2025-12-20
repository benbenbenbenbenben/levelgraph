---
created: 2025-12-20T17:24:55.243Z
---

# Add Structured Logging with slog

Currently there's no unified logging strategy. Add structured logging using Go's built-in `log/slog` package.

Areas to add logging:
- Database open/close
- Auto-embed failures (currently just warns)
- Journal operations
- Vector index operations
- Error conditions

Consider:
- Default to no-op logger
- Allow users to provide custom slog.Handler via option
- Use appropriate log levels (Debug, Info, Warn, Error)
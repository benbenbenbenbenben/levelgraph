---
created: 2025-12-20T10:37:38.960Z
---

# Investigate TinyGo for smaller WASM binary

The current WASM binary (`playground/levelgraph.wasm`) is ~3.7MB when built with standard Go.

Investigate whether TinyGo can produce a smaller binary:

- Compare binary sizes between Go and TinyGo builds
- Test if all features work correctly with TinyGo (syscall/js, maps, slices, etc.)
- Measure any performance differences
- Document any code changes needed for TinyGo compatibility

TinyGo often produces significantly smaller WASM binaries (sometimes 10-100x smaller) but may have limitations with reflection, goroutines, or certain stdlib packages.
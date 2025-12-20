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

## Notes

### Results

#### Binary Size Comparison
- Standard Go: 3.8MB
- TinyGo: 1.5MB
- **Size reduction: 60%**

#### Implementation
- Added `make wasm-tinygo` and `make playground-tinygo` targets
- Playground now supports build selection via dropdown
- Defaults to TinyGo build for faster loading
- Both builds pass all tests
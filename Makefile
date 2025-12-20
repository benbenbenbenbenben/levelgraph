.PHONY: test bench bench-update lint fmt vet clean examples check wasm playground serve

# Run tests with race detector
test:
	go test -race ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Run benchmarks and update README with results
# Usage: make bench-update
bench-update:
	@echo "Running benchmarks and updating README..."
	@go test -bench=. -benchmem -run=^$$ . 2>/dev/null | grep -E '^Benchmark' > /tmp/bench_results.txt
	@echo "Benchmark results:"
	@cat /tmp/bench_results.txt
	@echo ""
	@echo "To update README.md, replace the benchmark section with the results above."
	@echo "Or run: make bench-readme"

# Automatically update README benchmark section (requires sed)
bench-readme:
	@echo "Updating README.md benchmarks..."
	@go test -bench=. -benchmem -run=^$$ . 2>/dev/null | grep -E '^Benchmark' > /tmp/bench_results.txt
	@echo "Latest benchmark results saved. Manual update of README.md recommended."
	@cat /tmp/bench_results.txt

# Run linting (go vet)
vet:
	go vet ./...

# Format code
fmt:
	gofmt -w .

# Lint alias
lint: vet

# Clean test cache
clean:
	go clean -testcache

# Run examples
examples:
	go test -v -run Example ./...

# Run all checks before commit
check: fmt vet test

# Build WebAssembly module (standard Go)
wasm:
	GOOS=js GOARCH=wasm go build -o playground/levelgraph.wasm ./playground/wasm/

# Build WebAssembly module with TinyGo (smaller binary)
wasm-tinygo:
	tinygo build -o playground/levelgraph-tinygo.wasm -target wasm ./playground/wasm/

# Build and update playground (including wasm_exec.js)
playground: wasm
	@cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" playground/
	@echo "Playground built. Run 'make serve' to start local server."

# Build and update playground with TinyGo (smaller binary, ~60% size reduction)
playground-tinygo: wasm-tinygo
	@cp "$$(tinygo env TINYGOROOT)/targets/wasm_exec.js" playground/wasm_exec_tinygo.js
	@echo "TinyGo playground built. Run 'make serve-tinygo' to start local server."

# Serve playground locally for testing
serve: playground
	@echo "Starting local server at http://localhost:8080"
	@echo "Press Ctrl+C to stop"
	@cd playground && python3 -m http.server 8080

# Serve TinyGo playground locally (uses smaller WASM binary)
serve-tinygo: playground-tinygo
	@echo "Starting local server at http://localhost:8080 (TinyGo build)"
	@echo "Press Ctrl+C to stop"
	@cd playground && python3 -m http.server 8080

.PHONY: test bench bench-update lint fmt vet clean examples check wasm playground serve build cover

# Run tests with race detector
test:
	go test -race ./...

# Run fast tests (skip slow luxical embedder tests, ~1s vs ~25s)
test-fast:
	go test ./... -short

# Run tests without race detector (faster, still includes luxical)
test-quick:
	go test ./...

# Run tests with coverage report
cover:
	go test -cover ./...

# Run tests and generate HTML coverage report
cover-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

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

# Run staticcheck (install with: go install honnef.co/go/tools/cmd/staticcheck@latest)
staticcheck:
	@which staticcheck > /dev/null 2>&1 || (echo "staticcheck not installed. Run: go install honnef.co/go/tools/cmd/staticcheck@latest" && exit 1)
	staticcheck ./...

# Format code
fmt:
	gofmt -w .

# Lint alias (runs vet and staticcheck)
lint: vet staticcheck

# Clean test cache
clean:
	go clean -testcache

# Run examples
examples:
	go test -v -run Example ./...

# Run all checks before commit
check: fmt vet staticcheck test

# Build CLI tool
build:
	go build -o levelgraph ./cmd/levelgraph

# Build WebAssembly module (standard Go)
wasm:
	GOOS=js GOARCH=wasm go build -o levelgraph.wasm ./cmd/wasm/

# Build WebAssembly module with TinyGo (smaller binary)
wasm-tinygo:
	tinygo build -o levelgraph.wasm -target wasm ./cmd/wasm/

# Build and update playground (including wasm_exec.js)
playground: wasm
	@mkdir -p playground
	@cp levelgraph.wasm playground/
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

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build CLI tool"
	@echo "  test         - Run tests with race detector (slow, ~30s)"
	@echo "  test-fast    - Run tests in short mode (~2s, skips embedder)"
	@echo "  test-quick   - Run tests without race detector (~25s)"
	@echo "  cover        - Run tests with coverage report"
	@echo "  cover-html   - Generate HTML coverage report"
	@echo "  bench        - Run benchmarks"
	@echo "  lint         - Run go vet and staticcheck"
	@echo "  wasm         - Build WebAssembly module"
	@echo "  playground   - Build and setup playground"
	@echo "  serve        - Serve playground locally"
	@echo "  check        - Run fmt, vet, staticcheck, and test"
	@echo "  clean        - Clean build artifacts"

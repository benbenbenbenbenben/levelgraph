.PHONY: test bench bench-update lint fmt vet clean examples check

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

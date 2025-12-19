.PHONY: test bench lint fmt vet clean

# Run tests with race detector
test:
	go test -race ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

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

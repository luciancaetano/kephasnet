.PHONY: test

.PHONY: test test-unit test-e2e test-stress test-coverage chat-example clean help

# Default target
help:
	@echo "Available commands:"
	@echo "  make test          - Run all tests"
	@echo "  make test-unit     - Run unit tests only"
	@echo "  make test-e2e      - Run end-to-end tests only"
	@echo "  make test-stress   - Run stress tests (requires high ulimit)"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  chat-example	 	- Run the JS chat example"
	@echo "  make clean         - Clean test cache and coverage files"

# Run all tests
test:
	@echo "==> Running all tests..."
	go test ./tests/... -v

# Run unit tests
test-unit:
	@echo "==> Running unit tests..."
	go test ./tests/unit/... -v

# Run end-to-end tests
test-e2e:
	@echo "==> Running end-to-end tests..."
	go test ./tests/e2e/... -v

# Run stress tests
test-stress:
	@echo "==> Running stress tests (this may take a while)..."
	@echo "==> Note: You may need to run 'ulimit -n 65536' first"
	cd tests/stress && go test -v -timeout 30m

# Run tests with coverage
test-coverage:
	@echo "==> Running tests with coverage..."
	go test ./tests/... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "==> Coverage report generated: coverage.html"

# Clean test cache and coverage files
clean:
	@echo "==> Cleaning test cache and coverage files..."
	go clean -testcache
	rm -f coverage.out coverage.html

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

chat-example:
	cd examples/js-chat && go run main.go

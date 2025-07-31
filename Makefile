# Variables
BINARY_NAME=lamport_timestamp
BINARY_PATH=./bin/$(BINARY_NAME)
MAIN_PATH=./main.go
GO_FILES=$(shell find . -name "*.go" -type f)

# Default target
.DEFAULT_GOAL := help

# Build the application
.PHONY: build
build: ## Build the application binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	go build -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "Binary built at $(BINARY_PATH)"

# Build for production (with optimizations)
.PHONY: build-prod
build-prod: ## Build optimized binary for production
	@echo "Building $(BINARY_NAME) for production..."
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "Production binary built at $(BINARY_PATH)"

# Run the application
.PHONY: run
run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	go run $(MAIN_PATH)

# Run with build
.PHONY: start
start: build ## Build and run the application
	@echo "Starting $(BINARY_NAME)..."
	$(BINARY_PATH)

# Run tests
.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"


# Clean build artifacts
.PHONY: clean
clean: ## Remove build artifacts and temporary files
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean -testcache
	@echo "Cleanup completed"

# Install dependencies
.PHONY: install
install: ## Download and install dependencies
	@echo "Installing dependencies..."
	go mod download
	go mod verify

# Format code
.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...
	@echo "Code formatted"






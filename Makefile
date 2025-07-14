# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=data-extraction-service
BINARY_PATH=./bin/$(BINARY_NAME)

# Build the binary
.PHONY: build
build:
	$(GOBUILD) -o $(BINARY_PATH) ./cmd

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY_PATH)
	rm -f coverage.out coverage.html

# Download dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run golangci-lint
.PHONY: lint
lint:
	golangci-lint run

# Install golangci-lint (if not already installed)
.PHONY: install-lint
install-lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)

# Run all checks (lint, test)
.PHONY: check
check: lint test

# Run the application
.PHONY: run
run: build
	$(BINARY_PATH)

# Development server (if you have a server command)
.PHONY: dev
dev:
	$(GOCMD) run ./cmd server

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  lint          - Run golangci-lint"
	@echo "  install-lint  - Install golangci-lint if not present"
	@echo "  check         - Run lint and test"
	@echo "  run           - Build and run the application"
	@echo "  dev           - Run development server"
	@echo "  help          - Show this help message"

# Default target
.DEFAULT_GOAL := help 
# Simple Makefile for Amped Go Backend

# Default target
.DEFAULT_GOAL := help

# Help command
help: ## Show available commands
	@echo "Available commands:"
	@echo ""
	@echo "  build    - Build the application binary"
	@echo "  run      - Run the application directly"
	@echo "  dev      - Start development server with live reload (Air)"
	@echo "  format   - Format code using golangci-lint"
	@echo "  lint     - Run linter using golangci-lint"
	@echo "  test     - Run all tests"
	@echo "  check    - Run format, lint, and test"
	@echo "  clean    - Remove built binary"
	@echo "  help     - Show this help message"
	@echo ""

# Build the application
all: build

build:
	@go build -o main ./cmd

# Run the application
run:
	@go run ./cmd

# Format code
format:
	@golangci-lint fmt

# Lint code
lint:
	@golangci-lint run

# Check: format, lint, and test
check: format lint test

# Test the application
test:
	@go test -v ./...

# Clean the binary
clean:
	@rm -f main

# Live Reload with Air
dev:
	@if command -v $(shell go env GOPATH)/bin/air > /dev/null; then \
	    $(shell go env GOPATH)/bin/air; \
	    echo "Watching...";\
	elif command -v air > /dev/null; then \
	    air; \
	    echo "Watching...";\
	else \
	    read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
	    if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
	        go install github.com/air-verse/air@latest; \
	        $(shell go env GOPATH)/bin/air; \
	        echo "Watching...";\
	    else \
	        echo "You chose not to install air. Exiting..."; \
	        exit 1; \
	    fi; \
	fi

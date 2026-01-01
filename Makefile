# Makefile for Actime

.PHONY: all build test fmt lint clean install help

# Variables
BINARY_NAME=actime
DAEMON_NAME=actimed
BUILD_DIR=build
CMD_DIR=cmd
GO=/usr/local/go/bin/go
GOFLAGS=-v

# Version info
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

all: build

## build: Build the project
build:
	@echo "Building $(BINARY_NAME) and $(DAEMON_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)/$(BINARY_NAME)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(DAEMON_NAME) ./$(CMD_DIR)/$(DAEMON_NAME)

## test: Run tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...

## test-coverage: Run tests with coverage
test-coverage: test
	@echo "Coverage:"
	$(GO) tool cover -func=coverage.out

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	gofmt -s -w .

## lint: Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out

## install: Install binaries to GOPATH/bin
install: build
	@echo "Installing binaries..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@cp $(BUILD_DIR)/$(DAEMON_NAME) $(GOPATH)/bin/

## run: Run the daemon
run: build
	@echo "Running daemon..."
	./$(BUILD_DIR)/$(DAEMON_NAME)

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

## help: Show this help message
help:
	@echo "Available targets:"
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## /  /'
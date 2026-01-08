# Custom Network Proxy Server

.PHONY: all build run clean test help

# Binary name
BINARY_NAME=proxy-server

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w"

all: build

## build: Compile the proxy server binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: ./$(BINARY_NAME)"

## run: Build and run the proxy server
run: build
	@echo "Starting proxy server..."
	./$(BINARY_NAME)

## clean: Remove build artifacts and logs
clean:
	@echo "Cleaning up..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f proxy.log
	@echo "Clean complete"

## test: Run the test script (requires running proxy server)
test:
	@echo "Running tests..."
	@chmod +x test_proxy.sh
	./test_proxy.sh

## tidy: Tidy up Go modules
tidy:
	$(GOMOD) tidy

## fmt: Format Go source files
fmt:
	$(GOCMD) fmt ./...

## vet: Run go vet on source files
vet:
	$(GOCMD) vet ./...

## lint: Run all code quality checks
lint: fmt vet

## help: Display this help message
help:
	@echo "Custom Network Proxy Server - Makefile Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'

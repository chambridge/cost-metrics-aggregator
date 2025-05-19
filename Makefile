# Makefile for cost-metrics-aggregator
.PHONY: all build test lint fmt vet run db-migrate clean compose-up compose-down help

# Variables
APP_NAME := cost-metrics-aggregator
BINARY := server
GO := go
PODMAN := podman
MIGRATION_DIR := db/migrations
POSTGRES_USER := costmetrics
POSTGRES_DB := costmetrics
POSTGRES_HOST := localhost
POSTGRES_PORT := 5432

# Default target
all: build


# Build the Go binary
build:
	$(GO) build -o $(BINARY) ./cmd/server/main.go

# Run tests
test:
	$(GO) test -v ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Format code
fmt:
	$(GO) fmt ./...

# Run vet
vet:
	$(GO) vet ./...

# Run the application locally
run: build
	./$(BINARY)# Clean up build artifacts

clean:
	rm -f $(BINARY)

# Start services with podman-compose
compose-up:
	$(PODMAN)-compose -f podman-compose.yaml up -d

# Stop and remove services
compose-down:
	$(PODMAN)-compose -f podman-compose.yaml down


# Show help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all           Build the application (default)"
	@echo "  build         Build the Go binary"
	@echo "  test          Run tests"
	@echo "  lint          Run linter (requires golangci-lint)"
	@echo "  fmt           Format code"
	@echo "  vet           Run go vet"
	@echo "  clean         Remove build artifacts and image"
	@echo "  compose-up    Start services with podman-compose"
	@echo "  compose-down  Stop and remove services"
	@echo "  help          Show this help message"

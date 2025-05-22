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

# Default container runtime (podman or docker)
CONTAINER_RUNTIME ?= podman

# Default image name
IMAGE_NAME ?= quay.io/chambridge/cost-metrics-aggregator:latest

# Check if Containerfile exists
CONTAINERFILE := Containerfile
ifeq (,$(wildcard $(CONTAINERFILE)))
  $(error Containerfile not found at $(CONTAINERFILE))
endif

GENERATE_SCRIPT := scripts/generate_test_upload/main.go
OUTPUT_DIR := test_upload
OUTPUT_TAR := test_upload.tar.gz

# Default target
all: build


# Build the Go binary
build:
	$(GO) build -o $(BINARY) ./cmd/server/main.go

# Build the container image
build-image:
	@command -v $(CONTAINER_RUNTIME) >/dev/null 2>&1 || { echo "Error: $(CONTAINER_RUNTIME) is not installed"; exit 1; }
	$(CONTAINER_RUNTIME) build -t $(IMAGE_NAME) -f $(CONTAINERFILE) .

# Run tests
test:
	$(GO) test -cover -v -count=1 ./...

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

# Target to generate test upload
generate-test-upload:
	rm -rf $(OUTPUT_DIR) $(OUTPUT_TAR)
	go run $(GENERATE_SCRIPT)

# Target to upload the generated test file
upload-test: generate-test-upload
	curl -X POST -F "file=@$(OUTPUT_TAR)" http://localhost:8080/api/ingres/v1/upload

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
	@echo "  generate-test-upload Generate test payload"
	@echo "  upload-test   Send generated payload to upload endpoint"
	@echo "  help          Show this help message"

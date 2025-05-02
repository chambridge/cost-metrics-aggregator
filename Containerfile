# Build stage
FROM golang:1.21 AS builder

WORKDIR /app

# Copy go.mod and go.sum (if they exist) to cache dependencies
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all Go source code directories
COPY api/ /app/api/
COPY cmd/ /app/cmd/
COPY config/ /app/config/
COPY internal/ /app/internal/
COPY scripts/ /app/scripts/

# Compile Go programs into binaries
RUN go build -o /app/server /app/cmd/server/main.go
RUN go build -o /app/scripts/create_partitions /app/scripts/create_partitions.go
RUN go build -o /app/scripts/drop_partitions /app/scripts/drop_partitions.go

# Runtime stage
FROM registry.access.redhat.com/ubi9/ubi-minimal

# Install dependencies: libpq for PostgreSQL and curl-minimal for downloading migrate
RUN microdnf install -y libpq curl-minimal tar gzip && \
    microdnf clean all

# Install migrate CLI (version 4.17.0) for amd64
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz && \
    mv migrate /usr/local/bin/migrate && \
    chmod +x /usr/local/bin/migrate

# Copy migrations and compiled binaries from builder stage
COPY --from=builder /app/internal/db/migrations /app/migrations
COPY --from=builder /app/server /app/server
COPY --from=builder /app/scripts/create_partitions /app/scripts/create_partitions
COPY --from=builder /app/scripts/drop_partitions /app/scripts/drop_partitions

RUN microdnf install -y go && \
    microdnf clean all

# Set working directory
WORKDIR /app

# Ensure binaries are executable
RUN chmod +x /app/server /app/scripts/create_partitions /app/scripts/drop_partitions

# Default command to run the server
CMD ["/app/server"]

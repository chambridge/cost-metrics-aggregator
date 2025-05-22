# Build stage
FROM registry.access.redhat.com/ubi9/go-toolset:1.21 AS builder

# Set working directory
WORKDIR /app

# Ensure /app is writable by the non-root user (UID 1001)
USER root
RUN chown 1001:0 /app && chmod 775 /app
USER 1001

# Copy go.mod and go.sum (if they exist) to cache dependencies
COPY go.mod go.sum* ./
RUN go mod download

# Copy all Go source code directories
COPY api/ /app/api/
COPY cmd/ /app/cmd/
COPY internal/ /app/internal/
COPY scripts/ /app/scripts/

# Compile Go programs into binaries
RUN go build -o /app/server /app/cmd/server/main.go
RUN go build -o /app/create /app/scripts/create/main.go
RUN go build -o /app/drop /app/scripts/drop/main.go

# Runtime stage
FROM registry.access.redhat.com/ubi9/ubi-minimal

# Install dependencies: libpq for PostgreSQL, curl-minimal for downloading migrate, tar and gzip for tar.gz handling
RUN microdnf install -y libpq curl-minimal tar gzip && \
    microdnf clean all

# Install migrate CLI (version 4.17.0) for amd64
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz && \
    mv migrate /usr/local/bin/migrate && \
    chmod +x /usr/local/bin/migrate

# Copy migrations and compiled binaries from builder stage
COPY --from=builder /app/internal/db/migrations /app/migrations
COPY --from=builder /app/server /app/server
COPY --from=builder /app/create /app/create
COPY --from=builder /app/drop /app/drop

# Set working directory
WORKDIR /app

# Ensure binaries are executable
RUN chmod +x /app/server /app/create /app/drop

# Default command to run the server
CMD ["/app/server"]
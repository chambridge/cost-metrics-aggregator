# Build stage
FROM --platform=$BUILDPLATFORM golang:1.21 AS builder

WORKDIR /app

# Copy go.mod and go.sum (if they exist) to cache dependencies
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy migrations and scripts directories (if they exist)
COPY internal/db/migrations* /app/migrations/
COPY scripts* /app/scripts/

# Runtime stage
FROM registry.access.redhat.com/ubi9/ubi-minimal

# Install dependencies: libpq for PostgreSQL and curl for downloading migrate
RUN microdnf install -y libpq curl && \
    microdnf clean all

# Install migrate CLI (version 4.17.0) based on architecture
ARG TARGETARCH
RUN if [ "$TARGETARCH" = "amd64" ]; then \
        curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz && \
        mv migrate /usr/local/bin/migrate; \
    elif [ "$TARGETARCH" = "arm64" ]; then \
        curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-arm64.tar.gz | tar xvz && \
        mv migrate /usr/local/bin/migrate; \
    else \
        echo "Unsupported architecture: $TARGETARCH" && exit 1; \
    fi && \
    chmod +x /usr/local/bin/migrate

# Copy migrations and scripts from builder stage
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/scripts /app/scripts

# Install Go for running scripts in CronJobs and initContainer
RUN microdnf install -y go && \
    microdnf clean all

# Set working directory
WORKDIR /app

# Ensure scripts are executable (if they exist)
RUN chmod +x /app/scripts/*.go 2>/dev/null || true

# Placeholder command (adjust based on your app's needs)
CMD ["sleep", "infinity"]

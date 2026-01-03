# Build stage
FROM golang:1.23-alpine AS builder

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-1.0.0}" \
    -o /app/arcana-cloud-go \
    ./cmd/server

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 arcana && \
    adduser -u 1000 -G arcana -s /bin/sh -D arcana

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/arcana-cloud-go .

# Copy config files
COPY --from=builder /app/config ./config

# Create plugins directory
RUN mkdir -p /app/plugins && chown -R arcana:arcana /app

# Switch to non-root user
USER arcana

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/arcana-cloud-go"]

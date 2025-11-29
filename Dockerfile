# Multi-stage build for production optimization
# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies with security-focused packages
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    musl-dev \
    gcc \
    && update-ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies and verify integrity
RUN go mod download && \
    go mod verify

# Copy source code
COPY . .

# Build arguments for customization
ARG APP_VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

# Set build-time variables
ARG VERSION=${APP_VERSION}
ARG BUILT=${BUILD_TIME}
ARG COMMIT=${GIT_COMMIT}

# Build the application with optimized flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -ldflags="-X main.version=${VERSION} -X main.built=${BUILT} -X main.commit=${COMMIT}" \
    -a -installsuffix cgo \
    -o main cmd/api/main.go && \
    # Verify binary was created successfully
    test -f main

# Minimal runtime stage
FROM alpine:3.18

# Install runtime dependencies (security-focused minimal set)
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    dumb-init \
    wget \
    && update-ca-certificates \
    && rm -rf /var/cache/apk/*

# Create non-root user with proper security settings
RUN addgroup -g 1001 -S erpgo && \
    adduser -u 1001 -S erpgo -G erpgo -h /app -s /bin/sh

# Set working directory
WORKDIR /app

# Copy binary from builder stage with proper permissions
COPY --from=builder --chown=erpgo:erpgo /app/main .

# Copy configuration files
COPY --from=builder --chown=erpgo:erpgo /app/configs ./configs

# Create necessary directories with proper permissions
RUN mkdir -p logs uploads temp && \
    chown -R erpgo:erpgo /app && \
    chmod 755 /app && \
    chmod 755 /app/logs /app/uploads /app/temp

# Switch to non-root user
USER erpgo

# Expose port
EXPOSE 8080

# Add labels for metadata
LABEL maintainer="ERPGo Team" \
      version="${APP_VERSION}" \
      description="ERPGo API Service" \
      org.opencontainers.image.source="https://github.com/yourorg/erpgo"

# Health check with improved reliability
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider --timeout=5 http://localhost:8080/health || exit 1

# Set environment variables for security
ENV GIN_MODE=release
ENV TZ=UTC

# Use dumb-init as PID 1 for proper signal handling
ENTRYPOINT ["dumb-init", "--"]

# Run the application
CMD ["./main"]
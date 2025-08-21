# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o runestone .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 runestone && \
    adduser -D -s /bin/sh -u 1001 -G runestone runestone

# Set working directory
WORKDIR /home/runestone

# Copy binary from builder stage
COPY --from=builder /app/runestone /usr/local/bin/runestone

# Copy example configurations
COPY --from=builder /app/examples ./examples

# Change ownership
RUN chown -R runestone:runestone /home/runestone

# Switch to non-root user
USER runestone

# Set entrypoint
ENTRYPOINT ["runestone"]

# Default command
CMD ["--help"]

# Labels
LABEL maintainer="Ataiva Software <support@ataiva.com>"
LABEL description="Runestone - Declarative, drift-aware infrastructure"
LABEL version="0.1.0"
LABEL org.opencontainers.image.source="https://github.com/ataiva-software/runestone"

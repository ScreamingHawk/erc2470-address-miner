# Multi-stage build for ERC-2470 Address Miner
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -ldflags "-s -w" -o bin/erc2470-miner ./cmd/erc2470-miner

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S miner && \
    adduser -u 1001 -S miner -G miner

# Copy binary from builder stage
COPY --from=builder /app/bin/erc2470-miner /usr/local/bin/erc2470-miner

# Make binary executable
RUN chmod +x /usr/local/bin/erc2470-miner

# Switch to non-root user
USER miner

# Set working directory
WORKDIR /home/miner

# Expose port for profiling (optional)
EXPOSE 6060

# Run the application
ENTRYPOINT ["erc2470-miner"]

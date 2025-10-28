# Multi-stage build for efficient image size
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hls-converter-api .

# Final stage - runtime image
FROM alpine:3.19

# Install FFmpeg and runtime dependencies
RUN apk add --no-cache \
    ffmpeg \
    ca-certificates \
    tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/hls-converter-api .

# Create directory for temporary video storage
RUN mkdir -p /tmp/videos && chmod 777 /tmp/videos

# Expose port
EXPOSE 8080

# Set environment variables
ENV PORT=8080
ENV GIN_MODE=release

# Run the application
CMD ["./hls-converter-api"]

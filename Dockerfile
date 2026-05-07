# Build stage
FROM golang:1.25.3-alpine AS builder

WORKDIR /build

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application into build directory
RUN mkdir -p build && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o build/myapp ./cmd/web

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create build directory
RUN mkdir -p build

# Copy the binary from builder to build directory
COPY --from=builder /build/build/myapp ./build/

# Copy any static files or templates if needed
# COPY --from=builder /build/static ./static
# COPY --from=builder /build/templates ./templates

# Create non-root user
RUN adduser -D -g '' appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 8080

CMD ["./build/myapp"]

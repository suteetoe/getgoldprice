# Build stage
FROM golang:1.25-alpine AS builder

# Install git (needed for some Go modules)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main .

# Final stage - use distroless for better security
FROM gcr.io/distroless/static-debian11:nonroot

# Copy ca-certificates from builder (distroless already has them)
# Copy the binary from builder stage
COPY --from=builder /app/main /main

# Use non-root user (already set in distroless:nonroot)
# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/main"]
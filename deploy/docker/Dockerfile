# Build stage
FROM golang:1.23.2-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy any necessary config files
COPY --from=builder /app/.env* ./

# Expose port
EXPOSE 80

# Run the binary
CMD ["./server"]

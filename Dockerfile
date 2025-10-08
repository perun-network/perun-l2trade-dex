# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates
RUN apk update && apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/server/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy web files
COPY --from=builder /app/web ./web

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./main"]
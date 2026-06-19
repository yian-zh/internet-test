# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy your tracking configuration files
COPY go.mod ./

# Copy your source code so the compiler can scan it
COPY server.go .

# Tell Go to automatically find, download, and fix the missing driver signatures
RUN go mod tidy

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server server.go

# Final stage
FROM alpine:latest

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/server .

# Copy the frontend folder
COPY frontend ./frontend

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./server"]

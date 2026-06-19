# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod
COPY go.mod ./

# Download dependencies
RUN GO111MODULE=on go mod download -x

# Copy source code and modules
COPY . .

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

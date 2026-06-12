# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod
COPY go.mod ./
# (If there were a go.sum, we would copy it here as well)
# COPY go.sum ./
# RUN go mod download

# Copy source code
COPY server.go .

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

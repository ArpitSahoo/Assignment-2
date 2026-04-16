# Build stage: compiles Go app into Linux binary
FROM golang:1.25-alpine AS builder

# Set working dir inside the builder container
WORKDIR /app

# Copy dependency files first
COPY go.mod go.sum ./

# Download Go module dependencies
RUN go mod download

# Copy the rest of the project source code into the container
COPY . .

# Build the app binary from cmd package
# CGO disabled to get static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd

# Runtime: contains what is needed to run the server
FROM alpine:latest

# Set working dir inside runtime container
WORKDIR /app

# Install CA certificates so HTTPS request work for external APIs
RUN apk add --no-cache ca-certificates

# Copy compiled binary from builder stage into runtime image
COPY --from=builder /app/server ./server

# Ensure container is listening on port 8080
EXPOSE 8080

# Start the application when container runs
CMD ["./server"]
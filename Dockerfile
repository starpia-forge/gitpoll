# Build stage
FROM golang:alpine AS builder
WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o gitpoll ./cmd/gitpoll

# Final stage
FROM alpine:latest

# Install necessary runtime dependencies (bash, openssh)
RUN apk add --no-cache bash openssh

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/gitpoll .

# Ensure the binary is executable
RUN chmod +x ./gitpoll

# Set the entrypoint
ENTRYPOINT ["./gitpoll"]

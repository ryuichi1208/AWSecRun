# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /awsecrun

# Final stage
FROM alpine:3.18

# Install CA certificates for HTTPS connections
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /awsecrun .

# Set executable as entrypoint
ENTRYPOINT ["./awsecrun"]

# Default command will show usage instructions
CMD ["--help"]

FROM golang:1.21.6-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum (if exists) first to cache dependencies
COPY go.mod ./
# COPY go.sum ./ # We'll skip this if we haven't committed the lockfile yet, but usually good practice.

# Install dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the Go app
# -o main: output binary name
# ./cmd/webapi: where our main.go is
RUN go build -o /app/wasatext ./cmd/webapi

# ==========================================
# Final Stage (Small image for production)
# ==========================================
FROM alpine:3.19

WORKDIR /app

# Install basic certificates (needed for HTTPS calls if any, though we use HTTP)
RUN apk add --no-cache ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/wasatext /app/wasatext

# Copy the frontend files
COPY --from=builder /app/webui /app/webui

# Expose the port (informative)
EXPOSE 3000

# Set the binary as entrance
CMD ["/app/wasatext"]

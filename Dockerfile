# Build stage for backend
FROM golang:1.24 AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY cmd/ cmd/
COPY service/ service/
# Copy ALL webui files (plain HTML/JS/CSS app, no build step needed)
COPY webui/ webui/

# Build the Go binary (frontend is embedded via go:embed)
RUN go build -mod=vendor -o webapi ./cmd/webapi/

# Final stage
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy backend binary (frontend is embedded inside it)
COPY --from=backend-builder /app/webapi .
# Copy demo config
COPY demo/config.yaml /app/config.yaml
# Create data directory for SQLite
RUN mkdir -p /app/data

# Set environment variables
ENV WASATEXT_DB_FILENAME=/app/data/wasatext.db
ENV WASATEXT_WEB_APIHOST=:3000

EXPOSE 3000
CMD ["./webapi"]

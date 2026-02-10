# Build stage for frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/webui
COPY webui/package.json ./
# Install dependencies using yarn and offline mirror if configured, or just install
# Since we are checking functionality, we will use yarn install specific to the environment
RUN yarn install
COPY webui/ ./
RUN yarn build
# Build stage for backend
FROM golang:1.21 AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
COPY vendor/ vendor/
# Copy source code
COPY cmd/ cmd/
COPY service/ service/
# Use vendor directory
RUN go build -mod=vendor -o webapi ./cmd/webapi/
# Final stage
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
# Copy backend binary
COPY --from=backend-builder /app/webapi .
COPY demo/config.yaml /app/config.yaml
# Copy frontend build
COPY --from=frontend-builder /app/webui/dist ./webui/dist
# Create data directory for SQLite
RUN mkdir -p /app/data
EXPOSE 3000
# Set environment variables
ENV WASATEXT_DB_FILENAME=/app/data/wasatext.db
ENV WASATEXT_WEB_APIHOST=:3000
CMD ["./webapi"]
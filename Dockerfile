# syntax=docker/dockerfile:1
ARG CACHE_BUST=20251212

# --- builder stage ---
# syntax=docker/dockerfile:1

# --- builder stage ---
FROM golang:1.23.1 AS builder

WORKDIR /app

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the API binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/api ./cmd/api

# --- runtime stage ---
FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /bin/api /app/api

# Cloud Run listens on PORT (default 8080)
ENV PORT=8080

# Run as nonroot user (distroless provides nonroot)
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/api"]

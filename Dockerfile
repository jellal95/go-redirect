ARG GO_VERSION=1.21
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /app

# Copy only go.mod and go.sum first to cache dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build binary (tidak verbose)
RUN CGO_ENABLED=0 GOOS=linux go build -o /run-app ./...

# Runtime image
FROM debian:bookworm-slim

WORKDIR /app

# Copy binary
COPY --from=builder /run-app /usr/local/bin/run-app

# Copy config and views
COPY --from=builder /app/config ./config
COPY --from=builder /app/views ./views
COPY --from=builder /app/GeoLite2-*.mmdb ./

# Reduce image size
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

CMD ["run-app"]

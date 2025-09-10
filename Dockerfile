ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm as builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /run-app .


FROM debian:bookworm

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /run-app /usr/local/bin/

# Copy necessary files and directories
COPY --from=builder /usr/src/app/config ./config
COPY --from=builder /usr/src/app/views ./views
COPY --from=builder /usr/src/app/GeoLite2-City.mmdb ./

CMD ["run-app"]

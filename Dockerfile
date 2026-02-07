# -------- Build stage --------
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git (needed for go modules sometimes)
RUN apk add --no-cache git

# Copy go mod files first for better layer caching
COPY go.mod ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -o gocorsproxy

# -------- Runtime stage --------
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/gocorsproxy /app/gocorsproxy

# Expose the app port
EXPOSE 8111

# Run the binary
ENTRYPOINT ["/app/gocorsproxy"]

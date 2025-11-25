# api-gateway/Dockerfile

FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Copy common package and api-gateway code
COPY pkg ./pkg
COPY cmd/api-gateway ./cmd/api-gateway

# Build
RUN go build -o main ./cmd/api-gateway/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
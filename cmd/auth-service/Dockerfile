FROM golang:1.25-alpine AS builder

WORKDIR /app

# 依存関係をコピー
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# ビルド
WORKDIR /app/cmd/auth-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/auth-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/bin/auth-service .

EXPOSE 50051
CMD ["./auth-service"]


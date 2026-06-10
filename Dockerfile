FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bechend-test ./cmd/server

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/bechend-test .
COPY --from=builder /app/internal/web ./internal/web
EXPOSE 8080
CMD ["./bechend-test"]
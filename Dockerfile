FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bechend-test .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/bechend-test .
COPY index.html .
EXPOSE 8080
CMD ["./bechend-test"]
# Stage 1 — build the binary
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o rpc-proxy ./cmd/proxy

# Stage 2 — minimal runtime image
FROM alpine:3.22.4

WORKDIR /app

COPY --from=builder /app/rpc-proxy .

EXPOSE 8080

ENTRYPOINT ["./rpc-proxy"]
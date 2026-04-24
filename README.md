# Blockchain RPC Proxy

A lightweight JSON-RPC 2.0 reverse proxy for [polygon.drpc.org](https://polygon.drpc.org), written in Go.

## What it does

Exposes the same JSON-RPC interface as the upstream Polygon node — single and batch requests both supported. Clients connect to the proxy exactly as they would connect to the node directly.

## Run locally

```bash
go run ./cmd/proxy
```

```bash
# Single request
curl -X POST http://localhost:8080 \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# Batch request
curl -X POST http://localhost:8080 \
  -H 'Content-Type: application/json' \
  -d '[{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1},{"jsonrpc":"2.0","method":"net_version","params":[],"id":2}]'

# Health check
curl http://localhost:8080/healthz
```

## Configuration

| Variable       | Default                    | Description           |
|----------------|----------------------------|-----------------------|
| `UPSTREAM_URL` | `https://polygon.drpc.org` | Upstream RPC node URL |
| `LISTEN_ADDR`  | `:8080`                    | Listen address        |

## Run tests

```bash
go test -v ./...
```

## Docker

```bash
docker build -t rpc-proxy .
docker run -p 8080:8080 rpc-proxy
```

## Deploy to AWS

Infrastructure is defined in `terraform/` using AWS ECS Fargate.

```bash
cd terraform
terraform init
terraform apply
```
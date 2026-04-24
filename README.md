# Blockchain RPC Proxy

A lightweight JSON-RPC 2.0 reverse proxy for [polygon.drpc.org](https://polygon.drpc.org), written in Go and deployed to AWS ECS Fargate via Terraform.

## What it does

Forwards JSON-RPC calls transparently to the upstream Polygon node. Supports single and batch requests. Clients connect to the proxy exactly as they would connect to the node directly.


## Quick start

```bash
# Run locally
go run ./cmd/proxy

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

## Make commands

| Command           | Description                                      |
|-------------------|--------------------------------------------------|
| `make test`       | Run all tests                                    |
| `make build`      | Build binary locally                             |
| `make docker-build` | Build Docker image                             |
| `make docker-run` | Run Docker container                             |
| `make ci`         | Full pipeline: test → build → docker → smoke test |

## Run full CI pipeline

```bash
make ci
```

This runs tests, builds the binary, builds the Docker image, starts the container, hits `/healthz` and `eth_blockNumber` against the real Polygon node, then stops the container.

## Deploy to AWS

Infrastructure is defined in `terraform/` using AWS ECS Fargate.

```bash
cd terraform
terraform init
terraform apply
```
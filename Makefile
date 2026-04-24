.PHONY: test build docker-build docker-run ci

IMAGE_NAME = rpc-proxy

## Run tests
test:
	go test -v -race ./...

## Build binary locally
build:
	go build -o rpc-proxy ./cmd/proxy

## Build Docker image
docker-build:
	docker build -t $(IMAGE_NAME) .

## Run Docker container
docker-run:
	docker run -p 8080:8080 $(IMAGE_NAME)

## Run the full CI pipeline: test → build → docker build → smoke test
ci: test build docker-build
	@echo "Starting container for smoke test..."
	@docker run -d -p 8080:8080 --name ci-proxy $(IMAGE_NAME)
	@sleep 2
	@echo "Testing /healthz..."
	@curl -sf http://localhost:8080/healthz | grep '"status":"ok"' || (docker rm -f ci-proxy && exit 1)
	@echo "Testing eth_blockNumber..."
	@curl -sf -X POST http://localhost:8080 \
		-H 'Content-Type: application/json' \
		-d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
		| grep '"result"' || (docker rm -f ci-proxy && exit 1)
	@docker rm -f ci-proxy
	@echo ""
	@echo "✅ All CI checks passed!"
.PHONY: help test build clean tidy server docker-build docker-push

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run tests
	go test -v -race -cover ./auth/...

build: ## Build all binaries
	go build -o bin/auth-server ./auth/cmd/server
	go build -o bin/test-puzzle-auth ./auth/cmd/test-puzzle-auth

clean: ## Clean build artifacts
	rm -rf bin/

tidy: ## Tidy and format code
	go mod tidy
	go fmt ./auth/...
	go vet ./auth/...

server: build ## Run auth server
	CONFIG_FILE=auth/auth_config.json ./bin/auth-server

docker-build: ## Build Docker image locally
	docker build -f auth/Dockerfile -t ghcr.io/status-im/proxy-common/auth:latest .

docker-push: docker-build ## Build and push Docker image to GHCR
	docker push ghcr.io/status-im/proxy-common/auth:latest

.DEFAULT_GOAL := help

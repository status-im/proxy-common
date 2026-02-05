.PHONY: help test build clean tidy server

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

.DEFAULT_GOAL := help

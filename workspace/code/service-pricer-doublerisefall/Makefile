.PHONY: help build test clean proto run docker-build

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the service binary
	go build -o bin/doublerisefall ./cmd/doublerisefall

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out

proto: ## Generate protobuf code
	buf generate

run: ## Run the service locally
	go run ./cmd/doublerisefall

docker-build: ## Build Docker image
	docker build -t service-pricer-doublerisefall:latest .

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

mod-tidy: ## Tidy go modules
	go mod tidy

mod-download: ## Download go modules
	go mod download

deps: mod-download ## Install dependencies

dev: ## Run in development mode with hot reload
	air

.DEFAULT_GOAL := help

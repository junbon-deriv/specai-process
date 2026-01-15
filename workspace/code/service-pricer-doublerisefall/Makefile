.PHONY: all build test clean proto fmt lint run

# Build variables
BINARY_NAME=doublerisefall
BUILD_DIR=bin
MAIN_PATH=./cmd/doublerisefall

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOFMT=$(GOCMD) fmt

all: proto build

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	buf generate

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run ./...

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOGET) -v ./...

# Download Go modules
mod-download:
	@echo "Downloading Go modules..."
	$(GOCMD) mod download

# Tidy Go modules
mod-tidy:
	@echo "Tidying Go modules..."
	$(GOCMD) mod tidy

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t service-pricer-doublerisefall:latest .

# Run with Docker
docker-run:
	@echo "Running Docker container..."
	docker run -p 50051:50051 service-pricer-doublerisefall:latest

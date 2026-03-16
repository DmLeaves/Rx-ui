# Rx-ui Makefile

.PHONY: all build clean test lint run install

# Variables
BINARY_NAME = rx-ui
VERSION ?= $(shell cat config/version 2>/dev/null || echo "0.0.1")
GO = go
GOFLAGS = -v
LDFLAGS = -ldflags "-X 'Rx-ui/config.version=$(VERSION)'"

# Default target
all: build

# Build binary
build:
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) .

# Install dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Run the application
run: build
	./$(BINARY_NAME) run

# Run tests
test:
	$(GO) test ./... -v

# Run linter
lint:
	@if ! command -v golangci-lint >/dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.58.0; \
	fi
	golangci-lint run

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

# Install to system (requires root)
install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/
	sudo cp x-ui.service /etc/systemd/system/
	sudo systemctl daemon-reload

# Generate API docs (if swag is installed)
swag:
	@if ! command -v swag >/dev/null; then \
		echo "Installing swag..."; \
		$(GO) install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	swag init -g main.go -o ./docs

# Docker build
docker-build:
	docker build -t rx-ui:$(VERSION) .

# Docker run
docker-run:
	docker run -d --network=host \
		-v $(PWD)/db/:/etc/rx-ui/ \
		-v $(PWD)/cert/:/root/cert/ \
		--name rx-ui --restart=unless-stopped \
		rx-ui:$(VERSION)

# Update dependencies
update:
	$(GO) get -u ./...
	$(GO) mod tidy

# Show help
help:
	@echo "Available targets:"
	@echo "  build     - Build binary"
	@echo "  deps      - Install dependencies"
	@echo "  run       - Build and run"
	@echo "  test      - Run tests"
	@echo "  lint      - Run linter"
	@echo "  clean     - Clean build artifacts"
	@echo "  install   - Install to system"
	@echo "  swag      - Generate API docs"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  update    - Update dependencies"
	@echo "  help      - Show this help"

# Version info
version:
	@echo "Rx-ui v$(VERSION)"
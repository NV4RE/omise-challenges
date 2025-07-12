.PHONY: build run clean test fmt vet deps

# Binary name
BINARY_NAME=go-tamboon

# Build the application
build:
	go build -o $(BINARY_NAME) .

# Run the application
run: build
	./$(BINARY_NAME)

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 .
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe .
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 .

# Install dependencies and build
install: deps build

# Development workflow
dev: fmt vet test build

# Help
help:
	@echo "Available targets:"
	@echo "  build     - Build the application"
	@echo "  run       - Build and run the application"
	@echo "  clean     - Clean build artifacts"
	@echo "  test      - Run tests"
	@echo "  fmt       - Format code"
	@echo "  vet       - Run go vet"
	@echo "  deps      - Download and tidy dependencies"
	@echo "  build-all - Build for multiple platforms"
	@echo "  install   - Install dependencies and build"
	@echo "  dev       - Run development workflow (fmt, vet, test, build)"
	@echo "  help      - Show this help message"
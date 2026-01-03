.PHONY: build build-all clean test install

# Binary name
BINARY=tibber

# Build directory
DIST=dist

# Version (from git tag or commit)
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Default target
build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/tibber

# Install to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/tibber

# Run tests
test:
	go test -v ./...

# Build for all platforms
build-all: build-linux build-darwin build-windows

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-amd64 ./cmd/tibber
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-arm64 ./cmd/tibber

build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-amd64 ./cmd/tibber
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-arm64 ./cmd/tibber

build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-windows-amd64.exe ./cmd/tibber

# Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -rf $(DIST)

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Tidy dependencies
tidy:
	go mod tidy

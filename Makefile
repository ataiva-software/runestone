.PHONY: build test clean install docs dev release

# Development build
dev: test build docs

# Build the binary
build:
	go build -o runestone .

# Run tests
test:
	go test -v ./...

# Generate documentation
docs: build
	./runestone docs --output docs

# Install globally
install:
	go install .

# Clean build artifacts
clean:
	rm -f runestone runestone-*

# Release build with version
release: test
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required. Use: make release VERSION=v1.0.0"; exit 1; fi
	GOOS=linux GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o runestone-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags="-X main.version=$(VERSION)" -o runestone-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o runestone-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.version=$(VERSION)" -o runestone-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o runestone-windows-amd64.exe .

# Quick test with example
example: build
	./runestone bootstrap --config examples/vpc-demo.yaml

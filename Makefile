.PHONY: build test clean run install

# Default target
build:
	go build -o contacts-tui

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f contacts-tui
	go clean

# Run the application
run: build
	./contacts-tui

# Install to GOPATH/bin
install:
	go install

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o dist/contacts-tui-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -o dist/contacts-tui-darwin-arm64
	GOOS=linux GOARCH=amd64 go build -o dist/contacts-tui-linux-amd64
	GOOS=windows GOARCH=amd64 go build -o dist/contacts-tui-windows-amd64.exe

# Create distribution directory
dist:
	mkdir -p dist

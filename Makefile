.PHONY: build run clean install test

# Build the binary
build:
	go build -o bin/gaze cmd/gaze/main.go

# Run the application
run:
	go run cmd/gaze/main.go

# Clean build artifacts
clean:
	rm -rf bin/

# Install dependencies
install:
	go mod download
	go mod tidy

# Run tests
test:
	go test ./...

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o bin/gaze-darwin-amd64 cmd/gaze/main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/gaze-darwin-arm64 cmd/gaze/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/gaze-linux-amd64 cmd/gaze/main.go
	GOOS=windows GOARCH=amd64 go build -o bin/gaze-windows-amd64.exe cmd/gaze/main.go

# Development workflow
dev: clean install build run

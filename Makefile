# Makefile for artillery-ota-server

.PHONY: build build-embed build-linux-amd64 build-linux-arm64 build-all clean test run generate

# Generate embedded files from config
generate:
	go run tools/generate-embed/main.go

# Build the regular version
build: generate
	go build -o bin/ota-server .

# Build the embedded version with embedded assets
build-embed: generate
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w -extldflags -static" -o bin/ota-server-embed .

# Build AMD64 Linux version
build-linux-amd64: generate
	GOOS=linux GOARCH=amd64 go build -o bin/ota-server-linux-amd64 .

# Build ARM64 Linux version
build-linux-arm64: generate
	GOOS=linux GOARCH=arm64 go build -o bin/ota-server-linux-arm64 .

# Build all platform versions
build-all: generate
	GOOS=linux GOARCH=amd64 go build -o bin/ota-server-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o bin/ota-server-linux-arm64 .
	GOOS=darwin GOARCH=arm64 go build -o bin/ota-server-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o bin/ota-server-windows-amd64.exe .

# Build with go generate (for development)
build-dev:
	go generate
	go build -o bin/ota-server .

clean:
	rm -f bin/ota-server bin/ota-server-embed bin/ota-server-linux-amd64 bin/ota-server-linux-arm64 bin/ota-server-darwin-arm64 bin/ota-server-windows-amd64.exe

test:
	go test ./...

run: build
	./bin/ota-server

run-dev: build-dev
	./bin/ota-server
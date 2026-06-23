.PHONY: build test lint install clean

build:
	go build -o bin/programmapper-cli ./cmd/programmapper-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/programmapper-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/programmapper-mcp ./cmd/programmapper-mcp

install-mcp:
	go install ./cmd/programmapper-mcp

build-all: build build-mcp

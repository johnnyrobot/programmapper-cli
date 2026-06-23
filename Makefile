.PHONY: build test lint install clean

build:
	go build -o bin/programmapper-pp-cli ./cmd/programmapper-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/programmapper-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/programmapper-pp-mcp ./cmd/programmapper-pp-mcp

install-mcp:
	go install ./cmd/programmapper-pp-mcp

build-all: build build-mcp

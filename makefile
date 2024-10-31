BINARY_NAME=a3n-server
GOOS ?= linux
GOARCH ?= amd64

.PHONY: install build run-dev run clean

install:
	go run cmd/db_init/main.go

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o dist/$(BINARY_NAME) cmd/server/main.go

run:
	go run cmd/server/main.go -dev

clean:
	rm -rf dist/$(BINARY_NAME)

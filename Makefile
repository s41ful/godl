.PHONY: all build run

BINARY_NAME=godl

all: build

build:
	go build -o $(BINARY_NAME) cmd/main.go

run:
	go run main.go

test:
	go test -v ./...


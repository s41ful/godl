.PHONY: all build run

BINARY_NAME=godl

all: build

build:
	go build -o $(BINARY_NAME) cmd/main.go

install:
	go build -o /usr/bin/$(BINARY_NAME) cmd/main.go

run:
	go run main.go

test:
	go test -v ./...

uninstall:
	rm -rf /usr/bin/$(BINARY_NAME)


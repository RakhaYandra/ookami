VERSION ?= 0.1.2
PREFIX ?= $(HOME)/.local
BIN := bin/ookami
LDFLAGS := -X github.com/RakhaYandra/ookami/internal/cli.Version=$(VERSION)

.PHONY: build install test vet lint

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/ookami

install: build
	install -Dm755 $(BIN) $(PREFIX)/bin/ookami

test:
	go test -race ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

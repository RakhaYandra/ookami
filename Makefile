VERSION ?= 0.1.0
BIN := bin/ookami
LDFLAGS := -X github.com/RakhaYandra/ookami/internal/cli.Version=$(VERSION)

.PHONY: build test vet lint

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/ookami

test:
	go test -race ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

# Resolve golangci-lint: prefer PATH, fall back to default GOPATH install location
GOLANGCI_LINT := $(shell which golangci-lint 2>/dev/null || echo "$(HOME)/go/bin/golangci-lint")

.PHONY: all build test vet lint

## all: run vet, tests, and linter
all: vet test lint

## build: compile the binary
build:
	go build -o paceline .

## test: run all unit tests
test:
	go test ./...

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint
lint:
	$(GOLANGCI_LINT) run ./...

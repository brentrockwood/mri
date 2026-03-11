# Makefile
# Adjust APP_NAME, MODULE, and CMD_PATH to match your project.

APP_NAME  ?= $(shell basename $(CURDIR))
MODULE    ?= $(shell go list -m 2>/dev/null || echo "module-not-initialized")
CMD_PATH  ?= ./cmd/$(APP_NAME)
BIN_DIR   ?= bin
IMAGE_TAG ?= $(APP_NAME):latest

.PHONY: all build test docker lint vet fmt clean help

## all: run vet, lint, test, and build
all: vet lint test build

## build: compile the binary to bin/<APP_NAME>
build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) $(CMD_PATH)

## test: run all tests with race detector enabled
test:
	go test -race -count=1 ./...

## docker: build the Docker image
docker:
	docker build -t $(IMAGE_TAG) .

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## vet: run go vet
vet:
	go vet ./...

## fmt: format all Go source files
fmt:
	goimports -w .

## clean: remove build artifacts
clean:
	rm -rf $(BIN_DIR)

## help: list available targets
help:
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## //'

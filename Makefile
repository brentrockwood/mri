# Makefile

APP_NAME  ?= repo-mri
CMD_PATH  ?= ./cmd/$(APP_NAME)
BIN_DIR   ?= bin
DIST_DIR  ?= dist

# Embed version info from git. Falls back to "dev" when no tags exist.
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -s -w \
             -X main.version=$(VERSION) \
             -X main.commit=$(COMMIT) \
             -X main.buildDate=$(BUILD_DATE)

# Cross-compilation targets: os/arch pairs
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

UI_DIR     ?= ui
UI_STATIC  ?= internal/report/static/report.html

.PHONY: all build dist install version test lint vet fmt clean help ui-build ui-dev

## all: vet, lint, test, build
all: vet lint test build

## ui-build: build the UI and copy report.html to internal/report/static/
ui-build:
	cd $(UI_DIR) && npm ci && npm run build
	@mkdir -p $(dir $(UI_STATIC))
	cp $(UI_DIR)/dist/index.html $(UI_STATIC)

## ui-dev: start the Vite dev server for the UI
ui-dev:
	cd $(UI_DIR) && npm run dev

## build: compile native binary to bin/<APP_NAME> (includes ui-build)
build: ui-build
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) $(CMD_PATH)

## dist: cross-compile for all platforms to dist/
dist:
	@mkdir -p $(DIST_DIR)
	$(foreach PLATFORM,$(PLATFORMS), \
		$(eval OS   := $(word 1,$(subst /, ,$(PLATFORM)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(PLATFORM)))) \
		$(eval EXT  := $(if $(filter windows,$(OS)),.exe,)) \
		$(eval OUT  := $(DIST_DIR)/$(APP_NAME)-$(OS)-$(ARCH)$(EXT)) \
		GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 \
			go build -ldflags "$(LDFLAGS)" -o $(OUT) $(CMD_PATH) && \
		echo "  built $(OUT)" || exit 1; \
	)

## install: install native binary to /usr/local/bin
install: build
	install -m 0755 $(BIN_DIR)/$(APP_NAME) /usr/local/bin/$(APP_NAME)

## version: print the resolved version string
version:
	@echo $(VERSION)

## test: run all tests with race detector enabled
test:
	go test -race -count=1 ./...

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
	rm -rf $(BIN_DIR) $(DIST_DIR)

## help: list available targets
help:
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## //'

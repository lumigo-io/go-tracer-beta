# Binary
TAG ?= dev-local
BUILD_HASH := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date -u +%Y%m%d.%H%M%S)
LDFLAGS := '-s -w -X main.BuildVersion=${BUILD_HASH} -X main.BuildTime=${BUILD_TIME}'

## Golang
GO ?= go
GO_TEST_FLAGS ?= -race

## Binaries.
GO_INSTALL = ./scripts/go_install.sh
TOOLS_BIN_DIR := $(abspath bin)

GOLANGCILINT_VER := v1.41.1
GOLANGCILINT_BIN := ./bin/golangci-lint
GOLANGCILINT_GEN := $(TOOLS_BIN_DIR)/$(GOLANGCILINT_BIN)

OUTDATED_VER := master
OUTDATED_BIN := ./bin/go-mod-outdated
OUTDATED_GEN := $(TOOLS_BIN_DIR)/$(OUTDATED_BIN)

.PHONY: check-style
## check-style: Runs govet and gofmt against all packages.
check-style: govet lint
	@echo Checking for style guide compliance

.PHONY: clean
## clean: deletes all
clean:
	rm -rf build/_output/bin/
	rm -rf bin

.PHONY: vet
## govet: Runs govet against all packages.
govet:
	@echo Running govet
	$(GO) vet ./...
	@echo Govet success

.PHONY: lint
## lint: linting golang
lint:
	@echo Running lint
	@if ! [ -x "$$(command -v ./bin/golangci-lint)" ]; then \
		echo "\n\ngolangci-lint is not installed. Please use setup with make or see https://github.com/golangci/golangci-lint#install."; \
		exit 1; \
	fi; \
	${GOLANGCILINT_BIN} run -E gofmt --timeout 5m

.PHONY: setup
## setup: installs golangci-lint
setup:
	@echo Install golang-ci
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s ${GOLANGCILINT_VER}
	cp ./bin/golangci-lint ${GOPATH}/bin/

	@echo Install go-mod-outdated
	go get -u github.com/psampaz/go-mod-outdated
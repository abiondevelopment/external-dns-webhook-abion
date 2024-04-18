GO ?= $(shell which go)
OS ?= $(shell $(GO) env GOOS)
ARCH ?= $(shell $(GO) env GOARCH)

ARTIFACT_NAME := external-dns-webhook-abion
IMAGE_NAME := "abiondevelopment/external-dns-webhook-abion"
IMAGE_TAG := "1.0.0"

OUT := $(shell pwd)/_out

## General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

show: ## Show variables
	@echo "OS: $(OS)"
	@echo "ARCH: $(ARCH)"
	@echo "ARTIFACT_NAME: $(ARTIFACT_NAME)"
	@echo "IMAGE_NAME: $(IMAGE_NAME)"
	@echo "IMAGE_TAG: $(IMAGE_TAG)"


## Code analysis

.PHONY: fmt
fmt: ## Run gofumpt against code.
	go run mvdan.cc/gofumpt -w .

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint against code.
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run --timeout 2m

.PHONY: static-analysis
static-analysis: lint vet ## Run static analysis against code.

## Test

.PHONY: test
test:
	go test -v ./...

## Docker

.PHONY: build
build:
	docker build -t "$(IMAGE_NAME):latest" -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

.PHONY: push
push:
	docker push "$(IMAGE_NAME):$(IMAGE_TAG)"

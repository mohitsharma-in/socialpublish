BINARY_DIR := bin
IMAGE_BASE ?= ghcr.io/mohitsharma-in/socialpublish
GO := go
GOLANGCI_LINT := golangci-lint
GOLANGCI_LINT_VERSION := v2.12.2

.PHONY: all test lint build docker-build k8s-apply fmt clean

all: test lint build

test:
	$(GO) test ./...

lint:
	if ! command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then $(GO) install github.com/golangci/golangci-lint/v2/cmd/$(GOLANGCI_LINT)@$(GOLANGCI_LINT_VERSION); fi
	$(GOLANGCI_LINT) run ./...

build:
	mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY_DIR)/server ./cmd/server
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY_DIR)/worker ./cmd/worker
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY_DIR)/migrate ./cmd/migrate

docker-build:
	docker build --target server -t $(IMAGE_BASE):server-latest .
	docker build --target worker -t $(IMAGE_BASE):worker-latest .
	docker build --target migrate -t $(IMAGE_BASE):migrate-latest .

k8s-apply:
	kubectl apply -k deploy/k8s

fmt:
	$(GO) fmt ./...

clean:
	rm -rf $(BINARY_DIR)

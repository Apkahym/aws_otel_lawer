.PHONY: help build test lint package clean install-tools all

# Variables
BINARY_NAME=otel-wrapper
DIST_DIR=dist
VERSION?=dev

help: ## Mostrar ayuda
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

install-tools: ## Instalar herramientas de desarrollo
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

build: ## Compilar binarios para amd64 y arm64
	@bash scripts/build.sh

test: ## Ejecutar tests unitarios
	@echo "Running tests..."
	@go test -v -race -cover ./...

lint: ## Ejecutar linter
	@echo "Running linter..."
	@golangci-lint run ./... --timeout=5m

package: build ## Empaquetar Lambda Layers
	@bash scripts/package.sh

clean: ## Limpiar artefactos de build
	@echo "Cleaning..."
	@rm -rf $(DIST_DIR) layer/

all: lint test build package ## Ejecutar todo el pipeline

.DEFAULT_GOAL := help

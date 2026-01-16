#!/bin/bash
set -euo pipefail

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🔨 Building OTEL Lambda Wrapper...${NC}"

# Variables
OUTPUT_DIR="dist"
BINARY_NAME="otel-wrapper"
VERSION="${VERSION:-dev}"

# Limpiar directorio de salida
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Build para linux/amd64
echo -e "${YELLOW}Building for linux/amd64...${NC}"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
  -ldflags="-s -w -X main.Version=${VERSION}" \
  -trimpath \
  -o "$OUTPUT_DIR/$BINARY_NAME-amd64" \
  ./cmd/wrapper

# Build para linux/arm64
echo -e "${YELLOW}Building for linux/arm64...${NC}"
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
  -ldflags="-s -w -X main.Version=${VERSION}" \
  -trimpath \
  -o "$OUTPUT_DIR/$BINARY_NAME-arm64" \
  ./cmd/wrapper

echo -e "${GREEN}✅ Build completed successfully!${NC}"
echo ""
echo "Binaries created:"
ls -lh "$OUTPUT_DIR"

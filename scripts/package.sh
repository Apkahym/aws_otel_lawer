#!/bin/bash
set -euo pipefail

# Colores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}📦 Packaging Lambda Layers...${NC}"

DIST_DIR="dist"
LAYER_DIR="layer"

if [ ! -d "$DIST_DIR" ]; then
  echo "Error: dist/ directory not found. Run 'make build' first."
  exit 1
fi

# Empaquetar para cada arquitectura
for ARCH in amd64 arm64; do
  echo -e "${YELLOW}Packaging layer for $ARCH...${NC}"

  LAYER_PATH="$LAYER_DIR/$ARCH"
  mkdir -p "$LAYER_PATH/bin"

  # Copiar binario
  cp "$DIST_DIR/otel-wrapper-$ARCH" "$LAYER_PATH/bin/otel-wrapper"
  chmod +x "$LAYER_PATH/bin/otel-wrapper"

  # Crear ZIP
  cd "$LAYER_PATH"
  zip -r -q "../../$DIST_DIR/otel-layer-$ARCH.zip" .
  cd - > /dev/null

  # Limpiar
  rm -rf "$LAYER_PATH"
done

echo -e "${GREEN}✅ Layers packaged successfully!${NC}"
echo ""
echo "Layer artifacts:"
ls -lh "$DIST_DIR"/*.zip

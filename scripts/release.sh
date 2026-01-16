#!/bin/bash
set -euo pipefail

VERSION="${1:-}"

if [ -z "$VERSION" ]; then
  echo "Usage: $0 <version>"
  echo "Example: $0 v0.1.0"
  exit 1
fi

echo "Creating release $VERSION"

# Verificar que estamos en main
BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$BRANCH" != "main" ]; then
  echo "Error: Must be on main branch"
  exit 1
fi

# Verificar que no hay cambios sin commit
if [ -n "$(git status --porcelain)" ]; then
  echo "Error: Working directory not clean"
  exit 1
fi

# Crear tag
git tag -a "$VERSION" -m "Release $VERSION"

# Push tag
git push origin "$VERSION"

echo "✅ Tag $VERSION created and pushed"
echo "GitHub Actions will build and create the release automatically"

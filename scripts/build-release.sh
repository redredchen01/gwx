#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-0.1.0}"
BUILD_DIR="dist"
MODULE="./cmd/gwx"
LDFLAGS="-s -w -X github.com/user/gwx/internal/cmd.version=${VERSION}"

rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

TARGETS=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
  "windows/arm64"
)

for target in "${TARGETS[@]}"; do
  GOOS="${target%%/*}"
  GOARCH="${target##*/}"
  EXT=""
  [ "$GOOS" = "windows" ] && EXT=".exe"

  OUTPUT="${BUILD_DIR}/gwx_${VERSION}_${GOOS}_${GOARCH}${EXT}"
  echo "→ Building ${GOOS}/${GOARCH}..."
  GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o "$OUTPUT" "$MODULE"
  echo "  ✓ $OUTPUT ($(du -h "$OUTPUT" | awk '{print $1}'))"
done

echo ""
echo "✓ Built ${#TARGETS[@]} binaries in ${BUILD_DIR}/"
ls -lh "$BUILD_DIR/"

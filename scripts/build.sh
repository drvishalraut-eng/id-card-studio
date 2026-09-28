#!/usr/bin/env bash
# Runs the test suite, then cross-compiles idcard for every supported
# platform into dist/<os>-<arch>/idcard(.exe) — one self-contained,
# ready-to-copy folder per platform.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

echo "Running go vet..."
go vet ./...

echo "Running go test..."
go test ./...

dist="$root/dist"
rm -rf "$dist"

build() {
  local goos="$1" goarch="$2" out="$3"
  echo "Building $goos/$goarch..."
  local dir="$dist/$goos-$goarch"
  mkdir -p "$dir"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -o "$dir/$out" .
}

build windows amd64 idcard.exe
build darwin  arm64 idcard
build darwin  amd64 idcard
build linux   amd64 idcard

echo
echo "Built binaries:"
find "$dist" -type f -exec ls -lh {} \;

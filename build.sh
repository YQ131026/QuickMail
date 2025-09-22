#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
OUTPUT=${1:-"$ROOT_DIR/bin/quickmail"}

mkdir -p "$(dirname "$OUTPUT")"
export GOCACHE="${GOCACHE:-$ROOT_DIR/.gocache}"
mkdir -p "$GOCACHE"

printf '>> building QuickMail to %s\n' "$OUTPUT"
cd "$ROOT_DIR"
go build -o "$OUTPUT" ./cmd/quickmail
printf '>> build complete\n'

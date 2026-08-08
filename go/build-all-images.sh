#!/usr/bin/env bash
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "Building nasfile image..."
cd "$SCRIPT_DIR/nas/web" && ./build.sh

echo "All images built."

#!/usr/bin/env bash
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MODE="${1:-local}"
YAML="$SCRIPT_DIR/nasfile-${MODE}.yaml"

if [ ! -f "$YAML" ]; then
    echo "Unknown mode: $MODE (expected: local, baremetal, gke, kind)"
    exit 1
fi

echo "Undeploying nasfile ($MODE)..."
kubectl delete -f "$YAML" --ignore-not-found
echo "nasfile undeployed."

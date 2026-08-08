#!/usr/bin/env bash
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MODE="${1:-local}"
YAML="$SCRIPT_DIR/nasfile-${MODE}.yaml"

if [ ! -f "$YAML" ]; then
    echo "Unknown mode: $MODE (expected: local, baremetal, gke, kind)"
    exit 1
fi

echo "Deploying nasfile ($MODE)..."
kubectl apply -f "$YAML"
echo "Waiting for nasfile rollout..."
kubectl -n nasfile rollout status daemonset/nasfile 2>/dev/null || \
kubectl -n nasfile rollout status statefulset/nasfile 2>/dev/null
echo "nasfile deployed."

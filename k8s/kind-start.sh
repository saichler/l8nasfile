#!/usr/bin/env bash
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLUSTER_NAME="nasfile"

if ! command -v kind &>/dev/null; then
    echo "Installing kind..."
    go install sigs.k8s.io/kind@latest
fi

echo "Creating KIND cluster '$CLUSTER_NAME'..."
cat <<EOF | kind create cluster --name "$CLUSTER_NAME" --config -
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
EOF

echo "Loading images into KIND cluster..."
kind load docker-image saichler/nasfile:latest --name "$CLUSTER_NAME"

echo "Deploying nasfile..."
kubectl apply -f "$SCRIPT_DIR/nasfile-kind.yaml"

echo "Waiting for rollout..."
kubectl -n nasfile rollout status statefulset/nasfile --timeout=120s

echo "KIND cluster '$CLUSTER_NAME' is ready."

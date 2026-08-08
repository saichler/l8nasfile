#!/usr/bin/env bash
set -e
CLUSTER_NAME="nasfile"

echo "Deleting KIND cluster '$CLUSTER_NAME'..."
kind delete cluster --name "$CLUSTER_NAME"
echo "KIND cluster deleted."

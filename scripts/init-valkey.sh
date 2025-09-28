#!/bin/bash
set -e

echo "Checking if Valkey cluster already exists..."
if valkey-cli -h valkey-0 -p 6379 cluster info | grep -q "cluster_state:ok"; then
    echo "Cluster already initialized, skipping."
    exit 0
fi

echo "Cluster not found, creating..."
valkey-cli --cluster create \
    valkey-0:6379 valkey-1:6379 valkey-2:6379 \
    --cluster-replicas 0 --cluster-yes

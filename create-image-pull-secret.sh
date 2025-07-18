#!/bin/bash

# Script to create GitHub Container Registry image pull secret
# This script prompts for GitHub PAT and email, then creates the secret

set -e

echo "=== GitHub Container Registry Image Pull Secret Setup ==="
echo ""

# Check if namespace exists
if ! kubectl get namespace mimir-insights >/dev/null 2>&1; then
    echo "Creating namespace mimir-insights..."
    kubectl create namespace mimir-insights
fi

# Prompt for GitHub PAT
echo "Please enter your GitHub Personal Access Token (PAT):"
echo "Note: The token should have 'read:packages' scope"
read -s GITHUB_PAT
echo ""

# Prompt for email
echo "Please enter your GitHub email address:"
read GITHUB_EMAIL

# Validate inputs
if [ -z "$GITHUB_PAT" ]; then
    echo "Error: GitHub PAT is required"
    exit 1
fi

if [ -z "$GITHUB_EMAIL" ]; then
    echo "Error: GitHub email is required"
    exit 1
fi

echo "Creating image pull secret..."
kubectl create secret docker-registry ghcr-secret \
    --docker-server=ghcr.io \
    --docker-username=akshaydubey29 \
    --docker-password="$GITHUB_PAT" \
    --docker-email="$GITHUB_EMAIL" \
    -n mimir-insights \
    --dry-run=client -o yaml | kubectl apply -f -

echo ""
echo "✅ Image pull secret 'ghcr-secret' created successfully!"
echo ""
echo "Next steps:"
echo "1. Update your Helm values to use this secret"
echo "2. Redeploy your application"
echo ""
echo "To update values, add this to your imagePullSecrets:"
echo "imagePullSecrets:"
echo "  - name: ghcr-secret" 
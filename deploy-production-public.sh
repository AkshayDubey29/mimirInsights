#!/bin/bash

# Production Deployment Script for Public Images
# This script deploys MimirInsights using public GitHub Container Registry images

set -e

echo "=== MimirInsights Production Deployment (Public Images) ==="
echo ""

# Configuration
NAMESPACE="mimir-insights"
RELEASE_NAME="mimir-insights"
VALUES_FILE="./deployments/helm-chart/values-production-final.yaml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed or not in PATH"
    exit 1
fi

# Check if helm is available
if ! command -v helm &> /dev/null; then
    print_error "helm is not installed or not in PATH"
    exit 1
fi

# Check if values file exists
if [ ! -f "$VALUES_FILE" ]; then
    print_error "Values file not found: $VALUES_FILE"
    exit 1
fi

print_status "Starting production deployment with public images..."

# Create namespace if it doesn't exist
if ! kubectl get namespace $NAMESPACE >/dev/null 2>&1; then
    print_status "Creating namespace: $NAMESPACE"
    kubectl create namespace $NAMESPACE
else
    print_status "Namespace $NAMESPACE already exists"
fi

# Check if release already exists
if helm list -n $NAMESPACE | grep -q $RELEASE_NAME; then
    print_warning "Release $RELEASE_NAME already exists. Upgrading..."
    ACTION="upgrade"
else
    print_status "Installing new release: $RELEASE_NAME"
    ACTION="install"
fi

# Deploy using Helm
print_status "Deploying with Helm..."
helm $ACTION $RELEASE_NAME ./deployments/helm-chart \
    -f $VALUES_FILE \
    -n $NAMESPACE \
    --wait \
    --timeout=10m

if [ $? -eq 0 ]; then
    print_status "✅ Deployment completed successfully!"
else
    print_error "❌ Deployment failed!"
    exit 1
fi

# Wait for pods to be ready
print_status "Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mimir-insights -n $NAMESPACE --timeout=300s

# Show deployment status
echo ""
print_status "Deployment Status:"
kubectl get pods -n $NAMESPACE

echo ""
print_status "Services:"
kubectl get services -n $NAMESPACE

echo ""
print_status "Deployment completed!"
echo ""
echo "Next steps:"
echo "1. Check pod status: kubectl get pods -n $NAMESPACE"
echo "2. View logs: kubectl logs -f deployment/mimir-insights-backend -n $NAMESPACE"
echo "3. Port forward frontend: kubectl port-forward service/mimir-insights-frontend 8081:80 -n $NAMESPACE"
echo "4. Port forward backend: kubectl port-forward service/mimir-insights-backend 8080:8080 -n $NAMESPACE"
echo ""
echo "Access the application at:"
echo "- Frontend: http://localhost:8081"
echo "- Backend API: http://localhost:8080" 
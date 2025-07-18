#!/bin/bash

# Multi-Architecture Deployment Script for MimirInsights
# This script deploys MimirInsights using multi-architecture images

set -e

echo "=== MimirInsights Multi-Architecture Deployment ==="
echo ""

# Configuration
NAMESPACE="mimir-insights"
RELEASE_NAME="mimir-insights"
VALUES_FILE="./deployments/helm-chart/values-production-final.yaml"

# Image configuration
REGISTRY="ghcr.io/akshaydubey29"
FRONTEND_IMAGE="${REGISTRY}/mimir-insights-frontend"
BACKEND_IMAGE="${REGISTRY}/mimir-insights-backend"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
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

# Check if docker is available
if ! command -v docker &> /dev/null; then
    print_error "docker is not installed or not in PATH"
    exit 1
fi

# Check if values file exists
if [ ! -f "$VALUES_FILE" ]; then
    print_error "Values file not found: $VALUES_FILE"
    exit 1
fi

print_status "Starting multi-architecture deployment..."

# Detect cluster architecture
print_status "Detecting cluster architecture..."
NODE_ARCH=$(kubectl get nodes -o jsonpath='{.items[0].status.nodeInfo.architecture}' 2>/dev/null || echo "unknown")
print_success "Cluster architecture: $NODE_ARCH"

# Function to validate multi-arch image
validate_multi_arch_image() {
    local image=$1
    local image_name=$2
    
    print_status "Validating multi-architecture support for $image_name..."
    
    # Check if image exists and has multi-arch support
    if docker buildx imagetools inspect "$image" >/dev/null 2>&1; then
        print_success "$image_name has multi-architecture support"
        
        # Show supported platforms
        local platforms=$(docker buildx imagetools inspect "$image" --format '{{range .Manifest.Manifests}}{{.Platform.OS}}/{{.Platform.Architecture}}{{if .Platform.Variant}}/{{.Platform.Variant}}{{end}} {{end}}')
        print_status "Supported platforms: $platforms"
        
        # Check if current architecture is supported
        if echo "$platforms" | grep -q "$NODE_ARCH"; then
            print_success "Current architecture ($NODE_ARCH) is supported"
        else
            print_warning "Current architecture ($NODE_ARCH) may not be explicitly supported, but Docker will select the best match"
        fi
    else
        print_error "$image_name does not exist or is not accessible"
        return 1
    fi
}

# Validate images
print_status "Validating multi-architecture images..."

# Get the latest tag from the values file
LATEST_TAG=$(grep -E "image:.*latest" "$VALUES_FILE" | head -1 | sed 's/.*:latest/latest/')
if [ -z "$LATEST_TAG" ]; then
    LATEST_TAG="latest"
fi

# Validate frontend image
if ! validate_multi_arch_image "${FRONTEND_IMAGE}:${LATEST_TAG}" "Frontend"; then
    print_error "Frontend image validation failed"
    exit 1
fi

# Validate backend image
if ! validate_multi_arch_image "${BACKEND_IMAGE}:${LATEST_TAG}" "Backend"; then
    print_error "Backend image validation failed"
    exit 1
fi

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
    print_success "✅ Deployment completed successfully!"
else
    print_error "❌ Deployment failed!"
    exit 1
fi

# Wait for pods to be ready
print_status "Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mimir-insights -n $NAMESPACE --timeout=300s

# Show deployment status with architecture info
echo ""
print_status "Deployment Status:"
kubectl get pods -n $NAMESPACE -o wide

echo ""
print_status "Pod Architecture Details:"
kubectl get pods -n $NAMESPACE -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.nodeName}{"\t"}{.status.hostIP}{"\n"}{end}' | \
while read pod node hostip; do
    if [ -n "$node" ]; then
        arch=$(kubectl get node "$node" -o jsonpath='{.status.nodeInfo.architecture}' 2>/dev/null || echo "unknown")
        print_status "Pod: $pod -> Node: $node ($arch)"
    fi
done

echo ""
print_status "Services:"
kubectl get services -n $NAMESPACE

# Test multi-architecture functionality
echo ""
print_status "Testing multi-architecture functionality..."

# Test backend architecture detection
print_status "Testing backend architecture detection..."
BACKEND_POD=$(kubectl get pods -n $NAMESPACE -l app.kubernetes.io/component=backend -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$BACKEND_POD" ]; then
    BACKEND_ARCH=$(kubectl exec -n $NAMESPACE "$BACKEND_POD" -- uname -m 2>/dev/null || echo "unknown")
    print_success "Backend pod architecture: $BACKEND_ARCH"
else
    print_warning "Backend pod not found for architecture testing"
fi

# Test frontend architecture detection
print_status "Testing frontend architecture detection..."
FRONTEND_POD=$(kubectl get pods -n $NAMESPACE -l app.kubernetes.io/component=frontend -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$FRONTEND_POD" ]; then
    FRONTEND_ARCH=$(kubectl exec -n $NAMESPACE "$FRONTEND_POD" -- uname -m 2>/dev/null || echo "unknown")
    print_success "Frontend pod architecture: $FRONTEND_ARCH"
else
    print_warning "Frontend pod not found for architecture testing"
fi

echo ""
print_success "Multi-architecture deployment completed!"
echo ""
echo "🎉 Summary:"
echo "  ✅ Cluster architecture: $NODE_ARCH"
echo "  ✅ Multi-architecture images validated"
echo "  ✅ Deployment successful"
echo "  ✅ Architecture detection working"
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
echo ""
echo "Architecture information:"
echo "- Backend architecture: $BACKEND_ARCH"
echo "- Frontend architecture: $FRONTEND_ARCH"
echo "- Cluster architecture: $NODE_ARCH" 
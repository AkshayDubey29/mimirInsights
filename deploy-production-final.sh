#!/bin/bash

# Production Deployment Script for MimirInsights
# Uses values-production-final.yaml with monitoring and LLM disabled

set -e

echo "🚀 Starting MimirInsights Production Deployment..."

# Configuration
NAMESPACE="mimir-insights"
RELEASE_NAME="mimir-insights"
VALUES_FILE="deployments/helm-chart/values-production-final.yaml"
CHART_PATH="deployments/helm-chart"

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

# Check if values file exists
if [ ! -f "$VALUES_FILE" ]; then
    print_error "Values file not found: $VALUES_FILE"
    exit 1
fi

print_status "Checking cluster connectivity..."
if ! kubectl cluster-info &> /dev/null; then
    print_error "Cannot connect to Kubernetes cluster"
    exit 1
fi

print_success "Connected to cluster: $(kubectl config current-context)"

# Create namespace if it doesn't exist
print_status "Creating namespace $NAMESPACE if it doesn't exist..."
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# Check if release already exists
if helm list -n $NAMESPACE | grep -q $RELEASE_NAME; then
    print_warning "Release $RELEASE_NAME already exists. Upgrading..."
    ACTION="upgrade"
else
    print_status "Installing new release $RELEASE_NAME..."
    ACTION="install"
fi

# Deploy using Helm
print_status "Deploying with Helm using $VALUES_FILE..."

if [ "$ACTION" = "upgrade" ]; then
    helm upgrade $RELEASE_NAME $CHART_PATH \
        --namespace $NAMESPACE \
        --values $VALUES_FILE \
        --wait \
        --timeout 10m \
        --atomic
else
    helm install $RELEASE_NAME $CHART_PATH \
        --namespace $NAMESPACE \
        --values $VALUES_FILE \
        --wait \
        --timeout 10m \
        --atomic
fi

print_success "Helm deployment completed successfully!"

# Wait for pods to be ready
print_status "Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mimir-insights -n $NAMESPACE --timeout=300s

# Check pod status
print_status "Checking pod status..."
kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=mimir-insights

# Check services
print_status "Checking services..."
kubectl get svc -n $NAMESPACE

# Check ingress if enabled
if grep -q "enabled: true" $VALUES_FILE | grep -q "ingress"; then
    print_status "Checking ingress..."
    kubectl get ingress -n $NAMESPACE
fi

# Health check
print_status "Performing health check..."
sleep 10

# Check if backend is responding
BACKEND_POD=$(kubectl get pods -n $NAMESPACE -l app.kubernetes.io/component=backend -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$BACKEND_POD" ]; then
    print_status "Testing backend API..."
    if kubectl exec -n $NAMESPACE $BACKEND_POD -- curl -f http://localhost:8080/health 2>/dev/null; then
        print_success "Backend health check passed"
    else
        print_warning "Backend health check failed"
    fi
fi

# Check if frontend is responding
FRONTEND_POD=$(kubectl get pods -n $NAMESPACE -l app.kubernetes.io/component=frontend -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$FRONTEND_POD" ]; then
    print_status "Testing frontend..."
    if kubectl exec -n $NAMESPACE $FRONTEND_POD -- curl -f http://localhost:80 2>/dev/null; then
        print_success "Frontend health check passed"
    else
        print_warning "Frontend health check failed"
    fi
fi

# Display access information
print_success "Deployment completed successfully!"
echo ""
print_status "Access Information:"
echo "  Namespace: $NAMESPACE"
echo "  Release: $RELEASE_NAME"
echo "  Backend Service: mimir-insights-backend.$NAMESPACE.svc.cluster.local:8080"
echo "  Frontend Service: mimir-insights-frontend.$NAMESPACE.svc.cluster.local:80"
echo ""
print_status "To access the application:"
echo "  1. Port forward frontend: kubectl port-forward svc/mimir-insights-frontend 8081:80 -n $NAMESPACE"
echo "  2. Port forward backend: kubectl port-forward svc/mimir-insights-backend 8080:8080 -n $NAMESPACE"
echo "  3. Open browser: http://localhost:8081"
echo ""
print_status "To check logs:"
echo "  Backend: kubectl logs -f deployment/mimir-insights-backend -n $NAMESPACE"
echo "  Frontend: kubectl logs -f deployment/mimir-insights-frontend -n $NAMESPACE"
echo ""
print_status "To uninstall:"
echo "  helm uninstall $RELEASE_NAME -n $NAMESPACE"
echo ""

print_success "🎉 MimirInsights is now deployed and ready for production use!" 
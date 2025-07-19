#!/bin/bash

# ==============================================================================
# Deploy from CI/CD Script for MimirInsights
# Updates values file with timestamp and deploys to local kind cluster
# ==============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Configuration
VALUES_FILE="deployments/helm-chart/values-production-final.yaml"
HELM_CHART_DIR="deployments/helm-chart"
NAMESPACE="mimir-insights"
RELEASE_NAME="mimir-insights"

# Check if timestamp is provided as argument
if [ $# -eq 0 ]; then
    print_error "Usage: $0 <timestamp>"
    print_error "Example: $0 20250719-141012"
    print_error ""
    print_error "This script will:"
    print_error "1. Update values-production-final.yaml with the timestamp"
    print_error "2. Deploy to local kind cluster using Helm"
    print_error "3. Setup port forwarding for backend and frontend"
    print_error "4. Verify deployment status"
    exit 1
fi

TIMESTAMP=$1

# Validate timestamp format (YYYYMMDD-HHMMSS)
if [[ ! $TIMESTAMP =~ ^[0-9]{8}-[0-9]{6}$ ]]; then
    print_error "Invalid timestamp format. Expected: YYYYMMDD-HHMMSS"
    print_error "Example: 20250719-141012"
    exit 1
fi

echo "🚀 MimirInsights Deployment from CI/CD"
echo "📅 Timestamp: $TIMESTAMP"
echo ""

# Step 1: Update values file
print_status "Step 1: Updating values file with timestamp $TIMESTAMP"

# Check if values file exists
if [ ! -f "$VALUES_FILE" ]; then
    print_error "Values file not found: $VALUES_FILE"
    exit 1
fi

# Create backup
BACKUP_FILE="${VALUES_FILE}.backup.$(date +%Y%m%d-%H%M%S)"
cp "$VALUES_FILE" "$BACKUP_FILE"
print_status "Created backup: $BACKUP_FILE"

# Update backend image
print_status "Updating backend image..."
sed -i.bak "s|image: ghcr.io/akshaydubey29/mimir-insights-backend-[0-9]\{8\}-[0-9]\{6\}|image: ghcr.io/akshaydubey29/mimir-insights-backend-$TIMESTAMP|g" "$VALUES_FILE"

# Update frontend image
print_status "Updating frontend image..."
sed -i.bak "s|image: ghcr.io/akshaydubey29/mimir-insights-frontend-[0-9]\{8\}-[0-9]\{6\}|image: ghcr.io/akshaydubey29/mimir-insights-frontend-$TIMESTAMP|g" "$VALUES_FILE"

# Remove temporary backup files
rm -f "${VALUES_FILE}.bak"

# Verify the changes
BACKEND_IMAGE=$(grep -A 1 "backend:" "$VALUES_FILE" | grep "image:" | sed 's/.*image: //')
FRONTEND_IMAGE=$(grep -A 1 "frontend:" "$VALUES_FILE" | grep "image:" | sed 's/.*image: //')

print_success "Values file updated successfully!"
echo "  Backend:  $BACKEND_IMAGE"
echo "  Frontend: $FRONTEND_IMAGE"
echo ""

# Step 2: Check kind cluster status
print_status "Step 2: Checking kind cluster status"

if ! kubectl cluster-info &> /dev/null; then
    print_error "Kubernetes cluster is not accessible"
    print_error "Please ensure your kind cluster is running:"
    print_error "  kind create cluster --name mimir-insights"
    exit 1
fi

print_success "Kubernetes cluster is accessible"

# Step 3: Deploy using Helm
print_status "Step 3: Deploying to kind cluster using Helm"

# Check if Helm is installed
if ! command -v helm &> /dev/null; then
    print_error "Helm is not installed or not in PATH"
    exit 1
fi

# Check if Helm chart directory exists
if [ ! -d "$HELM_CHART_DIR" ]; then
    print_error "Helm chart directory not found: $HELM_CHART_DIR"
    exit 1
fi

# Deploy using Helm
print_status "Deploying MimirInsights to namespace: $NAMESPACE"
helm upgrade --install $RELEASE_NAME $HELM_CHART_DIR \
    --namespace $NAMESPACE \
    --create-namespace \
    --values $VALUES_FILE \
    --wait \
    --timeout 10m

print_success "Helm deployment completed successfully!"
echo ""

# Step 4: Verify deployment
print_status "Step 4: Verifying deployment status"

# Wait for pods to be ready
print_status "Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mimir-insights -n $NAMESPACE --timeout=300s

# Check pod status
print_status "Pod status:"
kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=mimir-insights

# Check service status
print_status "Service status:"
kubectl get svc -n $NAMESPACE -l app.kubernetes.io/name=mimir-insights

echo ""

# Step 5: Setup port forwarding
print_status "Step 5: Setting up port forwarding"

# Kill any existing port-forward processes
print_status "Killing existing port-forward processes..."
pkill -f "kubectl port-forward.*mimir-insights" || true
sleep 2

# Setup port forwarding for backend
print_status "Setting up port forwarding for backend (port 8080)..."
kubectl port-forward -n $NAMESPACE svc/mimir-insights-backend 8080:8080 &
BACKEND_PF_PID=$!

# Setup port forwarding for frontend
print_status "Setting up port forwarding for frontend (port 8081)..."
kubectl port-forward -n $NAMESPACE svc/mimir-insights-frontend 8081:80 &
FRONTEND_PF_PID=$!

# Wait for port forwarding to be established
sleep 5

# Check if port forwarding is working
if ! curl -s http://localhost:8080/api/health &> /dev/null; then
    print_warning "Backend health check failed, but continuing..."
else
    print_success "Backend is responding on port 8080"
fi

if ! curl -s http://localhost:8081 &> /dev/null; then
    print_warning "Frontend health check failed, but continuing..."
else
    print_success "Frontend is responding on port 8081"
fi

echo ""

# Step 6: Final status
print_success "🎉 Deployment completed successfully!"
echo ""
echo "📦 Deployed Images:"
echo "  Backend:  $BACKEND_IMAGE"
echo "  Frontend: $FRONTEND_IMAGE"
echo ""
echo "🌐 Access URLs:"
echo "  Frontend: http://localhost:8081"
echo "  Backend API: http://localhost:8080/api/tenants"
echo "  Backend Health: http://localhost:8080/api/health"
echo ""
echo "📊 Useful Commands:"
echo "  Check pods: kubectl get pods -n $NAMESPACE"
echo "  Check logs: kubectl logs -n $NAMESPACE -l app.kubernetes.io/name=mimir-insights"
echo "  Check services: kubectl get svc -n $NAMESPACE"
echo "  Stop port forwarding: pkill -f 'kubectl port-forward.*mimir-insights'"
echo ""
echo "🔧 Port Forwarding PIDs:"
echo "  Backend: $BACKEND_PF_PID"
echo "  Frontend: $FRONTEND_PF_PID"
echo ""
print_warning "Note: Port forwarding is running in background. Use 'pkill -f kubectl port-forward' to stop." 
#!/bin/bash

# Production Deployment Script for MimirInsights with AWS ALB
# Uses values-production-aws-alb.yaml configured for AWS Application Load Balancer

set -e

echo "🚀 Starting MimirInsights Production Deployment with AWS ALB..."

# Configuration
NAMESPACE="mimir-insights"
RELEASE_NAME="mimir-insights"
VALUES_FILE="deployments/helm-chart/values-production-aws-alb.yaml"
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

# Check if AWS Load Balancer Controller is installed
print_status "Checking AWS Load Balancer Controller..."
if ! kubectl get deployment -n kube-system aws-load-balancer-controller &> /dev/null; then
    print_warning "AWS Load Balancer Controller not found in kube-system namespace"
    print_warning "Please ensure AWS Load Balancer Controller is installed for ALB ingress to work"
    print_warning "Installation guide: https://kubernetes-sigs.github.io/aws-load-balancer-controller/latest/deploy/install/"
fi

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

# Check ingress
print_status "Checking ALB ingress..."
kubectl get ingress -n $NAMESPACE

# Wait for ALB to be provisioned
print_status "Waiting for AWS ALB to be provisioned..."
print_warning "This may take 2-5 minutes for AWS to provision the ALB..."

# Check ALB status
for i in {1..30}; do
    ALB_STATUS=$(kubectl get ingress -n $NAMESPACE -o jsonpath='{.items[0].status.loadBalancer.ingress[0].hostname}' 2>/dev/null || echo "")
    if [ -n "$ALB_STATUS" ]; then
        print_success "ALB provisioned: $ALB_STATUS"
        break
    fi
    print_status "Waiting for ALB... (attempt $i/30)"
    sleep 10
done

if [ -z "$ALB_STATUS" ]; then
    print_warning "ALB may still be provisioning. Check with: kubectl get ingress -n $NAMESPACE"
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
    if kubectl exec -n $NAMESPACE $FRONTEND_POD -- curl -f http://localhost/healthz 2>/dev/null; then
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
print_status "AWS ALB Information:"
if [ -n "$ALB_STATUS" ]; then
    echo "  ALB Hostname: $ALB_STATUS"
    echo "  Access URL: https://mimir-insights.yourdomain.com"
else
    echo "  ALB Status: Still provisioning (check with kubectl get ingress -n $NAMESPACE)"
fi
echo ""
print_status "Health Check Endpoints:"
echo "  Frontend Health: /healthz"
echo "  Backend Health: /api/health"
echo ""
print_status "To check logs:"
echo "  Backend: kubectl logs -f deployment/mimir-insights-backend -n $NAMESPACE"
echo "  Frontend: kubectl logs -f deployment/mimir-insights-frontend -n $NAMESPACE"
echo ""
print_status "To check ALB status:"
echo "  kubectl get ingress -n $NAMESPACE"
echo "  kubectl describe ingress -n $NAMESPACE"
echo ""
print_status "To uninstall:"
echo "  helm uninstall $RELEASE_NAME -n $NAMESPACE"
echo ""

print_success "🎉 MimirInsights is now deployed with AWS ALB and ready for production use!"
print_warning "Note: Update the domain name in the ingress configuration for your actual domain" 
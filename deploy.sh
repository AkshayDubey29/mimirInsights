#!/bin/bash

set -e

# Configuration
REGISTRY="ghcr.io/akshaydubey29"
BACKEND_IMAGE="mimir-insights-backend"
FRONTEND_IMAGE="mimir-insights-frontend"
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
CLUSTER_NAME="mimirInsights-test"
NAMESPACE="mimir-insights"
RELEASE_NAME="mimir-insights"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if docker is running
    if ! docker info > /dev/null 2>&1; then
        log_error "Docker is not running or not accessible"
        exit 1
    fi
    
    # Check if kind cluster exists
    if ! kind get clusters | grep -q "$CLUSTER_NAME"; then
        log_error "Kind cluster '$CLUSTER_NAME' not found"
        exit 1
    fi
    
    # Check if kubectl is available
    if ! command -v kubectl > /dev/null 2>&1; then
        log_error "kubectl is not installed"
        exit 1
    fi
    
    # Check if helm is available
    if ! command -v helm > /dev/null 2>&1; then
        log_error "helm is not installed"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Login to GitHub Container Registry
login_to_registry() {
    log_info "Logging in to GitHub Container Registry..."
    
    # Check if already logged in
    if docker info | grep -q "ghcr.io"; then
        log_info "Already logged in to ghcr.io"
        return
    fi
    
    # Try to login using GITHUB_TOKEN if available
    if [ -n "$GITHUB_TOKEN" ]; then
        echo "$GITHUB_TOKEN" | docker login ghcr.io -u akshaydubey29 --password-stdin
        log_success "Logged in to ghcr.io using GITHUB_TOKEN"
    else
        log_warning "GITHUB_TOKEN not set. Please login manually:"
        log_warning "docker login ghcr.io"
        read -p "Press Enter after logging in..."
    fi
}

# Build and tag images
build_images() {
    log_info "Building and tagging images..."
    
    # Build backend
    log_info "Building backend image..."
    docker build -f Dockerfile.backend -t "$REGISTRY/$BACKEND_IMAGE:$TIMESTAMP" .
    docker tag "$REGISTRY/$BACKEND_IMAGE:$TIMESTAMP" "$REGISTRY/$BACKEND_IMAGE:latest"
    
    # Build frontend
    log_info "Building frontend image..."
    docker build -f Dockerfile.frontend -t "$REGISTRY/$FRONTEND_IMAGE:$TIMESTAMP" .
    docker tag "$REGISTRY/$FRONTEND_IMAGE:$TIMESTAMP" "$REGISTRY/$FRONTEND_IMAGE:latest"
    
    log_success "Images built and tagged successfully"
}

# Push images to registry
push_images() {
    log_info "Pushing images to registry..."
    
    # Push backend
    log_info "Pushing backend image..."
    docker push "$REGISTRY/$BACKEND_IMAGE:$TIMESTAMP"
    docker push "$REGISTRY/$BACKEND_IMAGE:latest"
    
    # Push frontend
    log_info "Pushing frontend image..."
    docker push "$REGISTRY/$FRONTEND_IMAGE:$TIMESTAMP"
    docker push "$REGISTRY/$FRONTEND_IMAGE:latest"
    
    log_success "Images pushed successfully"
}

# Create values file with new image tags
create_values_file() {
    log_info "Creating values file with new image tags..."
    
    cat > values-production.yaml << EOF
# Mimir Insights Production Values
# Generated on: $(date)

global:
  imageRegistry: $REGISTRY
  imageTag: $TIMESTAMP

backend:
  image:
    repository: $BACKEND_IMAGE
    tag: $TIMESTAMP
    pullPolicy: Always
  replicaCount: 2
  resources:
    requests:
      memory: "256Mi"
      cpu: "250m"
    limits:
      memory: "512Mi"
      cpu: "500m"
  service:
    type: ClusterIP
    port: 8080

frontend:
  image:
    repository: $FRONTEND_IMAGE
    tag: $TIMESTAMP
    pullPolicy: Always
  replicaCount: 2
  resources:
    requests:
      memory: "128Mi"
      cpu: "100m"
    limits:
      memory: "256Mi"
      cpu: "200m"
  service:
    type: ClusterIP
    port: 80

ingress:
  enabled: true
  className: nginx
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
  hosts:
    - host: mimir-insights.local
      paths:
        - path: /
          pathType: Prefix
        - path: /api
          pathType: Prefix

persistence:
  enabled: false

monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s

config:
  mimir:
    namespace: mimir
    apiUrl: "http://mimir-distributor:9090"
    timeout: 30
  k8s:
    inCluster: true
    tenantLabel: "team"
    tenantPrefix: "tenant-"
  log:
    level: "info"
    format: "json"
  ui:
    theme: "dark"
    refreshInterval: 30
  llm:
    enabled: false
    provider: "openai"
    model: "gpt-4"
    maxTokens: 1000
EOF
    
    log_success "Values file created: values-production.yaml"
}

# Deploy to kind cluster
deploy_to_cluster() {
    log_info "Deploying to kind cluster '$CLUSTER_NAME'..."
    
    # Set kubectl context to kind cluster
    kubectl config use-context "kind-$CLUSTER_NAME"
    
    # Create namespace if it doesn't exist
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Add helm repository if needed
    helm repo add mimir-insights https://charts.bitnami.com/bitnami
    helm repo update
    
    # Install/upgrade the release
    log_info "Installing/upgrading helm release..."
    helm upgrade --install "$RELEASE_NAME" ./deployments/helm-chart \
        --namespace "$NAMESPACE" \
        --values values-production.yaml \
        --wait \
        --timeout 10m
    
    log_success "Deployment completed"
}

# Verify deployment
verify_deployment() {
    log_info "Verifying deployment..."
    
    # Wait for pods to be ready
    log_info "Waiting for pods to be ready..."
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mimir-insights-backend -n "$NAMESPACE" --timeout=300s
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mimir-insights-frontend -n "$NAMESPACE" --timeout=300s
    
    # Check pod status
    log_info "Pod status:"
    kubectl get pods -n "$NAMESPACE" -o wide
    
    # Check services
    log_info "Service status:"
    kubectl get svc -n "$NAMESPACE"
    
    # Check ingress
    log_info "Ingress status:"
    kubectl get ingress -n "$NAMESPACE"
    
    # Test backend health endpoint
    log_info "Testing backend health endpoint..."
    BACKEND_POD=$(kubectl get pod -l app.kubernetes.io/name=mimir-insights-backend -n "$NAMESPACE" -o jsonpath='{.items[0].metadata.name}')
    if kubectl exec "$BACKEND_POD" -n "$NAMESPACE" -- wget --no-verbose --tries=1 -O- http://localhost:8080/api/health; then
        log_success "Backend health check passed"
    else
        log_error "Backend health check failed"
        return 1
    fi
    
    log_success "Deployment verification completed"
}

# Show access information
show_access_info() {
    log_info "Deployment Summary:"
    echo "=================================="
    echo "Registry: $REGISTRY"
    echo "Backend Image: $REGISTRY/$BACKEND_IMAGE:$TIMESTAMP"
    echo "Frontend Image: $REGISTRY/$FRONTEND_IMAGE:$TIMESTAMP"
    echo "Cluster: $CLUSTER_NAME"
    echo "Namespace: $NAMESPACE"
    echo "Release: $RELEASE_NAME"
    echo ""
    echo "To access the application:"
    echo "1. Add to /etc/hosts: 127.0.0.1 mimir-insights.local"
    echo "2. Visit: http://mimir-insights.local"
    echo ""
    echo "To check logs:"
    echo "kubectl logs -f deployment/$RELEASE_NAME-backend -n $NAMESPACE"
    echo "kubectl logs -f deployment/$RELEASE_NAME-frontend -n $NAMESPACE"
    echo ""
    echo "To uninstall:"
    echo "helm uninstall $RELEASE_NAME -n $NAMESPACE"
}

# Main execution
main() {
    log_info "Starting Mimir Insights deployment..."
    log_info "Timestamp: $TIMESTAMP"
    
    check_prerequisites
    login_to_registry
    build_images
    push_images
    create_values_file
    deploy_to_cluster
    verify_deployment
    show_access_info
    
    log_success "Deployment completed successfully!"
}

# Run main function
main "$@" 
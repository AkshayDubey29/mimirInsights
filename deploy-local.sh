#!/bin/bash

set -e

# Configuration
BACKEND_IMAGE="mimir-insights-backend"
FRONTEND_IMAGE="mimir-insights-frontend"
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
CLUSTER_NAME="mimirinsights-test"
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

# Build and load images into kind cluster
build_and_load_images() {
    log_info "Building and loading images into kind cluster..."
    
    # Build backend
    log_info "Building backend image..."
    docker build -f Dockerfile.backend -t "$BACKEND_IMAGE:$TIMESTAMP" .
    docker tag "$BACKEND_IMAGE:$TIMESTAMP" "$BACKEND_IMAGE:latest"
    
    # Build frontend
    log_info "Building frontend image..."
    docker build -f Dockerfile.frontend -t "$FRONTEND_IMAGE:$TIMESTAMP" .
    docker tag "$FRONTEND_IMAGE:$TIMESTAMP" "$FRONTEND_IMAGE:latest"
    
    # Load images into kind cluster
    log_info "Loading images into kind cluster..."
    kind load docker-image "$BACKEND_IMAGE:$TIMESTAMP" --name "$CLUSTER_NAME"
    kind load docker-image "$FRONTEND_IMAGE:$TIMESTAMP" --name "$CLUSTER_NAME"
    
    log_success "Images built and loaded successfully"
}

# Create values file with local image tags
create_values_file() {
    log_info "Creating values file with local image tags..."
    
    cat > values-local.yaml << EOF
# Mimir Insights Local Values
# Generated on: $(date)

global:
  imageRegistry: ""
  imageTag: $TIMESTAMP

backend:
  image:
    repository: $BACKEND_IMAGE
    tag: $TIMESTAMP
    pullPolicy: Never
  replicaCount: 1
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
    pullPolicy: Never
  replicaCount: 1
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
  enabled: false

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
    
    log_success "Values file created: values-local.yaml"
}

# Deploy to kind cluster
deploy_to_cluster() {
    log_info "Deploying to kind cluster '$CLUSTER_NAME'..."
    
    # Set kubectl context to kind cluster
    kubectl config use-context "kind-$CLUSTER_NAME"
    
    # Create namespace if it doesn't exist
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Install/upgrade the release
    log_info "Installing/upgrading helm release..."
    helm upgrade --install "$RELEASE_NAME" ./deployments/helm-chart \
        --namespace "$NAMESPACE" \
        --values values-local.yaml \
        --wait \
        --timeout 10m
    
    log_success "Deployment completed"
}

# Verify deployment
verify_deployment() {
    log_info "Verifying deployment..."
    
    # Wait for pods to be ready
    log_info "Waiting for pods to be ready..."
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mimir-insights-backend -n "$NAMESPACE" --timeout=300s || true
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mimir-insights-frontend -n "$NAMESPACE" --timeout=300s || true
    
    # Check pod status
    log_info "Pod status:"
    kubectl get pods -n "$NAMESPACE" -o wide
    
    # Check services
    log_info "Service status:"
    kubectl get svc -n "$NAMESPACE"
    
    # Check ingress
    log_info "Ingress status:"
    kubectl get ingress -n "$NAMESPACE"
    
    log_success "Deployment verification completed"
}

# Show access information
show_access_info() {
    log_info "Deployment Summary:"
    echo "=================================="
    echo "Backend Image: $BACKEND_IMAGE:$TIMESTAMP"
    echo "Frontend Image: $FRONTEND_IMAGE:$TIMESTAMP"
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
    log_info "Starting Mimir Insights local deployment..."
    log_info "Timestamp: $TIMESTAMP"
    
    check_prerequisites
    build_and_load_images
    create_values_file
    deploy_to_cluster
    verify_deployment
    show_access_info
    
    log_success "Local deployment completed successfully!"
}

# Run main function
main "$@" 
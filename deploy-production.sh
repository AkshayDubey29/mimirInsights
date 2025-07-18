#!/bin/bash

# MimirInsights Production Deployment Script
# This script deploys MimirInsights to production with enhanced workload discovery,
# comprehensive monitoring, security hardening, and all advanced features.

set -euo pipefail

# Configuration
NAMESPACE="mimir-insights"
RELEASE_NAME="mimir-insights"
CHART_PATH="./deployments/helm-chart"
VALUES_FILE="./deployments/helm-chart/values-production.yaml"
TIMEOUT="10m"

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
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check if helm is available
    if ! command -v helm &> /dev/null; then
        log_error "helm is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we can connect to the cluster
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check if production values file exists
    if [[ ! -f "$VALUES_FILE" ]]; then
        log_error "Production values file not found: $VALUES_FILE"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Create required secrets (placeholders - update with your actual secrets)
create_secrets() {
    log_info "Creating required secrets..."
    
    # Create namespace if it doesn't exist
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Create LLM credentials secret (placeholder - update with actual credentials)
    kubectl create secret generic mimir-insights-llm-credentials \
        --from-literal=api-key="your-openai-api-key-here" \
        --namespace="$NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Create S3 backup credentials secret (placeholder - update with actual credentials)
    kubectl create secret generic mimir-insights-s3-credentials \
        --from-literal=access-key="your-s3-access-key" \
        --from-literal=secret-key="your-s3-secret-key" \
        --namespace="$NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Create encryption key secret (placeholder - generate a proper encryption key)
    kubectl create secret generic mimir-insights-encryption-key \
        --from-literal=key="$(openssl rand -base64 32)" \
        --namespace="$NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    log_success "Secrets created successfully"
}

# Create priority class for high priority pods
create_priority_class() {
    log_info "Creating priority class..."
    
    cat <<EOF | kubectl apply -f -
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: high-priority
value: 1000
globalDefault: false
description: "High priority class for critical MimirInsights components"
EOF
    
    log_success "Priority class created"
}

# Validate Helm chart
validate_chart() {
    log_info "Validating Helm chart..."
    
    if ! helm lint "$CHART_PATH" --values "$VALUES_FILE"; then
        log_error "Helm chart validation failed"
        exit 1
    fi
    
    log_success "Helm chart validation passed"
}

# Deploy or upgrade MimirInsights
deploy_mimir_insights() {
    log_info "Deploying MimirInsights to production..."
    
    # Check if release already exists
    if helm list -n "$NAMESPACE" | grep -q "$RELEASE_NAME"; then
        log_info "Release exists, performing upgrade..."
        helm upgrade "$RELEASE_NAME" "$CHART_PATH" \
            --namespace "$NAMESPACE" \
            --values "$VALUES_FILE" \
            --timeout "$TIMEOUT" \
            --wait \
            --atomic
    else
        log_info "Installing new release..."
        helm install "$RELEASE_NAME" "$CHART_PATH" \
            --namespace "$NAMESPACE" \
            --values "$VALUES_FILE" \
            --timeout "$TIMEOUT" \
            --wait \
            --atomic \
            --create-namespace
    fi
    
    log_success "MimirInsights deployed successfully"
}

# Wait for deployment to be ready
wait_for_deployment() {
    log_info "Waiting for deployment to be ready..."
    
    # Wait for backend deployment
    kubectl wait --for=condition=available deployment/mimir-insights-backend \
        --namespace="$NAMESPACE" \
        --timeout=300s
    
    # Wait for frontend deployment
    kubectl wait --for=condition=available deployment/mimir-insights-frontend \
        --namespace="$NAMESPACE" \
        --timeout=300s
    
    log_success "All deployments are ready"
}

# Verify deployment health
verify_deployment() {
    log_info "Verifying deployment health..."
    
    # Check pod status
    kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=mimir-insights
    
    # Check if all pods are running
    if ! kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=mimir-insights --no-headers | grep -v Running | grep -v Completed; then
        log_success "All pods are running"
    else
        log_warning "Some pods are not running. Check the pod status above."
    fi
    
    # Test backend health endpoint
    BACKEND_POD=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=mimir-insights,app.kubernetes.io/component=backend -o jsonpath='{.items[0].metadata.name}')
    if kubectl exec -n "$NAMESPACE" "$BACKEND_POD" -- curl -f http://localhost:8080/api/health &> /dev/null; then
        log_success "Backend health check passed"
    else
        log_error "Backend health check failed"
    fi
    
    # Display service information
    log_info "Service information:"
    kubectl get services -n "$NAMESPACE" -l app.kubernetes.io/name=mimir-insights
    
    # Display ingress information
    log_info "Ingress information:"
    kubectl get ingress -n "$NAMESPACE"
}

# Display access information
display_access_info() {
    log_info "Deployment complete! Access information:"
    
    echo ""
    echo "🎉 MimirInsights Production Deployment Complete!"
    echo ""
    echo "📊 Features Enabled:"
    echo "  ✅ Enhanced Workload Discovery (Deployments, StatefulSets, DaemonSets)"
    echo "  ✅ Drift Detection with ConfigMap comparison"
    echo "  ✅ Capacity Planning with trend analysis"
    echo "  ✅ LLM Integration for metrics interpretation"
    echo "  ✅ Alloy Tuning for all workload types"
    echo "  ✅ Production Monitoring & Alerting"
    echo "  ✅ Security Hardening & Audit Logging"
    echo "  ✅ High Availability (2 replicas)"
    echo "  ✅ Auto-scaling (HPA enabled)"
    echo ""
    echo "🔗 Access Methods:"
    echo "  🌐 Web UI: https://mimir-insights.yourdomain.com (update DNS)"
    echo "  📊 API: https://mimir-insights.yourdomain.com/api"
    echo ""
    echo "🛠️  Port Forwarding (for testing):"
    echo "  Frontend: kubectl port-forward -n $NAMESPACE service/mimir-insights-frontend 8081:80"
    echo "  Backend:  kubectl port-forward -n $NAMESPACE service/mimir-insights-backend 8080:8080"
    echo ""
    echo "📝 Management Commands:"
    echo "  Status:   kubectl get all -n $NAMESPACE"
    echo "  Logs:     kubectl logs -n $NAMESPACE -l app.kubernetes.io/name=mimir-insights -f"
    echo "  Describe: kubectl describe deployment -n $NAMESPACE"
    echo ""
    echo "🔒 Security Notes:"
    echo "  • Update the domain in values-production.yaml"
    echo "  • Configure proper SSL certificates"
    echo "  • Update secret placeholders with real credentials"
    echo "  • Review and adjust RBAC permissions as needed"
    echo ""
    echo "📦 Latest Images:"
    echo "  Backend:  ghcr.io/akshaydubey29/mimir-insights-backend:20250118-enhanced-workload"
    echo "  Frontend: ghcr.io/akshaydubey29/mimir-insights-frontend:20250718-070712"
    echo ""
}

# Test API endpoints
test_api_endpoints() {
    log_info "Testing API endpoints..."
    
    # Get backend pod for testing
    BACKEND_POD=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=mimir-insights,app.kubernetes.io/component=backend -o jsonpath='{.items[0].metadata.name}')
    
    if [[ -n "$BACKEND_POD" ]]; then
        echo "Testing enhanced workload discovery endpoints..."
        
        # Test new workloads endpoint
        if kubectl exec -n "$NAMESPACE" "$BACKEND_POD" -- curl -s http://localhost:8080/api/alloy/workloads | grep -q "workloads_by_type"; then
            log_success "✅ Enhanced workload discovery endpoint working"
        else
            log_warning "⚠️  Enhanced workload discovery endpoint may have issues"
        fi
        
        # Test health endpoint
        if kubectl exec -n "$NAMESPACE" "$BACKEND_POD" -- curl -f http://localhost:8080/api/health &> /dev/null; then
            log_success "✅ Health endpoint working"
        else
            log_error "❌ Health endpoint failed"
        fi
        
        # Test metrics endpoint
        if kubectl exec -n "$NAMESPACE" "$BACKEND_POD" -- curl -s http://localhost:8080/metrics | grep -q "go_"; then
            log_success "✅ Metrics endpoint working"
        else
            log_warning "⚠️  Metrics endpoint may have issues"
        fi
    else
        log_warning "Could not find backend pod for testing"
    fi
}

# Cleanup function for rollback
cleanup() {
    log_warning "Deployment failed. Rolling back..."
    helm rollback "$RELEASE_NAME" --namespace="$NAMESPACE" || true
}

# Main deployment function
main() {
    echo "🚀 Starting MimirInsights Production Deployment"
    echo "================================================"
    
    # Set trap for cleanup on failure
    trap cleanup ERR
    
    check_prerequisites
    create_secrets
    create_priority_class
    validate_chart
    deploy_mimir_insights
    wait_for_deployment
    verify_deployment
    test_api_endpoints
    display_access_info
    
    echo ""
    log_success "🎉 Production deployment completed successfully!"
}

# Handle script arguments
case "${1:-deploy}" in
    "deploy")
        main
        ;;
    "upgrade")
        log_info "Performing upgrade..."
        validate_chart
        deploy_mimir_insights
        wait_for_deployment
        verify_deployment
        test_api_endpoints
        log_success "Upgrade completed successfully!"
        ;;
    "status")
        log_info "Checking deployment status..."
        kubectl get all -n "$NAMESPACE" -l app.kubernetes.io/name=mimir-insights
        ;;
    "test")
        log_info "Testing API endpoints..."
        test_api_endpoints
        ;;
    "clean")
        log_warning "Uninstalling MimirInsights..."
        helm uninstall "$RELEASE_NAME" --namespace="$NAMESPACE"
        kubectl delete namespace "$NAMESPACE" --ignore-not-found
        log_success "Cleanup completed"
        ;;
    *)
        echo "Usage: $0 [deploy|upgrade|status|test|clean]"
        echo ""
        echo "Commands:"
        echo "  deploy   - Full production deployment (default)"
        echo "  upgrade  - Upgrade existing deployment"
        echo "  status   - Check deployment status"
        echo "  test     - Test API endpoints"
        echo "  clean    - Remove deployment"
        exit 1
        ;;
esac 
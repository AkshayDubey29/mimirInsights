#!/bin/bash

# MimirInsights Local Deployment Script
# This script updates the values file with the latest image tags and deploys to local kind cluster

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REGISTRY="ghcr.io"
REPO_NAME="akshaydubey29/mimir-insights"
NAMESPACE="mimir-insights"
VALUES_FILE="deployments/helm-chart/values-production-final.yaml"

# Get the latest timestamp tag from GitHub Container Registry
get_latest_timestamp() {
    echo -e "${BLUE}🔍 Getting latest image timestamp...${NC}"
    
    # Generate current timestamp in the format used by CI
    CURRENT_TIMESTAMP=$(date +'%Y%m%d-%H%M%S')
    
    # Try to get the latest timestamp tag from local images
    LATEST_TAG=$(docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.CreatedAt}}" | grep "${REPO_NAME}-frontend" | grep -E "[0-9]{8}-[0-9]{6}" | head -1 | awk '{print $2}')
    
    if [ -z "$LATEST_TAG" ]; then
        echo -e "${YELLOW}⚠️  No timestamp tags found locally, using current timestamp: ${CURRENT_TIMESTAMP}${NC}"
        LATEST_TAG="$CURRENT_TIMESTAMP"
    else
        echo -e "${GREEN}✅ Found latest timestamp: ${LATEST_TAG}${NC}"
    fi
    
    echo "$LATEST_TAG"
}

# Update values file with new image tags
update_values_file() {
    local timestamp=$1
    
    echo -e "${BLUE}📝 Updating values file with timestamp: ${timestamp}${NC}"
    
    # Create backup
    cp "$VALUES_FILE" "${VALUES_FILE}.backup.$(date +%Y%m%d-%H%M%S)"
    
    # Update frontend image
    sed -i.bak "s|repository:.*mimir-insights-frontend|repository: ${REPO_NAME}-frontend|g" "$VALUES_FILE"
    sed -i.bak "s|tag:.*\"[^\"]*\"|tag: \"${timestamp}\"|g" "$VALUES_FILE"
    
    # Update backend image
    sed -i.bak "s|repository:.*mimir-insights-backend|repository: ${REPO_NAME}-backend|g" "$VALUES_FILE"
    sed -i.bak "s|tag:.*\"[^\"]*\"|tag: \"${timestamp}\"|g" "$VALUES_FILE"
    
    # Clean up backup files
    rm -f "${VALUES_FILE}.bak"
    
    echo -e "${GREEN}✅ Values file updated successfully${NC}"
}

# Check prerequisites
check_prerequisites() {
    echo -e "${BLUE}🔍 Checking prerequisites...${NC}"
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}❌ kubectl is not installed${NC}"
        exit 1
    fi
    
    # Check if helm is available
    if ! command -v helm &> /dev/null; then
        echo -e "${RED}❌ helm is not installed${NC}"
        exit 1
    fi
    
    # Check if kind cluster is running
    if ! kubectl cluster-info &> /dev/null; then
        echo -e "${RED}❌ Kubernetes cluster is not accessible${NC}"
        echo -e "${YELLOW}💡 Make sure your kind cluster is running: kind start cluster${NC}"
        exit 1
    fi
    
    # Check if values file exists
    if [ ! -f "$VALUES_FILE" ]; then
        echo -e "${RED}❌ Values file not found: $VALUES_FILE${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ All prerequisites are satisfied${NC}"
}

# Create namespace
create_namespace() {
    echo -e "${BLUE}📦 Creating namespace...${NC}"
    
    if kubectl get namespace ${NAMESPACE} &> /dev/null; then
        echo -e "${YELLOW}⚠️  Namespace ${NAMESPACE} already exists${NC}"
    else
        kubectl create namespace ${NAMESPACE}
        echo -e "${GREEN}✅ Namespace ${NAMESPACE} created${NC}"
    fi
}

# Deploy using Helm
deploy_with_helm() {
    echo -e "${BLUE}🚀 Deploying with Helm...${NC}"
    
    # Check if helm chart exists
    if [ ! -d "./deployments/helm-chart" ]; then
        echo -e "${RED}❌ Helm chart not found in ./deployments/helm-chart${NC}"
        exit 1
    fi
    
    # Deploy using Helm
    echo -e "${YELLOW}Installing/upgrading MimirInsights...${NC}"
    helm upgrade --install mimir-insights ./deployments/helm-chart \
        --namespace ${NAMESPACE} \
        --values "$VALUES_FILE" \
        --wait \
        --timeout 10m
    
    echo -e "${GREEN}✅ Helm deployment completed${NC}"
}

# Verify deployment
verify_deployment() {
    echo -e "${BLUE}🔍 Verifying deployment...${NC}"
    
    # Wait for pods to be ready
    echo -e "${YELLOW}Waiting for pods to be ready...${NC}"
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mimir-insights --namespace ${NAMESPACE} --timeout=300s
    
    # Check pod status
    echo -e "${YELLOW}Checking pod status...${NC}"
    kubectl get pods -n ${NAMESPACE}
    
    # Check services
    echo -e "${YELLOW}Checking services...${NC}"
    kubectl get services -n ${NAMESPACE}
    
    echo -e "${GREEN}✅ Deployment verification completed${NC}"
}

# Setup port forwarding
setup_port_forwarding() {
    echo -e "${BLUE}🔗 Setting up port forwarding...${NC}"
    
    # Kill any existing port-forward processes
    pkill -f "kubectl port-forward" || true
    
    # Start port forwarding for backend
    echo -e "${YELLOW}Starting backend port forward (8080:8080)...${NC}"
    kubectl port-forward -n ${NAMESPACE} svc/mimir-insights-backend 8080:8080 &
    BACKEND_PF_PID=$!
    
    # Start port forwarding for frontend
    echo -e "${YELLOW}Starting frontend port forward (8081:80)...${NC}"
    kubectl port-forward -n ${NAMESPACE} svc/mimir-insights-frontend 8081:80 &
    FRONTEND_PF_PID=$!
    
    # Wait a moment for port forwarding to establish
    sleep 5
    
    echo -e "${GREEN}✅ Port forwarding setup completed${NC}"
    echo -e "${BLUE}📱 Access URLs:${NC}"
    echo -e "${YELLOW}  Frontend:${NC} http://localhost:8081"
    echo -e "${YELLOW}  Backend API:${NC} http://localhost:8080/api/tenants"
    echo ""
    echo -e "${YELLOW}💡 To stop port forwarding, run:${NC}"
    echo -e "${YELLOW}   kill ${BACKEND_PF_PID} ${FRONTEND_PF_PID}${NC}"
}

# Test the deployment
test_deployment() {
    echo -e "${BLUE}🧪 Testing deployment...${NC}"
    
    # Wait for services to be ready
    sleep 10
    
    # Test backend health
    echo -e "${YELLOW}Testing backend health...${NC}"
    if curl -s http://localhost:8080/health | grep -q "healthy"; then
        echo -e "${GREEN}✅ Backend health check passed${NC}"
    else
        echo -e "${RED}❌ Backend health check failed${NC}"
    fi
    
    # Test backend API
    echo -e "${YELLOW}Testing backend API...${NC}"
    if curl -s http://localhost:8080/api/tenants | jq -e '.discovery_info' > /dev/null; then
        echo -e "${GREEN}✅ Backend API test passed${NC}"
    else
        echo -e "${RED}❌ Backend API test failed${NC}"
    fi
    
    # Test frontend
    echo -e "${YELLOW}Testing frontend...${NC}"
    if curl -s http://localhost:8081/ | grep -q "MimirInsights"; then
        echo -e "${GREEN}✅ Frontend test passed${NC}"
    else
        echo -e "${RED}❌ Frontend test failed${NC}"
    fi
    
    echo -e "${GREEN}✅ All tests completed${NC}"
}

# Main execution
main() {
    echo -e "${BLUE}🎯 Starting MimirInsights local deployment...${NC}"
    echo ""
    
    check_prerequisites
    echo ""
    
    create_namespace
    echo ""
    
    # Get latest timestamp and update values file
    LATEST_TIMESTAMP=$(get_latest_timestamp)
    update_values_file "$LATEST_TIMESTAMP"
    echo ""
    
    deploy_with_helm
    echo ""
    
    verify_deployment
    echo ""
    
    setup_port_forwarding
    echo ""
    
    test_deployment
    echo ""
    
    echo -e "${GREEN}🎉 MimirInsights deployment completed successfully!${NC}"
    echo ""
    echo -e "${BLUE}📋 Summary:${NC}"
    echo -e "${YELLOW}  Image Tag:${NC} ${LATEST_TIMESTAMP}"
    echo -e "${YELLOW}  Namespace:${NC} ${NAMESPACE}"
    echo -e "${YELLOW}  Frontend:${NC} http://localhost:8081"
    echo -e "${YELLOW}  Backend API:${NC} http://localhost:8080/api/tenants"
    echo -e "${YELLOW}  Helm Release:${NC} mimir-insights"
    echo ""
    echo -e "${BLUE}🔧 Useful commands:${NC}"
    echo -e "${YELLOW}  View logs:${NC} kubectl logs -f -l app.kubernetes.io/name=mimir-insights -n ${NAMESPACE}"
    echo -e "${YELLOW}  View pods:${NC} kubectl get pods -n ${NAMESPACE}"
    echo -e "${YELLOW}  View services:${NC} kubectl get services -n ${NAMESPACE}"
    echo -e "${YELLOW}  Uninstall:${NC} helm uninstall mimir-insights -n ${NAMESPACE}"
}

# Run main function
main "$@" 
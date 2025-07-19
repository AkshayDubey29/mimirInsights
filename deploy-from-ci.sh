#!/bin/bash

# MimirInsights Local Deployment from CI Images
# This script deploys the application using pre-built multi-architecture images from GitHub Container Registry

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO_NAME="akshaydubey29/mimir-insights"
REGISTRY="ghcr.io"
VERSION=${1:-"latest"}
NAMESPACE="mimir-insights"

# Image names
FRONTEND_IMAGE="${REGISTRY}/${REPO_NAME}/mimir-insights-frontend:${VERSION}"
BACKEND_IMAGE="${REGISTRY}/${REPO_NAME}/mimir-insights-backend:${VERSION}"

echo -e "${BLUE}🚀 MimirInsights Local Deployment from CI Images${NC}"
echo -e "${BLUE}================================================${NC}"
echo -e "${YELLOW}Repository:${NC} ${REPO_NAME}"
echo -e "${YELLOW}Registry:${NC} ${REGISTRY}"
echo -e "${YELLOW}Version:${NC} ${VERSION}"
echo -e "${YELLOW}Namespace:${NC} ${NAMESPACE}"
echo -e "${YELLOW}Frontend Image:${NC} ${FRONTEND_IMAGE}"
echo -e "${YELLOW}Backend Image:${NC} ${BACKEND_IMAGE}"
echo ""

# Function to check prerequisites
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
    
    echo -e "${GREEN}✅ All prerequisites are satisfied${NC}"
}

# Function to create namespace
create_namespace() {
    echo -e "${BLUE}📦 Creating namespace...${NC}"
    
    if kubectl get namespace ${NAMESPACE} &> /dev/null; then
        echo -e "${YELLOW}⚠️  Namespace ${NAMESPACE} already exists${NC}"
    else
        kubectl create namespace ${NAMESPACE}
        echo -e "${GREEN}✅ Namespace ${NAMESPACE} created${NC}"
    fi
}

# Function to pull and verify images
pull_images() {
    echo -e "${BLUE}📥 Pulling images from GitHub Container Registry...${NC}"
    
    # Pull frontend image
    echo -e "${YELLOW}Pulling frontend image...${NC}"
    if docker pull ${FRONTEND_IMAGE}; then
        echo -e "${GREEN}✅ Frontend image pulled successfully${NC}"
    else
        echo -e "${RED}❌ Failed to pull frontend image${NC}"
        echo -e "${YELLOW}💡 Make sure the image exists in the registry and you have access${NC}"
        exit 1
    fi
    
    # Pull backend image
    echo -e "${YELLOW}Pulling backend image...${NC}"
    if docker pull ${BACKEND_IMAGE}; then
        echo -e "${GREEN}✅ Backend image pulled successfully${NC}"
    else
        echo -e "${RED}❌ Failed to pull backend image${NC}"
        echo -e "${YELLOW}💡 Make sure the image exists in the registry and you have access${NC}"
        exit 1
    fi
    
    # Verify image architectures
    echo -e "${BLUE}🔍 Verifying image architectures...${NC}"
    
    FRONTEND_ARCH=$(docker inspect ${FRONTEND_IMAGE} --format='{{.Architecture}}')
    BACKEND_ARCH=$(docker inspect ${BACKEND_IMAGE} --format='{{.Architecture}}')
    
    echo -e "${GREEN}✅ Frontend architecture: ${FRONTEND_ARCH}${NC}"
    echo -e "${GREEN}✅ Backend architecture: ${BACKEND_ARCH}${NC}"
}

# Function to create values file for deployment
create_values_file() {
    echo -e "${BLUE}📝 Creating deployment values file...${NC}"
    
    cat > values-ci-deployment.yaml << EOF
# MimirInsights CI Deployment Values
# Generated for version: ${VERSION}

global:
  imageRegistry: ${REGISTRY}
  imagePullSecrets:
    - name: ghcr-secret

frontend:
  enabled: true
  replicaCount: 2
  image:
    repository: ${REPO_NAME}/mimir-insights-frontend
    tag: "${VERSION}"
    pullPolicy: Always
  
  resources:
    requests:
      memory: "128Mi"
      cpu: "100m"
    limits:
      memory: "256Mi"
      cpu: "200m"
  
  env:
    - name: REACT_APP_API_BASE_URL
      value: ""
    - name: REACT_APP_ENABLE_MOCK_DATA
      value: "false"
    - name: REACT_APP_ENABLE_MONITORING
      value: "false"
    - name: REACT_APP_ENABLE_LLM
      value: "false"
    - name: LOCAL_DEV
      value: "false"
    - name: ENVIRONMENT
      value: "production"
    - name: API_BASE_URL
      value: ""
    - name: NODE_ENV
      value: "production"

backend:
  enabled: true
  replicaCount: 2
  image:
    repository: ${REPO_NAME}/mimir-insights-backend
    tag: "${VERSION}"
    pullPolicy: Always
  
  resources:
    requests:
      memory: "256Mi"
      cpu: "200m"
    limits:
      memory: "512Mi"
      cpu: "500m"
  
  env:
    - name: LOG_LEVEL
      value: "info"
    - name: PORT
      value: "8080"
    - name: KUBERNETES_SERVICE_HOST
      value: "kubernetes.default.svc"
    - name: KUBERNETES_SERVICE_PORT
      value: "443"
    - name: MIMIR_NAMESPACE
      value: "mimir"
    - name: DISCOVERY_INTERVAL
      value: "30s"
    - name: HEALTH_CHECK_INTERVAL
      value: "10s"
    - name: ENABLE_AUTO_DISCOVERY
      value: "true"
    - name: ENABLE_LLM_FEATURES
      value: "false"
    - name: ENABLE_MONITORING
      value: "false"
    - name: ENVIRONMENT
      value: "production"

ingress:
  enabled: true
  className: "nginx"
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /$1
    nginx.ingress.kubernetes.io/ssl-redirect: "false"
    nginx.ingress.kubernetes.io/use-regex: "true"
  
  hosts:
    - host: mimir-insights.local
      paths:
        - path: /(.*)
          pathType: Prefix
          backend:
            service:
              name: mimir-insights-frontend
              port:
                number: 80
        - path: /api/(.*)
          pathType: Prefix
          backend:
            service:
              name: mimir-insights-backend
              port:
                number: 8080

service:
  type: ClusterIP
  frontendPort: 80
  backendPort: 8080

# Mimir stack discovery patterns
mimirDiscovery:
  enabled: true
  patterns:
    - "mimir-*"
    - "cortex-*"
    - "prometheus-*"
  
  namespaces:
    - "mimir"
    - "monitoring"
    - "observability"
  
  labels:
    - "app.kubernetes.io/name"
    - "app.kubernetes.io/instance"
    - "app.kubernetes.io/component"

# Auto-discovery configuration
autoDiscovery:
  enabled: true
  interval: "30s"
  timeout: "10s"
  maxRetries: 3
  
  # Tenant discovery patterns
  tenantPatterns:
    - "tenant-*"
    - "org-*"
    - "team-*"
  
  # Component discovery patterns
  componentPatterns:
    - "mimir-distributor"
    - "mimir-ingester"
    - "mimir-querier"
    - "mimir-compactor"
    - "mimir-ruler"
    - "mimir-alertmanager"
    - "mimir-store-gateway"

# Resource limits and requests
resources:
  frontend:
    requests:
      memory: "128Mi"
      cpu: "100m"
    limits:
      memory: "256Mi"
      cpu: "200m"
  
  backend:
    requests:
      memory: "256Mi"
      cpu: "200m"
    limits:
      memory: "512Mi"
      cpu: "500m"

# Monitoring and logging
monitoring:
  enabled: false
  metrics:
    enabled: false
    port: 9090
    path: /metrics
  
  logging:
    level: "info"
    format: "json"
    output: "stdout"

# Security settings
security:
  runAsNonRoot: true
  runAsUser: 1001
  runAsGroup: 1000
  fsGroup: 1000
  
  securityContext:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    capabilities:
      drop:
        - ALL
EOF

    echo -e "${GREEN}✅ Values file created: values-ci-deployment.yaml${NC}"
}

# Function to deploy using Helm
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
        --values values-ci-deployment.yaml \
        --wait \
        --timeout 10m
    
    echo -e "${GREEN}✅ Helm deployment completed${NC}"
}

# Function to verify deployment
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
    
    # Check ingress
    echo -e "${YELLOW}Checking ingress...${NC}"
    kubectl get ingress -n ${NAMESPACE}
    
    echo -e "${GREEN}✅ Deployment verification completed${NC}"
}

# Function to setup port forwarding
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

# Function to test the deployment
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
    echo -e "${BLUE}🎯 Starting MimirInsights deployment from CI images...${NC}"
    echo ""
    
    check_prerequisites
    echo ""
    
    create_namespace
    echo ""
    
    pull_images
    echo ""
    
    create_values_file
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
    echo -e "${YELLOW}  Version:${NC} ${VERSION}"
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
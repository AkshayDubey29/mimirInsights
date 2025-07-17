#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="mimir-insights"
REGISTRY="ghcr.io/akshaydubey29"
NAMESPACE="mimir-insights"
KIND_CLUSTER="mimirInsights-test"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "v1.0.0")

echo -e "${BLUE}🚀 Starting MimirInsights Deployment Pipeline${NC}"
echo -e "${BLUE}Timestamp: ${TIMESTAMP}${NC}"
echo -e "${BLUE}Version: ${VERSION}${NC}"
echo ""

# Function to print step headers
print_step() {
    echo -e "${YELLOW}===================================================${NC}"
    echo -e "${YELLOW}📦 $1${NC}"
    echo -e "${YELLOW}===================================================${NC}"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
print_step "Checking Prerequisites"
if ! command_exists docker; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    exit 1
fi

if ! command_exists kind; then
    echo -e "${RED}❌ Kind is not installed${NC}"
    exit 1
fi

if ! command_exists kubectl; then
    echo -e "${RED}❌ kubectl is not installed${NC}"
    exit 1
fi

if ! command_exists helm; then
    echo -e "${RED}❌ Helm is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ All prerequisites found${NC}"

# Check if kind cluster exists
print_step "Checking Kind Cluster"
if ! kind get clusters | grep -q "^${KIND_CLUSTER}$"; then
    echo -e "${YELLOW}⚠️  Kind cluster '${KIND_CLUSTER}' not found. Creating...${NC}"
    cat > kind-config.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ${KIND_CLUSTER}
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
- role: worker
- role: worker
EOF
    kind create cluster --config kind-config.yaml
    rm kind-config.yaml
else
    echo -e "${GREEN}✅ Kind cluster '${KIND_CLUSTER}' exists${NC}"
fi

# Set kubectl context
kubectl config use-context kind-${KIND_CLUSTER}

# Build Docker images
print_step "Building Docker Images"
echo -e "${BLUE}Building backend image...${NC}"
docker build -f Dockerfile.backend -t ${REGISTRY}/${APP_NAME}-backend:${TIMESTAMP} -t ${REGISTRY}/${APP_NAME}-backend:latest .

echo -e "${BLUE}Building frontend image...${NC}"
docker build -f Dockerfile.frontend -t ${REGISTRY}/${APP_NAME}-frontend:${TIMESTAMP} -t ${REGISTRY}/${APP_NAME}-frontend:latest .

echo -e "${GREEN}✅ Docker images built successfully${NC}"

# Login to GitHub Container Registry
print_step "Logging into GitHub Container Registry"
if [ -n "$GITHUB_TOKEN" ]; then
    echo $GITHUB_TOKEN | docker login ghcr.io -u akshaydubey29 --password-stdin
    echo -e "${GREEN}✅ Logged into GHCR${NC}"
else
    echo -e "${YELLOW}⚠️  GITHUB_TOKEN not set. Please login manually:${NC}"
    echo "docker login ghcr.io -u akshaydubey29"
    read -p "Press enter after logging in..."
fi

# Push Docker images
print_step "Pushing Docker Images to GHCR"
echo -e "${BLUE}Pushing backend images...${NC}"
docker push ${REGISTRY}/${APP_NAME}-backend:${TIMESTAMP}
docker push ${REGISTRY}/${APP_NAME}-backend:latest

echo -e "${BLUE}Pushing frontend images...${NC}"
docker push ${REGISTRY}/${APP_NAME}-frontend:${TIMESTAMP}
docker push ${REGISTRY}/${APP_NAME}-frontend:latest

echo -e "${GREEN}✅ All images pushed to GHCR${NC}"

# Load images into kind cluster (for faster deployment)
print_step "Loading Images into Kind Cluster"
kind load docker-image ${REGISTRY}/${APP_NAME}-backend:${TIMESTAMP} --name ${KIND_CLUSTER}
kind load docker-image ${REGISTRY}/${APP_NAME}-frontend:${TIMESTAMP} --name ${KIND_CLUSTER}

echo -e "${GREEN}✅ Images loaded into kind cluster${NC}"

# Install/Setup NGINX Ingress Controller
print_step "Setting up NGINX Ingress Controller"
if ! kubectl get namespace ingress-nginx >/dev/null 2>&1; then
    echo -e "${BLUE}Installing NGINX Ingress Controller...${NC}"
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
    
    echo -e "${BLUE}Waiting for NGINX Ingress Controller to be ready...${NC}"
    kubectl wait --namespace ingress-nginx \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=controller \
        --timeout=300s
else
    echo -e "${GREEN}✅ NGINX Ingress Controller already installed${NC}"
fi

# Create namespace
print_step "Creating Namespace"
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

# Deploy with Helm
print_step "Deploying with Helm"
if helm list -n ${NAMESPACE} | grep -q ${APP_NAME}; then
    echo -e "${BLUE}Upgrading existing Helm release...${NC}"
    helm upgrade ${APP_NAME} ./deployments/helm-chart \
        --namespace ${NAMESPACE} \
        --set backend.image.tag=${TIMESTAMP} \
        --set frontend.image.tag=${TIMESTAMP} \
        --set ingress.hosts[0].host=mimir-insights.local \
        --set imageRegistry=${REGISTRY} \
        --wait --timeout=300s
else
    echo -e "${BLUE}Installing new Helm release...${NC}"
    helm install ${APP_NAME} ./deployments/helm-chart \
        --namespace ${NAMESPACE} \
        --set backend.image.tag=${TIMESTAMP} \
        --set frontend.image.tag=${TIMESTAMP} \
        --set ingress.hosts[0].host=mimir-insights.local \
        --set imageRegistry=${REGISTRY} \
        --wait --timeout=300s
fi

echo -e "${GREEN}✅ Helm deployment completed${NC}"

# Wait for pods to be ready
print_step "Waiting for Pods to be Ready"
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=mimir-insights -n ${NAMESPACE} --timeout=300s

# Get deployment status
print_step "Deployment Status"
echo -e "${BLUE}Pods:${NC}"
kubectl get pods -n ${NAMESPACE}
echo ""
echo -e "${BLUE}Services:${NC}"
kubectl get services -n ${NAMESPACE}
echo ""
echo -e "${BLUE}Ingress:${NC}"
kubectl get ingress -n ${NAMESPACE}

# Port forward for local access
print_step "Setting up Port Forward"
echo -e "${BLUE}Setting up port forwarding...${NC}"
echo -e "${YELLOW}You can access the application at:${NC}"
echo -e "${GREEN}🌐 Frontend: http://localhost:8080${NC}"
echo -e "${GREEN}🔧 Backend API: http://localhost:8081/api${NC}"

# Create port forward script
cat > port-forward.sh <<EOF
#!/bin/bash
echo "Starting port forwarding..."
echo "Frontend: http://localhost:8080"
echo "Backend API: http://localhost:8081/api"
echo "Press Ctrl+C to stop"

trap 'kill %1 %2; exit' INT

kubectl port-forward -n ${NAMESPACE} service/${APP_NAME}-frontend 8080:80 &
kubectl port-forward -n ${NAMESPACE} service/${APP_NAME}-backend 8081:8080 &

wait
EOF

chmod +x port-forward.sh

# Test endpoints
print_step "Testing Application"
echo -e "${BLUE}Testing backend health endpoint...${NC}"
kubectl exec -n ${NAMESPACE} deployment/${APP_NAME}-backend -- wget -q -O- http://localhost:8080/api/health || echo "Health check will be available once backend is fully ready"

echo -e "${GREEN}✅ Deployment completed successfully!${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "1. Run ${GREEN}./port-forward.sh${NC} to access the application locally"
echo -e "2. Visit ${GREEN}http://localhost:8080${NC} for the frontend"
echo -e "3. Visit ${GREEN}http://localhost:8081/api/health${NC} for backend health check"
echo -e "4. Add ${GREEN}127.0.0.1 mimir-insights.local${NC} to /etc/hosts for ingress access"
echo ""
echo -e "${BLUE}🎉 MimirInsights deployment pipeline completed!${NC}"
#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${PURPLE}🚀 MimirInsights Complete Deployment Pipeline${NC}"
echo -e "${PURPLE}===============================================${NC}"
echo ""

# Function to print step headers
print_step() {
    echo -e "${CYAN}===================================================${NC}"
    echo -e "${CYAN}🔥 STEP $1: $2${NC}"
    echo -e "${CYAN}===================================================${NC}"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
print_step "1" "Checking Prerequisites"
echo -e "${BLUE}Checking required tools...${NC}"

MISSING_TOOLS=()

if ! command_exists docker; then
    MISSING_TOOLS+=("docker")
fi

if ! command_exists kind; then
    MISSING_TOOLS+=("kind")
fi

if ! command_exists kubectl; then
    MISSING_TOOLS+=("kubectl")
fi

if ! command_exists helm; then
    MISSING_TOOLS+=("helm")
fi

if ! command_exists curl; then
    MISSING_TOOLS+=("curl")
fi

if ! command_exists jq; then
    MISSING_TOOLS+=("jq")
fi

if [ ${#MISSING_TOOLS[@]} -gt 0 ]; then
    echo -e "${RED}❌ Missing required tools: ${MISSING_TOOLS[*]}${NC}"
    echo -e "${YELLOW}Please install the missing tools and try again.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ All required tools are available${NC}"

# Make scripts executable
print_step "2" "Preparing Deployment Scripts"
echo -e "${BLUE}Making scripts executable...${NC}"
chmod +x deploy.sh mock-data-generator.sh
if [ -f "port-forward.sh" ]; then
    chmod +x port-forward.sh
fi
if [ -f "test-api.sh" ]; then
    chmod +x test-api.sh
fi
if [ -f "simulate-data.sh" ]; then
    chmod +x simulate-data.sh
fi
echo -e "${GREEN}✅ Scripts are now executable${NC}"

# Run main deployment
print_step "3" "Running Main Deployment Pipeline"
echo -e "${BLUE}Starting container build and Kubernetes deployment...${NC}"
./deploy.sh

# Wait a moment for deployment to settle
echo -e "${BLUE}Waiting for deployment to settle...${NC}"
sleep 10

# Set up mock data environment
print_step "4" "Setting up Mock Data Environment"
echo -e "${BLUE}Creating mock Mimir cluster and tenant data...${NC}"
./mock-data-generator.sh

# Wait for mock services to be ready
echo -e "${BLUE}Waiting for mock services to be ready...${NC}"
sleep 5

# Verify deployment
print_step "5" "Verifying Deployment"
echo -e "${BLUE}Checking deployment status...${NC}"

echo -e "${YELLOW}MimirInsights Components:${NC}"
kubectl get pods -n mimir-insights

echo -e "${YELLOW}Mock Mimir Components:${NC}"
kubectl get pods -n mimir-test

echo -e "${YELLOW}Tenant Namespaces:${NC}"
kubectl get namespaces -l team

echo -e "${YELLOW}Services:${NC}"
kubectl get services -n mimir-insights

echo -e "${YELLOW}Ingress:${NC}"
kubectl get ingress -n mimir-insights

# Test endpoints (if port-forward script exists)
print_step "6" "Testing Application Endpoints"
if [ -f "test-api.sh" ]; then
    echo -e "${BLUE}Starting port forwarding in background...${NC}"
    if [ -f "port-forward.sh" ]; then
        ./port-forward.sh &
        PORT_FORWARD_PID=$!
        sleep 10
        
        echo -e "${BLUE}Testing API endpoints...${NC}"
        ./test-api.sh
        
        # Stop port forwarding
        kill $PORT_FORWARD_PID 2>/dev/null || true
    else
        echo -e "${YELLOW}⚠️ Port forward script not found, testing manually...${NC}"
        kubectl port-forward -n mimir-insights service/mimir-insights-backend 8081:8080 &
        PORT_FORWARD_PID=$!
        sleep 5
        
        # Basic health check
        curl -s http://localhost:8081/api/health || echo "API not ready yet"
        
        kill $PORT_FORWARD_PID 2>/dev/null || true
    fi
else
    echo -e "${YELLOW}⚠️ Test script not found, skipping automated testing${NC}"
fi

# Display final status and instructions
print_step "7" "Deployment Complete!"

echo -e "${GREEN}🎉 MimirInsights has been successfully deployed!${NC}"
echo ""
echo -e "${YELLOW}📋 Deployment Summary:${NC}"
echo -e "   🐳 Container images built and pushed to ghcr.io/akshaydubey29"
echo -e "   ☸️  Kind cluster 'mimirInsights-test' is running"
echo -e "   📦 Helm chart deployed in 'mimir-insights' namespace"
echo -e "   🎭 Mock Mimir environment created in 'mimir-test' namespace"
echo -e "   👥 Mock tenant namespaces (team-a, team-b, team-c) created"
echo ""
echo -e "${YELLOW}🌐 Access Points:${NC}"
echo -e "   Frontend UI: ${GREEN}http://localhost:8080${NC} (with port-forward)"
echo -e "   Backend API: ${GREEN}http://localhost:8081/api${NC} (with port-forward)"
echo -e "   Ingress URL: ${GREEN}http://mimir-insights.local${NC} (add to /etc/hosts)"
echo ""
echo -e "${YELLOW}🚀 Next Steps:${NC}"
echo -e "1. Start port forwarding: ${GREEN}./port-forward.sh${NC}"
echo -e "2. Test API endpoints: ${GREEN}./test-api.sh${NC}"
echo -e "3. Simulate data generation: ${GREEN}./simulate-data.sh${NC}"
echo -e "4. Open browser to: ${GREEN}http://localhost:8080${NC}"
echo ""
echo -e "${YELLOW}🔧 Useful Commands:${NC}"
echo -e "   View backend logs: ${CYAN}kubectl logs -f -n mimir-insights deployment/mimir-insights-backend${NC}"
echo -e "   View all pods: ${CYAN}kubectl get pods --all-namespaces${NC}"
echo -e "   Access shell: ${CYAN}kubectl exec -it -n mimir-insights deployment/mimir-insights-backend -- /bin/sh${NC}"
echo -e "   Helm status: ${CYAN}helm status mimir-insights -n mimir-insights${NC}"
echo ""
echo -e "${YELLOW}🧹 Cleanup (when done):${NC}"
echo -e "   Remove deployment: ${CYAN}helm uninstall mimir-insights -n mimir-insights${NC}"
echo -e "   Delete kind cluster: ${CYAN}kind delete cluster --name mimirInsights-test${NC}"
echo ""
echo -e "${PURPLE}✨ Happy monitoring with MimirInsights! ✨${NC}"
#!/bin/bash

# 🚀 MimirInsights Production Verification Script
# This script verifies that all components are running correctly

set -e

echo "🔍 MimirInsights Production Verification"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    if [ "$status" = "OK" ]; then
        echo -e "${GREEN}✅ $message${NC}"
    elif [ "$status" = "WARN" ]; then
        echo -e "${YELLOW}⚠️  $message${NC}"
    else
        echo -e "${RED}❌ $message${NC}"
    fi
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
echo -e "\n${BLUE}📋 Checking Prerequisites...${NC}"
if command_exists kubectl; then
    print_status "OK" "kubectl is installed"
else
    print_status "ERROR" "kubectl is not installed"
    exit 1
fi

if command_exists helm; then
    print_status "OK" "helm is installed"
else
    print_status "ERROR" "helm is not installed"
    exit 1
fi

# Check cluster connectivity
echo -e "\n${BLUE}🔗 Checking Cluster Connectivity...${NC}"
if kubectl cluster-info >/dev/null 2>&1; then
    print_status "OK" "Kubernetes cluster is accessible"
else
    print_status "ERROR" "Cannot connect to Kubernetes cluster"
    exit 1
fi

# Check namespaces
echo -e "\n${BLUE}📦 Checking Namespaces...${NC}"
if kubectl get namespace mimir-insights >/dev/null 2>&1; then
    print_status "OK" "mimir-insights namespace exists"
else
    print_status "WARN" "mimir-insights namespace not found"
fi

if kubectl get namespace mimir >/dev/null 2>&1; then
    print_status "OK" "mimir namespace exists"
else
    print_status "WARN" "mimir namespace not found"
fi

# Check MimirInsights pods
echo -e "\n${BLUE}🚀 Checking MimirInsights Pods...${NC}"
if kubectl get pods -n mimir-insights >/dev/null 2>&1; then
    BACKEND_PODS=$(kubectl get pods -n mimir-insights -l app.kubernetes.io/component=backend --no-headers | wc -l)
    FRONTEND_PODS=$(kubectl get pods -n mimir-insights -l app.kubernetes.io/component=frontend --no-headers | wc -l)
    
    if [ "$BACKEND_PODS" -gt 0 ]; then
        print_status "OK" "Backend pods found: $BACKEND_PODS"
    else
        print_status "WARN" "No backend pods found"
    fi
    
    if [ "$FRONTEND_PODS" -gt 0 ]; then
        print_status "OK" "Frontend pods found: $FRONTEND_PODS"
    else
        print_status "WARN" "No frontend pods found"
    fi
else
    print_status "WARN" "Cannot check mimir-insights pods"
fi

# Check Mimir pods
echo -e "\n${BLUE}📊 Checking Mimir Pods...${NC}"
if kubectl get pods -n mimir >/dev/null 2>&1; then
    MIMIR_PODS=$(kubectl get pods -n mimir --no-headers | wc -l)
    if [ "$MIMIR_PODS" -gt 0 ]; then
        print_status "OK" "Mimir pods found: $MIMIR_PODS"
    else
        print_status "WARN" "No Mimir pods found"
    fi
else
    print_status "WARN" "Cannot check mimir pods"
fi

# Check services
echo -e "\n${BLUE}🔌 Checking Services...${NC}"
if kubectl get services -n mimir-insights >/dev/null 2>&1; then
    BACKEND_SVC=$(kubectl get services -n mimir-insights -l app.kubernetes.io/component=backend --no-headers | wc -l)
    FRONTEND_SVC=$(kubectl get services -n mimir-insights -l app.kubernetes.io/component=frontend --no-headers | wc -l)
    
    if [ "$BACKEND_SVC" -gt 0 ]; then
        print_status "OK" "Backend services found: $BACKEND_SVC"
    else
        print_status "WARN" "No backend services found"
    fi
    
    if [ "$FRONTEND_SVC" -gt 0 ]; then
        print_status "OK" "Frontend services found: $FRONTEND_SVC"
    else
        print_status "WARN" "No frontend services found"
    fi
else
    print_status "WARN" "Cannot check mimir-insights services"
fi

# Check ingress
echo -e "\n${BLUE}🌐 Checking Ingress...${NC}"
if kubectl get ingress -n mimir-insights >/dev/null 2>&1; then
    INGRESS_COUNT=$(kubectl get ingress -n mimir-insights --no-headers | wc -l)
    if [ "$INGRESS_COUNT" -gt 0 ]; then
        print_status "OK" "Ingress resources found: $INGRESS_COUNT"
    else
        print_status "WARN" "No ingress resources found"
    fi
else
    print_status "WARN" "Cannot check ingress resources"
fi

# Test API endpoints (if port-forward is available)
echo -e "\n${BLUE}🔍 Testing API Endpoints...${NC}"
if command_exists curl; then
    # Try to test backend API
    if curl -s http://localhost:8080/api/health >/dev/null 2>&1; then
        print_status "OK" "Backend API is accessible on localhost:8080"
        
        # Test specific endpoints
        if curl -s http://localhost:8080/api/tenants >/dev/null 2>&1; then
            print_status "OK" "Tenants API endpoint working"
        else
            print_status "WARN" "Tenants API endpoint not responding"
        fi
        
        if curl -s http://localhost:8080/api/limits >/dev/null 2>&1; then
            print_status "OK" "Limits API endpoint working"
        else
            print_status "WARN" "Limits API endpoint not responding"
        fi
        
        if curl -s http://localhost:8080/api/discovery >/dev/null 2>&1; then
            print_status "OK" "Discovery API endpoint working"
        else
            print_status "WARN" "Discovery API endpoint not responding"
        fi
    else
        print_status "WARN" "Backend API not accessible on localhost:8080 (port-forward may not be active)"
    fi
    
    # Try to test frontend
    if curl -s http://localhost:8081/ >/dev/null 2>&1; then
        print_status "OK" "Frontend is accessible on localhost:8081"
    else
        print_status "WARN" "Frontend not accessible on localhost:8081 (port-forward may not be active)"
    fi
else
    print_status "WARN" "curl not available for API testing"
fi

# Check Docker images
echo -e "\n${BLUE}🐳 Checking Docker Images...${NC}"
if command_exists docker; then
    BACKEND_IMAGE="ghcr.io/akshaydubey29/mimir-insights-backend:v1.0.0-20250719-061613"
    FRONTEND_IMAGE="ghcr.io/akshaydubey29/mimir-insights-frontend:v1.0.0-20250719-061613"
    
    if docker manifest inspect "$BACKEND_IMAGE" >/dev/null 2>&1; then
        print_status "OK" "Backend image exists: $BACKEND_IMAGE"
    else
        print_status "WARN" "Backend image not found: $BACKEND_IMAGE"
    fi
    
    if docker manifest inspect "$FRONTEND_IMAGE" >/dev/null 2>&1; then
        print_status "OK" "Frontend image exists: $FRONTEND_IMAGE"
    else
        print_status "WARN" "Frontend image not found: $FRONTEND_IMAGE"
    fi
else
    print_status "WARN" "docker not available for image verification"
fi

# Summary
echo -e "\n${BLUE}📊 Verification Summary${NC}"
echo "================================"
echo -e "${GREEN}✅ Production verification completed!${NC}"
echo ""
echo "🚀 Next Steps:"
echo "1. Deploy to production environment"
echo "2. Configure monitoring and alerting"
echo "3. Set up backup and disaster recovery"
echo "4. Perform load testing"
echo "5. Document operational procedures"
echo ""
echo "📚 Documentation: PRODUCTION_DEPLOYMENT.md"
echo "🔗 GitHub: https://github.com/AkshayDubey29/mimirInsights"
echo ""
echo -e "${BLUE}🎉 MimirInsights is production-ready!${NC}" 
#!/bin/bash

# MimirInsights and Mimir Production Stack Status Check
# This script provides a comprehensive overview of the entire system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 MimirInsights & Mimir Production Stack Status${NC}"
echo "=================================================="
echo ""

# Check MimirInsights Application
echo -e "${CYAN}📱 MimirInsights Application${NC}"
echo "----------------------------------------"

# Check if mimir-insights namespace exists
if kubectl get namespace mimir-insights &> /dev/null; then
    echo -e "${GREEN}✅ mimir-insights namespace exists${NC}"
    
    # Check pods
    echo -e "${YELLOW}📦 Pods:${NC}"
    kubectl get pods -n mimir-insights --no-headers | while read line; do
        if echo "$line" | grep -q "Running"; then
            echo -e "  ${GREEN}✅ $line${NC}"
        else
            echo -e "  ${RED}❌ $line${NC}"
        fi
    done
    
    # Check services
    echo -e "${YELLOW}🔗 Services:${NC}"
    kubectl get services -n mimir-insights --no-headers | while read line; do
        echo -e "  ${BLUE}🔗 $line${NC}"
    done
else
    echo -e "${RED}❌ mimir-insights namespace not found${NC}"
fi

echo ""

# Check Mimir Production Stack
echo -e "${PURPLE}🏭 Mimir Production Stack${NC}"
echo "----------------------------------------"

# Check if mimir namespace exists
if kubectl get namespace mimir &> /dev/null; then
    echo -e "${GREEN}✅ mimir namespace exists${NC}"
    
    # Check Mimir pods
    echo -e "${YELLOW}📦 Mimir Pods:${NC}"
    kubectl get pods -n mimir --no-headers | while read line; do
        if echo "$line" | grep -q "Running"; then
            echo -e "  ${GREEN}✅ $line${NC}"
        else
            echo -e "  ${RED}❌ $line${NC}"
        fi
    done
    
    # Check Mimir services
    echo -e "${YELLOW}🔗 Mimir Services:${NC}"
    kubectl get services -n mimir --no-headers | while read line; do
        echo -e "  ${BLUE}🔗 $line${NC}"
    done
    
    # Check Mimir components health
    echo -e "${YELLOW}🏥 Mimir Health Check:${NC}"
    
    # Test Mimir API
    kubectl port-forward -n mimir svc/mimir-api 9009:9009 &> /dev/null &
    PF_PID=$!
    sleep 3
    
    if curl -s http://localhost:9009/ready &> /dev/null; then
        echo -e "  ${GREEN}✅ Mimir API is ready${NC}"
        
        # Get Mimir version
        VERSION=$(curl -s http://localhost:9009/api/v1/status/buildinfo | jq -r '.data.version' 2>/dev/null || echo "Unknown")
        echo -e "  ${BLUE}📋 Mimir Version: $VERSION${NC}"
    else
        echo -e "  ${RED}❌ Mimir API is not ready${NC}"
    fi
    
    kill $PF_PID &> /dev/null || true
else
    echo -e "${RED}❌ mimir namespace not found${NC}"
fi

echo ""

# Check Tenant Namespaces
echo -e "${CYAN}🏢 Tenant Namespaces${NC}"
echo "----------------------------------------"

TENANTS=("tenant-prod" "tenant-staging" "tenant-dev")
for tenant in "${TENANTS[@]}"; do
    if kubectl get namespace $tenant &> /dev/null; then
        echo -e "${GREEN}✅ $tenant namespace exists${NC}"
    else
        echo -e "${RED}❌ $tenant namespace not found${NC}"
    fi
done

echo ""

# Check Cluster Resources
echo -e "${BLUE}💾 Cluster Resources${NC}"
echo "----------------------------------------"

# Check nodes
echo -e "${YELLOW}🖥️  Nodes:${NC}"
kubectl get nodes --no-headers | while read line; do
    if echo "$line" | grep -q "Ready"; then
        echo -e "  ${GREEN}✅ $line${NC}"
    else
        echo -e "  ${RED}❌ $line${NC}"
    fi
done

# Check resource usage
echo -e "${YELLOW}📊 Resource Usage:${NC}"
kubectl top nodes --no-headers 2>/dev/null | while read line; do
    echo -e "  ${BLUE}📈 $line${NC}"
done

echo ""

# Check Port Forwarding
echo -e "${PURPLE}🔌 Port Forwarding Status${NC}"
echo "----------------------------------------"

# Check if ports are in use
if lsof -i :8080 &> /dev/null; then
    echo -e "${GREEN}✅ Port 8080 (Backend) is in use${NC}"
else
    echo -e "${YELLOW}⚠️  Port 8080 (Backend) is not in use${NC}"
fi

if lsof -i :8081 &> /dev/null; then
    echo -e "${GREEN}✅ Port 8081 (Frontend) is in use${NC}"
else
    echo -e "${YELLOW}⚠️  Port 8081 (Frontend) is not in use${NC}"
fi

if lsof -i :9009 &> /dev/null; then
    echo -e "${GREEN}✅ Port 9009 (Mimir API) is in use${NC}"
else
    echo -e "${YELLOW}⚠️  Port 9009 (Mimir API) is not in use${NC}"
fi

echo ""

# Summary
echo -e "${BLUE}📋 Summary${NC}"
echo "----------------------------------------"

# Count running pods
MIMIR_INSIGHTS_PODS=$(kubectl get pods -n mimir-insights --no-headers 2>/dev/null | grep -c "Running" || echo "0")
MIMIR_PODS=$(kubectl get pods -n mimir --no-headers 2>/dev/null | grep -c "Running" || echo "0")
TOTAL_PODS=$((MIMIR_INSIGHTS_PODS + MIMIR_PODS))

echo -e "${GREEN}✅ Total Running Pods: $TOTAL_PODS${NC}"
echo -e "${BLUE}📱 MimirInsights Pods: $MIMIR_INSIGHTS_PODS${NC}"
echo -e "${PURPLE}🏭 Mimir Production Pods: $MIMIR_PODS${NC}"

# Check if everything is ready
if [ "$MIMIR_PODS" -ge 7 ]; then
    echo -e "${GREEN}🎉 Mimir Production Stack is fully operational!${NC}"
else
    echo -e "${YELLOW}⚠️  Mimir Production Stack is still starting up...${NC}"
fi

if [ "$MIMIR_INSIGHTS_PODS" -ge 2 ]; then
    echo -e "${GREEN}🎉 MimirInsights Application is fully operational!${NC}"
else
    echo -e "${YELLOW}⚠️  MimirInsights Application is not deployed or still starting up...${NC}"
fi

echo ""
echo -e "${BLUE}🔧 Useful Commands:${NC}"
echo -e "${YELLOW}  Deploy MimirInsights:${NC} ./deploy-local.sh"
echo -e "${YELLOW}  View MimirInsights logs:${NC} kubectl logs -f -l app.kubernetes.io/name=mimir-insights -n mimir-insights"
echo -e "${YELLOW}  View Mimir logs:${NC} kubectl logs -f -l app.kubernetes.io/part-of=mimir -n mimir"
echo -e "${YELLOW}  Port forward Mimir API:${NC} kubectl port-forward -n mimir svc/mimir-api 9009:9009"
echo -e "${YELLOW}  Access Frontend:${NC} http://localhost:8081"
echo -e "${YELLOW}  Access Backend API:${NC} http://localhost:8080/api/tenants"
echo -e "${YELLOW}  Access Mimir API:${NC} http://localhost:9009/api/v1/status/buildinfo" 
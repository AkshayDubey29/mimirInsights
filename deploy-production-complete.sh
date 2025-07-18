#!/bin/bash

# Complete Production Deployment Script for MimirInsights
# This script handles image pull secrets and deployment

set -e

# Configuration
NAMESPACE="mimir-insights"
RELEASE_NAME="mimir-insights"
VALUES_FILE="deployments/helm-chart/values-production-final.yaml"

echo "🚀 Starting MimirInsights Production Deployment..."

# Step 1: Check prerequisites
echo "📋 Checking prerequisites..."

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed or not in PATH"
    exit 1
fi

# Check if helm is available
if ! command -v helm &> /dev/null; then
    echo "❌ helm is not installed or not in PATH"
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &> /dev/null; then
    echo "❌ Cannot connect to Kubernetes cluster"
    exit 1
fi

echo "✅ Prerequisites check passed"

# Step 2: Create namespace if it doesn't exist
echo "📦 Creating namespace if needed..."
if ! kubectl get namespace $NAMESPACE &> /dev/null; then
    kubectl create namespace $NAMESPACE
    echo "✅ Namespace $NAMESPACE created"
else
    echo "✅ Namespace $NAMESPACE already exists"
fi

# Step 3: Create image pull secret if it doesn't exist
echo "🔐 Setting up image pull secret..."
if ! kubectl get secret ghcr-secret -n $NAMESPACE &> /dev/null; then
    echo "Please enter your GitHub Personal Access Token (PAT) with 'read:packages' scope:"
    read -s GITHUB_TOKEN
    
    if [ -z "$GITHUB_TOKEN" ]; then
        echo "❌ GitHub token is required"
        exit 1
    fi
    
    kubectl create secret docker-registry ghcr-secret \
        --namespace=$NAMESPACE \
        --docker-server=ghcr.io \
        --docker-username=akshaydubey29 \
        --docker-password=$GITHUB_TOKEN \
        --docker-email="akshaydubey29@gmail.com"
    
    echo "✅ Image pull secret created"
else
    echo "✅ Image pull secret already exists"
fi

# Step 4: Validate Helm chart
echo "🔍 Validating Helm chart..."
if ! helm template $RELEASE_NAME ./deployments/helm-chart --values $VALUES_FILE --namespace $NAMESPACE &> /dev/null; then
    echo "❌ Helm chart validation failed"
    exit 1
fi
echo "✅ Helm chart validation passed"

# Step 5: Deploy with Helm
echo "🚀 Deploying MimirInsights..."
helm upgrade --install $RELEASE_NAME \
    ./deployments/helm-chart \
    --namespace $NAMESPACE \
    --values $VALUES_FILE \
    --wait \
    --timeout 10m \
    --create-namespace

echo "✅ Deployment completed successfully!"

# Step 6: Verify deployment
echo "🔍 Verifying deployment..."

# Wait a moment for pods to start
sleep 10

# Check pod status
echo "📊 Pod status:"
kubectl get pods -n $NAMESPACE

# Check services
echo "🌐 Services:"
kubectl get svc -n $NAMESPACE

# Check ingress
echo "🔗 Ingress:"
kubectl get ingress -n $NAMESPACE

# Check HPA
echo "⚖️  Horizontal Pod Autoscaler:"
kubectl get hpa -n $NAMESPACE

echo ""
echo "🎉 MimirInsights Production Deployment Complete!"
echo ""
echo "📋 Next steps:"
echo "1. Wait for all pods to be in 'Running' status"
echo "2. Configure DNS to point to your ingress"
echo "3. Access the application at: https://mimir-insights.yourdomain.com"
echo ""
echo "🔧 Useful commands:"
echo "  # Check pod logs"
echo "  kubectl logs -f deployment/mimir-insights-backend -n $NAMESPACE"
echo "  kubectl logs -f deployment/mimir-insights-frontend -n $NAMESPACE"
echo ""
echo "  # Port forward for local testing"
echo "  kubectl port-forward service/mimir-insights-frontend 8081:80 -n $NAMESPACE"
echo "  kubectl port-forward service/mimir-insights-backend 8080:8080 -n $NAMESPACE"
echo ""
echo "  # Check events"
echo "  kubectl get events -n $NAMESPACE --sort-by='.lastTimestamp'" 
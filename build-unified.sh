#!/bin/bash

# ==============================================================================
# Unified Build Script for MimirInsights
# Efficiently builds both frontend and backend containers
# ==============================================================================

set -e

echo "🚀 Unified Build Process Started..."
echo "💡 Building both frontend and backend with optimized strategy"
echo ""

# Build variables
FRONTEND_IMAGE="${1:-mimir-insights-frontend}"
BACKEND_IMAGE="${2:-mimir-insights-backend}"
TAG="${3:-unified-$(date +%s)}"

echo "📦 Target Images:"
echo "  Frontend: ${FRONTEND_IMAGE}:${TAG}"
echo "  Backend: ${BACKEND_IMAGE}:${TAG}"
echo ""

# Step 1: Build React application locally (if needed)
if [ ! -d "web-ui/build" ] || [ "web-ui/package.json" -nt "web-ui/build" ]; then
    echo "🏗️  Building React application..."
    cd web-ui
    
    # Clean any previous build
    rm -rf build/
    
    # Install dependencies if needed
    if [ ! -d "node_modules" ] || [ "package.json" -nt "node_modules/.package-lock.json" ]; then
        echo "📦 Installing npm dependencies..."
        npm ci --prefer-offline --no-audit --no-fund
    fi
    
    # Build with optimizations
    echo "⚡ Building with performance optimizations..."
    DISABLE_ESLINT_PLUGIN=true \
    GENERATE_SOURCEMAP=false \
    SKIP_PREFLIGHT_CHECK=true \
    npm run build
    
    echo "✅ React build completed!"
    cd ..
else
    echo "✅ Using existing React build (up to date)"
fi

# Step 2: Build backend (Go builds are fast)
echo ""
echo "⚙️  Building backend Docker image..."
docker build \
    --file Dockerfile.backend \
    --tag "${BACKEND_IMAGE}:${TAG}" \
    --no-cache \
    .

echo "✅ Backend Docker image created!"

# Step 3: Build frontend (uses pre-built files)
echo ""
echo "🎨 Building frontend Docker image..."
docker build \
    --file Dockerfile.frontend.simple \
    --tag "${FRONTEND_IMAGE}:${TAG}" \
    --no-cache \
    .

echo "✅ Frontend Docker image created!"

# Step 4: Show results
echo ""
echo "📊 Build Results:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}" | head -1
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}" | grep -E "${FRONTEND_IMAGE}:${TAG}|${BACKEND_IMAGE}:${TAG}"
echo ""

# Step 5: Deploy to Kubernetes using Helm
echo "🚀 Deploying to Kubernetes using Helm..."
echo ""

# Check if namespace exists
if ! kubectl get namespace mimir-insights >/dev/null 2>&1; then
    echo "📦 Creating mimir-insights namespace..."
    kubectl create namespace mimir-insights
fi

# Deploy using Helm
echo "📦 Deploying with Helm..."
helm upgrade --install mimir-insights \
    ./deployments/helm-chart \
    --namespace mimir-insights \
    --set frontend.image.repository=${FRONTEND_IMAGE} \
    --set frontend.image.tag=${TAG} \
    --set backend.image.repository=${BACKEND_IMAGE} \
    --set backend.image.tag=${TAG} \
    --set frontend.replicaCount=1 \
    --set backend.replicaCount=1 \
    --wait \
    --timeout=5m

echo ""
echo "✅ Deployment completed!"
echo ""
echo "📋 Next Steps:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "# Check deployment status:"
echo "kubectl get pods -n mimir-insights"
echo ""
echo "# Port forward to access the application:"
echo "kubectl port-forward service/mimir-insights-frontend 8081:80 -n mimir-insights &"
echo "kubectl port-forward service/mimir-insights-backend 8080:8080 -n mimir-insights &"
echo ""
echo "# Access the application:"
echo "Frontend: http://localhost:8081"
echo "Backend API: http://localhost:8080"
echo ""
echo "🎉 Unified build and deployment completed successfully!" 
#!/bin/bash

# ==============================================================================
# Memory-Safe Build Script for MimirInsights
# Builds React locally to avoid Docker memory issues
# ==============================================================================

set -e

echo "🧠 Memory-Safe Build Process Started..."
echo "💡 This approach builds React locally to avoid Docker memory constraints"
echo ""

# Build variables
FRONTEND_IMAGE="${1:-mimir-insights-frontend}"
BACKEND_IMAGE="${2:-mimir-insights-backend}"
TAG="${3:-memory-safe-$(date +%s)}"

echo "📦 Target Images:"
echo "  Frontend: ${FRONTEND_IMAGE}:${TAG}"
echo "  Backend: ${BACKEND_IMAGE}:${TAG}"
echo ""

# Step 1: Check if React build exists, build if needed
if [ ! -d "web-ui/build" ] || [ "web-ui/package.json" -nt "web-ui/build" ]; then
    echo "🏗️  Building React application locally..."
    echo "💾 Using system memory instead of Docker memory"
    
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

# Step 2: Create lightweight Docker images
echo ""
echo "🐳 Creating Docker images (memory-efficient)..."

# Build frontend (uses pre-built files, very fast)
echo "🎨 Building frontend Docker image..."
docker build \
    --file Dockerfile.frontend.local \
    --tag "${FRONTEND_IMAGE}:${TAG}" \
    .

echo "✅ Frontend Docker image created!"

# Build backend (Go builds are memory-efficient)
echo "⚙️  Building backend Docker image..."
docker build \
    --file Dockerfile.backend \
    --tag "${BACKEND_IMAGE}:${TAG}" \
    .

echo "✅ Backend Docker image created!"

# Step 3: Show results
echo ""
echo "📊 Build Results:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}" | head -1
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}" | grep -E "${FRONTEND_IMAGE}:${TAG}|${BACKEND_IMAGE}:${TAG}"
echo ""

# Step 4: Provide deployment commands
echo "🚀 Ready to Deploy!"
echo ""
echo "📋 Next Steps:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "# 1. Tag for Kubernetes registry"
echo "docker tag ${FRONTEND_IMAGE}:${TAG} ghcr.io/akshaydubey29/mimir-insights-ui:${TAG}"
echo "docker tag ${BACKEND_IMAGE}:${TAG} ghcr.io/akshaydubey29/mimir-insights-backend:${TAG}"
echo ""
echo "# 2. Load into kind cluster"
echo "kind load docker-image ghcr.io/akshaydubey29/mimir-insights-ui:${TAG} --name mimirinsights-test"
echo "kind load docker-image ghcr.io/akshaydubey29/mimir-insights-backend:${TAG} --name mimirinsights-test"
echo ""
echo "# 3. Deploy to Kubernetes"
echo "kubectl set image deployment/mimir-insights-frontend mimir-insights-frontend=ghcr.io/akshaydubey29/mimir-insights-ui:${TAG} -n mimir-insights"
echo "kubectl set image deployment/mimir-insights-backend mimir-insights-backend=ghcr.io/akshaydubey29/mimir-insights-backend:${TAG} -n mimir-insights"
echo ""

# Step 5: Check current memory usage
echo "💾 Current Docker Memory Usage:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
docker system info | grep "Total Memory"
echo ""
echo "💡 TIP: For faster builds, consider increasing Docker memory to 8GB+ in Docker Desktop settings"
echo "🎉 Memory-safe build completed successfully!" 
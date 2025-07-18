#!/bin/bash

# ==============================================================================
# Local Multi-Architecture Build Script for MimirInsights
# Builds both frontend and backend containers for local testing
# Supports: linux/amd64, linux/arm64 (for Apple Silicon compatibility)
# ==============================================================================

set -e

echo "🚀 Local Multi-Architecture Build Process Started..."
echo "📦 Building containers for local testing"
echo ""

# Configuration
REGISTRY="ghcr.io/akshaydubey29"
FRONTEND_IMAGE="${REGISTRY}/mimir-insights-frontend"
BACKEND_IMAGE="${REGISTRY}/mimir-insights-backend"
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
LOCAL_TAG="local-${TIMESTAMP}"

# Target architectures (reduced for local testing)
PLATFORMS="linux/amd64,linux/arm64"

echo "📦 Target Images:"
echo "  Frontend: ${FRONTEND_IMAGE}:${LOCAL_TAG}"
echo "  Backend: ${BACKEND_IMAGE}:${LOCAL_TAG}"
echo "  Platforms: ${PLATFORMS}"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
print_status "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed or not in PATH"
    exit 1
fi

if ! command -v go &> /dev/null; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

if ! command -v npm &> /dev/null; then
    print_error "npm is not installed or not in PATH"
    exit 1
fi

# Check if Docker Buildx is available
if ! docker buildx version &> /dev/null; then
    print_error "Docker Buildx is not available. Please install Docker Desktop or enable buildx."
    exit 1
fi

print_success "All prerequisites are available"

# Setup Docker Buildx builder
print_status "Setting up Docker Buildx builder..."

# Create a new builder instance if it doesn't exist
if ! docker buildx inspect local-multiarch-builder &> /dev/null; then
    print_status "Creating new buildx builder instance for local builds..."
    docker buildx create --name local-multiarch-builder --use
else
    print_status "Using existing buildx builder instance..."
    docker buildx use local-multiarch-builder
fi

# Bootstrap the builder
print_status "Bootstrapping buildx builder..."
docker buildx inspect --bootstrap

print_success "Buildx builder is ready"

# Step 1: Build React application
print_status "Building React application..."

cd web-ui

# Clean any previous build
rm -rf build/

# Install dependencies
print_status "Installing npm dependencies..."
npm ci --prefer-offline --no-audit --no-fund

# Build with production optimizations
print_status "Building React application with production optimizations..."
DISABLE_ESLINT_PLUGIN=true \
GENERATE_SOURCEMAP=false \
SKIP_PREFLIGHT_CHECK=true \
REACT_APP_ENABLE_MOCK_DATA=false \
REACT_APP_ENABLE_MONITORING=false \
REACT_APP_ENABLE_LLM=false \
REACT_APP_API_BASE_URL=/api \
npm run build

print_success "React build completed!"
cd ..

# Step 2: Build multi-arch backend (load to local registry)
print_status "Building multi-architecture backend container..."

# Create multi-arch backend Dockerfile
cat > Dockerfile.backend.local << 'EOF'
# Build stage for each architecture
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod tidy

# Copy source code
COPY . .

# Build the application for the target platform
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN echo "Building for $TARGETOS/$TARGETARCH" && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -a -installsuffix cgo \
    -ldflags="-w -s -X main.Version=${LOCAL_TAG} -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S_UTC')" \
    -o main ./cmd/server

# Final stage
FROM alpine:3.19

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/main .

# Copy config files
COPY --from=builder /app/config ./config

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/health || exit 1

# Run the application
CMD ["./main"]
EOF

# Build multi-arch backend and load to local registry
print_status "Building multi-architecture backend (loading to local registry)..."
docker buildx build \
    --file Dockerfile.backend.local \
    --platform ${PLATFORMS} \
    --tag "${BACKEND_IMAGE}:${LOCAL_TAG}" \
    --load \
    .

print_success "Multi-architecture backend built and loaded locally!"

# Step 3: Build multi-arch frontend (load to local registry)
print_status "Building multi-architecture frontend container..."

# Create multi-arch frontend Dockerfile
cat > Dockerfile.frontend.local << 'EOF'
FROM nginx:1.26-alpine

# Install necessary packages
RUN apk add --no-cache \
    wget \
    curl \
    && rm -rf /var/cache/apk/*

# Copy the pre-built React application
COPY web-ui/build /usr/share/nginx/html

# Copy nginx configuration
COPY web-ui/nginx.conf /etc/nginx/conf.d/default.conf

# Set proper permissions
RUN chmod -R 755 /usr/share/nginx/html && \
    chown -R nginx:nginx /usr/share/nginx/html

# Create startup script
RUN echo '#!/bin/sh' > /start.sh && \
    echo 'set -e' >> /start.sh && \
    echo '' >> /start.sh && \
    echo 'echo "🚀 Starting MimirInsights Frontend..."' >> /start.sh && \
    echo 'echo "📅 Version: LOCAL-'"${LOCAL_TAG}"'"' >> /start.sh && \
    echo 'echo "🏗️  Architecture: $(uname -m)"' >> /start.sh && \
    echo '' >> /start.sh && \
    echo '# Create runtime configuration' >> /start.sh && \
    echo 'cat > /usr/share/nginx/html/config/config.js << EOFCONFIG' >> /start.sh && \
    echo 'window.CONFIG = {' >> /start.sh && \
    echo '  apiBaseUrl: "http://mimir-insights-backend:8080",' >> /start.sh && \
    echo '  environment: "local",' >> /start.sh && \
    echo '  version: "LOCAL-'"${LOCAL_TAG}"'",' >> /start.sh && \
    echo '  architecture: "$(uname -m)",' >> /start.sh && \
    echo '  features: {' >> /start.sh && \
    echo '    autoRefresh: true,' >> /start.sh && \
    echo '    mockData: false' >> /start.sh && \
    echo '  }' >> /start.sh && \
    echo '};' >> /start.sh && \
    echo 'EOFCONFIG' >> /start.sh && \
    echo '' >> /start.sh && \
    echo 'echo "✅ Configuration created"' >> /start.sh && \
    echo '' >> /start.sh && \
    echo '# Test nginx configuration' >> /start.sh && \
    echo 'nginx -t' >> /start.sh && \
    echo '' >> /start.sh && \
    echo '# Start nginx' >> /start.sh && \
    echo 'echo "🌐 Starting nginx..."' >> /start.sh && \
    echo 'exec nginx -g "daemon off;"' >> /start.sh && \
    chmod +x /start.sh

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost/ || exit 1

EXPOSE 80

ENTRYPOINT ["/start.sh"]
EOF

# Build multi-arch frontend and load to local registry
print_status "Building multi-architecture frontend (loading to local registry)..."
docker buildx build \
    --file Dockerfile.frontend.local \
    --platform ${PLATFORMS} \
    --tag "${FRONTEND_IMAGE}:${LOCAL_TAG}" \
    --load \
    .

print_success "Multi-architecture frontend built and loaded locally!"

# Step 4: Verify the local images
print_status "Verifying local multi-architecture images..."

echo ""
echo "🔍 Checking local image manifests..."

# Check backend image
print_status "Backend image info:"
docker images | grep "${BACKEND_IMAGE}:${LOCAL_TAG}" || echo "Backend image not found"

echo ""
print_status "Frontend image info:"
docker images | grep "${FRONTEND_IMAGE}:${LOCAL_TAG}" || echo "Frontend image not found"

# Cleanup temporary files
print_status "Cleaning up temporary files..."
rm -f Dockerfile.backend.local Dockerfile.frontend.local

print_success "Local multi-architecture build completed successfully!"
echo ""
echo "🎉 Summary:"
echo "  ✅ Backend: ${BACKEND_IMAGE}:${LOCAL_TAG}"
echo "  ✅ Frontend: ${FRONTEND_IMAGE}:${LOCAL_TAG}"
echo "  ✅ Platforms: ${PLATFORMS}"
echo ""
echo "🚀 Images are now available locally for testing!"
echo "   - Compatible with AMD64 (x86_64) systems"
echo "   - Compatible with ARM64 (Apple Silicon) systems"
echo ""
echo "📋 To test locally, you can run:"
echo "   docker run -p 8080:8080 ${BACKEND_IMAGE}:${LOCAL_TAG}"
echo "   docker run -p 8081:80 ${FRONTEND_IMAGE}:${LOCAL_TAG}"
echo ""
echo "📋 To deploy to Kubernetes, update your Helm values:"
echo "   frontend.image: ${FRONTEND_IMAGE}:${LOCAL_TAG}"
echo "   backend.image: ${BACKEND_IMAGE}:${LOCAL_TAG}" 
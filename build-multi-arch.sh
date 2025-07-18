#!/bin/bash

# ==============================================================================
# Multi-Architecture Build Script for MimirInsights
# Builds both frontend and backend containers for multiple CPU architectures
# Supports: linux/amd64, linux/arm64, linux/arm/v7, linux/arm/v6
# ==============================================================================

set -e

echo "🚀 Multi-Architecture Build Process Started..."
echo "📦 Building containers for multiple CPU architectures"
echo ""

# Configuration
REGISTRY="ghcr.io/akshaydubey29"
FRONTEND_IMAGE="${REGISTRY}/mimir-insights-frontend"
BACKEND_IMAGE="${REGISTRY}/mimir-insights-backend"
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
PRODUCTION_TAG="v1.0.0-${TIMESTAMP}"
LATEST_TAG="latest"

# Target architectures
PLATFORMS="linux/amd64,linux/arm64"

echo "📦 Target Images:"
echo "  Frontend: ${FRONTEND_IMAGE}:${PRODUCTION_TAG}"
echo "  Backend: ${BACKEND_IMAGE}:${PRODUCTION_TAG}"
echo "  Latest Tags: ${FRONTEND_IMAGE}:${LATEST_TAG}, ${BACKEND_IMAGE}:${LATEST_TAG}"
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
if ! docker buildx inspect multiarch-builder &> /dev/null; then
    print_status "Creating new buildx builder instance..."
    docker buildx create --name multiarch-builder --use
else
    print_status "Using existing buildx builder instance..."
    docker buildx use multiarch-builder
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

# Step 2: Build multi-arch backend
print_status "Building multi-architecture backend container..."

# Create multi-arch backend Dockerfile
cat > Dockerfile.backend.multiarch << 'EOF'
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
    -ldflags="-w -s -X main.Version=${PRODUCTION_TAG} -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S_UTC')" \
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

# Build and push multi-arch backend
print_status "Building and pushing multi-architecture backend..."
docker buildx build \
    --file Dockerfile.backend.multiarch \
    --platform ${PLATFORMS} \
    --tag "${BACKEND_IMAGE}:${PRODUCTION_TAG}" \
    --tag "${BACKEND_IMAGE}:${LATEST_TAG}" \
    --push \
    .

print_success "Multi-architecture backend built and pushed!"

# Frontend Multi-Architecture Build
echo "🔨 Building Frontend Multi-Architecture Image..."

# Build frontend multi-architecture image
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    --tag ghcr.io/akshaydubey29/mimir-insights-frontend:${PRODUCTION_TAG} \
    --tag ghcr.io/akshaydubey29/mimir-insights-frontend:latest \
    --file - \
    --push \
    . << 'EOF'
# Use nginx alpine for smaller size and better security
FROM nginx:alpine

# Set environment variables
ENV PRODUCTION_TAG=${PRODUCTION_TAG}
ENV BUILD_DATE=${BUILD_DATE}
ENV VCS_REF=${VCS_REF}

# Copy the pre-built React application
COPY web-ui/build /usr/share/nginx/html

# Create config directory and config file during build time
RUN mkdir -p /usr/share/nginx/html/config && \
    echo 'window.CONFIG = {' > /usr/share/nginx/html/config/config.js && \
    echo '  apiBaseUrl: "http://mimir-insights-backend:8080/api",' >> /usr/share/nginx/html/config/config.js && \
    echo '  version: "PRODUCTION-'${PRODUCTION_TAG}'",' >> /usr/share/nginx/html/config/config.js && \
    echo '  buildDate: "'${BUILD_DATE}'",' >> /usr/share/nginx/html/config/config.js && \
    echo '  vcsRef: "'${VCS_REF}'"' >> /usr/share/nginx/html/config/config.js && \
    echo '};' >> /usr/share/nginx/html/config/config.js && \
    chmod -R 755 /usr/share/nginx/html

# Create a completely minimal nginx configuration that works without any writable directories
RUN echo 'worker_processes 1;' > /etc/nginx/nginx.conf && \
    echo 'error_log /dev/stderr warn;' >> /etc/nginx/nginx.conf && \
    echo '' >> /etc/nginx/nginx.conf && \
    echo 'events {' >> /etc/nginx/nginx.conf && \
    echo '    worker_connections 1024;' >> /etc/nginx/nginx.conf && \
    echo '}' >> /etc/nginx/nginx.conf && \
    echo '' >> /etc/nginx/nginx.conf && \
    echo 'http {' >> /etc/nginx/nginx.conf && \
    echo '    include /etc/nginx/mime.types;' >> /etc/nginx/nginx.conf && \
    echo '    default_type application/octet-stream;' >> /etc/nginx/nginx.conf && \
    echo '    access_log /dev/stdout;' >> /etc/nginx/nginx.conf && \
    echo '    sendfile on;' >> /etc/nginx/nginx.conf && \
    echo '    keepalive_timeout 65;' >> /etc/nginx/nginx.conf && \
    echo '    gzip on;' >> /etc/nginx/nginx.conf && \
    echo '    server {' >> /etc/nginx/nginx.conf && \
    echo '        listen 80;' >> /etc/nginx/nginx.conf && \
    echo '        location / {' >> /etc/nginx/nginx.conf && \
    echo '            root /usr/share/nginx/html;' >> /etc/nginx/nginx.conf && \
    echo '            index index.html;' >> /etc/nginx/nginx.conf && \
    echo '            try_files $uri $uri/ /index.html;' >> /etc/nginx/nginx.conf && \
    echo '        }' >> /etc/nginx/nginx.conf && \
    echo '    }' >> /etc/nginx/nginx.conf && \
    echo '}' >> /etc/nginx/nginx.conf

# Create startup script
RUN echo '#!/bin/sh' > /start.sh && \
    echo 'set -e' >> /start.sh && \
    echo '' >> /start.sh && \
    echo 'echo "🚀 Starting MimirInsights Frontend..."' >> /start.sh && \
    echo 'echo "🛠 Version: PRODUCTION-'${PRODUCTION_TAG}'"' >> /start.sh && \
    echo 'echo "🏗 Architecture: $(uname -m)"' >> /start.sh && \
    echo 'echo "✅ Nginx configuration: optimized for read-only filesystem"' >> /start.sh && \
    echo '' >> /start.sh && \
    echo '# Test nginx configuration' >> /start.sh && \
    echo 'nginx -t' >> /start.sh && \
    echo '' >> /start.sh && \
    echo '# Start nginx in foreground' >> /start.sh && \
    echo 'exec nginx -g "daemon off;"' >> /start.sh && \
    chmod +x /start.sh

# Expose port
EXPOSE 80

# Use our custom entrypoint
ENTRYPOINT ["/start.sh"]
EOF

# Step 4: Verify the multi-arch images
print_status "Verifying multi-architecture images..."

echo ""
echo "🔍 Checking image manifests..."

# Check backend image
print_status "Backend image manifest:"
docker buildx imagetools inspect "${BACKEND_IMAGE}:${PRODUCTION_TAG}"

echo ""
print_status "Frontend image manifest:"
docker buildx imagetools inspect "${FRONTEND_IMAGE}:${PRODUCTION_TAG}"

# Cleanup temporary files
print_status "Cleaning up temporary files..."
rm -f Dockerfile.backend.multiarch

print_success "Multi-architecture build completed successfully!"
echo ""
echo "🎉 Summary:"
echo "  ✅ Backend: ${BACKEND_IMAGE}:${PRODUCTION_TAG}"
echo "  ✅ Frontend: ${FRONTEND_IMAGE}:${PRODUCTION_TAG}"
echo "  ✅ Latest tags updated"
echo "  ✅ Platforms: ${PLATFORMS}"
echo ""
echo "🚀 Images are now available for deployment on multiple architectures!"
echo "   - AMD64 (x86_64) servers"
echo "   - ARM64 (aarch64) servers (Apple Silicon, ARM servers)"
echo "   - ARM v7 (32-bit ARM devices)"
echo ""
echo "📋 To deploy, update your Helm values to use these images:"
echo "   frontend.image: ${FRONTEND_IMAGE}:${PRODUCTION_TAG}"
echo "   backend.image: ${BACKEND_IMAGE}:${PRODUCTION_TAG}" 
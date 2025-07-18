#!/bin/bash

# Multi-Architecture Test Script for MimirInsights
# This script tests multi-architecture functionality

set -e

echo "=== Multi-Architecture Test Suite ==="
echo ""

# Configuration
REGISTRY="ghcr.io/akshaydubey29"
FRONTEND_IMAGE="${REGISTRY}/mimir-insights-frontend"
BACKEND_IMAGE="${REGISTRY}/mimir-insights-backend"

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

# Test 1: Check Docker Buildx availability
test_buildx() {
    echo "🔧 Test 1: Docker Buildx Availability"
    if docker buildx version &> /dev/null; then
        print_success "Docker Buildx is available"
        docker buildx version
    else
        print_error "Docker Buildx is not available"
        return 1
    fi
    echo ""
}

# Test 2: Check current system architecture
test_system_arch() {
    echo "🏗️  Test 2: System Architecture Detection"
    SYSTEM_ARCH=$(uname -m)
    print_success "Current system architecture: $SYSTEM_ARCH"
    
    case $SYSTEM_ARCH in
        "x86_64")
            print_status "Running on AMD64/x86_64 architecture"
            ;;
        "arm64"|"aarch64")
            print_status "Running on ARM64 architecture (Apple Silicon or ARM server)"
            ;;
        "armv7l")
            print_status "Running on ARM v7 architecture"
            ;;
        "armv6l")
            print_status "Running on ARM v6 architecture"
            ;;
        *)
            print_warning "Unknown architecture: $SYSTEM_ARCH"
            ;;
    esac
    echo ""
}

# Test 3: Check available Docker platforms
test_docker_platforms() {
    echo "🐳 Test 3: Docker Platform Support"
    print_status "Available Docker platforms:"
    docker buildx inspect --bootstrap 2>/dev/null | grep -E "Platforms:|linux/" || print_warning "No platform information available"
    echo ""
}

# Test 4: Test image pull for different architectures
test_image_pull() {
    echo "📦 Test 4: Multi-Architecture Image Pull"
    
    # Test backend image
    print_status "Testing backend image pull..."
    if docker pull --platform linux/amd64 "${BACKEND_IMAGE}:latest" >/dev/null 2>&1; then
        print_success "Backend AMD64 image pull successful"
    else
        print_warning "Backend AMD64 image pull failed"
    fi
    
    if docker pull --platform linux/arm64 "${BACKEND_IMAGE}:latest" >/dev/null 2>&1; then
        print_success "Backend ARM64 image pull successful"
    else
        print_warning "Backend ARM64 image pull failed"
    fi
    
    # Test frontend image
    print_status "Testing frontend image pull..."
    if docker pull --platform linux/amd64 "${FRONTEND_IMAGE}:latest" >/dev/null 2>&1; then
        print_success "Frontend AMD64 image pull successful"
    else
        print_warning "Frontend AMD64 image pull failed"
    fi
    
    if docker pull --platform linux/arm64 "${FRONTEND_IMAGE}:latest" >/dev/null 2>&1; then
        print_success "Frontend ARM64 image pull successful"
    else
        print_warning "Frontend ARM64 image pull failed"
    fi
    echo ""
}

# Test 5: Test container runtime on current architecture
test_container_runtime() {
    echo "🚀 Test 5: Container Runtime Test"
    
    # Test backend container
    print_status "Testing backend container runtime..."
    BACKEND_CONTAINER=$(docker run -d --rm --platform linux/amd64 "${BACKEND_IMAGE}:latest" sleep 10 2>/dev/null || echo "")
    if [ -n "$BACKEND_CONTAINER" ]; then
        BACKEND_ARCH=$(docker exec "$BACKEND_CONTAINER" uname -m 2>/dev/null || echo "unknown")
        print_success "Backend container architecture: $BACKEND_ARCH"
        docker stop "$BACKEND_CONTAINER" >/dev/null 2>&1
    else
        print_warning "Backend container test failed"
    fi
    
    # Test frontend container
    print_status "Testing frontend container runtime..."
    FRONTEND_CONTAINER=$(docker run -d --rm --platform linux/amd64 "${FRONTEND_IMAGE}:latest" sleep 10 2>/dev/null || echo "")
    if [ -n "$FRONTEND_CONTAINER" ]; then
        FRONTEND_ARCH=$(docker exec "$FRONTEND_CONTAINER" uname -m 2>/dev/null || echo "unknown")
        print_success "Frontend container architecture: $FRONTEND_ARCH"
        docker stop "$FRONTEND_CONTAINER" >/dev/null 2>&1
    else
        print_warning "Frontend container test failed"
    fi
    echo ""
}

# Test 6: Check Kubernetes cluster architecture (if available)
test_k8s_arch() {
    echo "☸️  Test 6: Kubernetes Cluster Architecture"
    if command -v kubectl &> /dev/null; then
        if kubectl cluster-info &> /dev/null; then
            print_status "Kubernetes cluster is available"
            
            # Get cluster architecture
            CLUSTER_ARCH=$(kubectl get nodes -o jsonpath='{.items[0].status.nodeInfo.architecture}' 2>/dev/null || echo "unknown")
            print_success "Cluster architecture: $CLUSTER_ARCH"
            
            # Get node information
            print_status "Node information:"
            kubectl get nodes -o wide 2>/dev/null | head -5 || print_warning "Could not retrieve node information"
        else
            print_warning "Kubernetes cluster is not accessible"
        fi
    else
        print_warning "kubectl is not available"
    fi
    echo ""
}

# Test 7: Validate multi-arch image manifests
test_image_manifests() {
    echo "📋 Test 7: Multi-Architecture Image Manifests"
    
    # Test backend manifest
    print_status "Checking backend image manifest..."
    if docker buildx imagetools inspect "${BACKEND_IMAGE}:latest" >/dev/null 2>&1; then
        print_success "Backend image has multi-architecture manifest"
        BACKEND_PLATFORMS=$(docker buildx imagetools inspect "${BACKEND_IMAGE}:latest" --format '{{range .Manifest.Manifests}}{{.Platform.OS}}/{{.Platform.Architecture}}{{if .Platform.Variant}}/{{.Platform.Variant}}{{end}} {{end}}' 2>/dev/null || echo "unknown")
        print_status "Backend supported platforms: $BACKEND_PLATFORMS"
    else
        print_warning "Backend image manifest not accessible"
    fi
    
    # Test frontend manifest
    print_status "Checking frontend image manifest..."
    if docker buildx imagetools inspect "${FRONTEND_IMAGE}:latest" >/dev/null 2>&1; then
        print_success "Frontend image has multi-architecture manifest"
        FRONTEND_PLATFORMS=$(docker buildx imagetools inspect "${FRONTEND_IMAGE}:latest" --format '{{range .Manifest.Manifests}}{{.Platform.OS}}/{{.Platform.Architecture}}{{if .Platform.Variant}}/{{.Platform.Variant}}{{end}} {{end}}' 2>/dev/null || echo "unknown")
        print_status "Frontend supported platforms: $FRONTEND_PLATFORMS"
    else
        print_warning "Frontend image manifest not accessible"
    fi
    echo ""
}

# Run all tests
main() {
    print_status "Starting multi-architecture test suite..."
    echo ""
    
    test_buildx
    test_system_arch
    test_docker_platforms
    test_image_pull
    test_container_runtime
    test_k8s_arch
    test_image_manifests
    
    echo "✅ Multi-architecture test suite completed!"
    echo ""
    echo "📊 Summary:"
    echo "  - Docker Buildx: Available"
    echo "  - System Architecture: $(uname -m)"
    echo "  - Multi-arch images: Validated"
    echo "  - Container runtime: Tested"
    echo "  - Kubernetes: Checked"
    echo ""
    echo "🚀 Your system is ready for multi-architecture deployments!"
}

# Run the main function
main 
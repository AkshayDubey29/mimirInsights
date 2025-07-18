# Multi-Architecture Container Setup for MimirInsights

This document explains how to build and deploy MimirInsights containers for multiple CPU architectures, making them compatible with different platforms like AMD64, ARM64 (Apple Silicon), and ARM v7.

## Overview

MimirInsights now supports multi-architecture containers that can run on:
- **AMD64 (x86_64)**: Traditional Intel/AMD servers and desktops
- **ARM64 (aarch64)**: Apple Silicon Macs, ARM-based servers (AWS Graviton, etc.)
- **ARM v7**: 32-bit ARM devices (Raspberry Pi 3, etc.)

## Prerequisites

### Required Tools
- Docker Desktop with Buildx support
- Go 1.21 or later
- Node.js 18 or later
- npm

### Docker Buildx Setup
Multi-architecture builds require Docker Buildx. Most modern Docker installations include it by default.

To verify Buildx is available:
```bash
docker buildx version
```

If not available, install Docker Desktop or enable Buildx manually.

## Build Scripts

### 1. Production Multi-Arch Build (`build-multi-arch.sh`)

This script builds and pushes multi-architecture images to GitHub Container Registry.

**Features:**
- Builds for all supported architectures
- Pushes to ghcr.io/akshaydubey29
- Creates both versioned and latest tags
- Includes comprehensive health checks

**Usage:**
```bash
./build-multi-arch.sh
```

**Supported Platforms:**
- linux/amd64
- linux/arm64
- linux/arm/v7

### 2. Local Multi-Arch Build (`build-multi-arch-local.sh`)

This script builds multi-architecture images for local testing.

**Features:**
- Builds for AMD64 and ARM64 only (faster builds)
- Loads images to local Docker registry
- Perfect for development and testing
- No registry push required

**Usage:**
```bash
./build-multi-arch-local.sh
```

**Supported Platforms:**
- linux/amd64
- linux/arm64

## Architecture-Specific Optimizations

### Backend (Go Application)

The backend uses Go's cross-compilation capabilities:

```dockerfile
# Build stage for each architecture
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

# Build the application for the target platform
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o main ./cmd/server
```

**Key Features:**
- CGO disabled for maximum compatibility
- Stripped binaries for smaller size
- Platform-specific optimizations
- Non-root user for security

### Frontend (React + Nginx)

The frontend uses Nginx's multi-architecture support:

```dockerfile
FROM nginx:1.26-alpine

# Copy pre-built React application
COPY web-ui/build /usr/share/nginx/html

# Runtime configuration injection
RUN echo '#!/bin/sh' > /start.sh && \
    echo 'echo "🏗️  Architecture: $(uname -m)"' >> /start.sh && \
    # ... configuration setup
```

**Key Features:**
- Pre-built React application
- Runtime architecture detection
- Optimized Nginx configuration
- Health checks included

## Deployment

### Kubernetes Deployment

Update your Helm values to use multi-architecture images:

```yaml
# values.yaml
frontend:
  image: ghcr.io/akshaydubey29/mimir-insights-frontend:v1.0.0-20241217-143022
  imagePullPolicy: Always

backend:
  image: ghcr.io/akshaydubey29/mimir-insights-backend:v1.0.0-20241217-143022
  imagePullPolicy: Always
```

### Local Testing

Test multi-architecture images locally:

```bash
# Test backend
docker run -p 8080:8080 ghcr.io/akshaydubey29/mimir-insights-backend:local-20241217-143022

# Test frontend
docker run -p 8081:80 ghcr.io/akshaydubey29/mimir-insights-frontend:local-20241217-143022
```

## Verification

### Check Image Manifests

Verify multi-architecture support:

```bash
# Check production images
docker buildx imagetools inspect ghcr.io/akshaydubey29/mimir-insights-backend:v1.0.0-20241217-143022

# Check local images
docker images | grep mimir-insights
```

### Architecture Detection

The containers include runtime architecture detection:

```javascript
// Frontend configuration
window.CONFIG = {
  architecture: "$(uname -m)",
  // ... other config
};
```

## Performance Considerations

### Build Time
- **Production build**: ~10-15 minutes (all architectures)
- **Local build**: ~5-8 minutes (AMD64 + ARM64 only)

### Image Size
- **Backend**: ~15-20MB per architecture
- **Frontend**: ~50-60MB per architecture
- **Total**: ~200-300MB for all architectures

### Runtime Performance
- Native performance on target architecture
- No emulation overhead
- Optimized binaries for each platform

## Troubleshooting

### Common Issues

1. **Buildx not available**
   ```bash
   # Install Docker Desktop or enable buildx
   docker buildx install
   ```

2. **Memory issues during build**
   ```bash
   # Increase Docker memory allocation to 8GB+
   # Use build-multi-arch-local.sh for faster builds
   ```

3. **Platform-specific build failures**
   ```bash
   # Check platform support
   docker buildx ls
   
   # Use specific platform
   --platform linux/amd64,linux/arm64
   ```

### Debug Commands

```bash
# Check builder status
docker buildx inspect --bootstrap

# List available builders
docker buildx ls

# Check image details
docker buildx imagetools inspect <image>

# Test specific platform
docker run --rm --platform linux/arm64 <image> uname -m
```

## Best Practices

### Development
1. Use `build-multi-arch-local.sh` for development
2. Test on target architecture when possible
3. Use Docker Desktop on Apple Silicon for ARM64 testing

### Production
1. Use `build-multi-arch.sh` for production builds
2. Include all target architectures
3. Tag with version and timestamp
4. Push to registry for distribution

### Security
1. Use non-root users in containers
2. Include health checks
3. Scan images for vulnerabilities
4. Keep base images updated

## Future Enhancements

### Planned Features
- Support for ARM v6 (Raspberry Pi Zero)
- Automated vulnerability scanning
- Multi-stage optimization
- Platform-specific optimizations

### Monitoring
- Architecture-specific metrics
- Performance monitoring
- Resource usage tracking

## Support

For issues with multi-architecture builds:
1. Check Docker Buildx setup
2. Verify platform support
3. Review build logs
4. Test on target architecture

---

**Note**: Multi-architecture builds require more resources and time but provide maximum compatibility across different platforms. 
# MimirInsights CI/CD Workflow

## Overview

This project uses a streamlined CI/CD workflow that focuses on building multi-architecture Docker images in GitHub Actions and deploying them locally to a kind cluster for production simulation.

## Architecture

### CI/CD Pipeline
- **GitHub Actions**: Builds multi-architecture Docker images (linux/amd64, linux/arm64)
- **GitHub Container Registry (GHCR)**: Stores and distributes Docker images
- **Local Kind Cluster**: Simulates production environment for testing
- **Helm Charts**: Manages Kubernetes deployments

### Image Naming Convention
- **Frontend**: `ghcr.io/akshaydubey29/mimir-insights-frontend`
- **Backend**: `ghcr.io/akshaydubey29/mimir-insights-backend`
- **Tags**: Timestamp-based (e.g., `20250719-061613`) for production deployments

## Workflow Steps

### 1. CI Pipeline (GitHub Actions)

The CI pipeline automatically triggers on:
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches
- Manual workflow dispatch

**What it does:**
1. Builds multi-architecture Docker images using Docker Buildx
2. Pushes images to GHCR with timestamp tags
3. Tests images for basic functionality
4. Creates GitHub releases (on manual dispatch)

**Generated Tags:**
- `{timestamp}` (e.g., `20250719-061613`) - Primary production tag
- `{version}` (e.g., `v1.0.0`) - Version tag
- `{branch}` - Branch-specific tags for development

### 2. Local Deployment

After CI builds and pushes images:

1. **Update Values File**: The deployment script automatically updates `deployments/helm-chart/values-production-final.yaml` with the latest timestamp tag
2. **Deploy to Kind Cluster**: Uses Helm to deploy to local kind cluster
3. **Setup Port Forwarding**: Establishes local access to services
4. **Verify Deployment**: Runs health checks and API tests

## Files Structure

### Essential Files (Kept)
```
├── .github/workflows/build-multiarch.yml  # CI/CD pipeline
├── deploy-local.sh                        # Local deployment script
├── build-multi-arch.sh                    # Multi-arch build script
├── Dockerfile.backend                     # Backend Dockerfile
├── Dockerfile.frontend                    # Frontend Dockerfile
├── deployments/helm-chart/                # Helm chart
│   └── values-production-final.yaml       # Production values
└── create-image-pull-secret.sh            # Image pull secret setup
```

### Removed Files (Cleaned Up)
- Multiple build scripts (build-fast.sh, build-local-memory-safe.sh, etc.)
- Multiple Dockerfiles (Dockerfile.frontend.local, Dockerfile.frontend.simple, etc.)
- Multiple deployment scripts (deploy-production.sh, deploy-multi-arch.sh, etc.)
- Test and verification scripts

## Usage

### Building Images Locally
```bash
# Build multi-architecture images
./build-multi-arch.sh
```

### Deploying Locally
```bash
# Deploy to local kind cluster
./deploy-local.sh
```

### Manual CI Trigger
1. Go to GitHub Actions tab
2. Select "Build Multi-Architecture Docker Images"
3. Click "Run workflow"
4. Enter version (e.g., "v1.0.0")
5. Click "Run workflow"

## Local Development

### Prerequisites
- Docker with Buildx support
- kubectl
- helm
- kind cluster running
- jq (for JSON parsing)

### Kind Cluster Setup
```bash
# Create kind cluster
kind create cluster --name mimir-insights

# Verify cluster is running
kubectl cluster-info
```

### Access URLs
After deployment:
- **Frontend**: http://localhost:8081
- **Backend API**: http://localhost:8080/api/tenants
- **Health Check**: http://localhost:8080/health

## Production Simulation

The local kind cluster is configured to simulate production with:
- High resource allocation
- Production Mimir components
- Real data (no mock data)
- Production-like networking and security

## Troubleshooting

### Port Forwarding Issues
```bash
# Kill existing port-forward processes
pkill -f "kubectl port-forward"

# Check if ports are in use
lsof -i :8080
lsof -i :8081
```

### Image Pull Issues
```bash
# Create image pull secret
./create-image-pull-secret.sh

# Verify images exist
docker pull ghcr.io/akshaydubey29/mimir-insights-frontend:latest
docker pull ghcr.io/akshaydubey29/mimir-insights-backend:latest
```

### Helm Deployment Issues
```bash
# Check Helm chart
helm lint ./deployments/helm-chart

# Check values file
helm template mimir-insights ./deployments/helm-chart --values ./deployments/helm-chart/values-production-final.yaml
```

## Best Practices

1. **Always use timestamp tags** for production deployments
2. **Test locally** before pushing to main branch
3. **Monitor CI/CD pipeline** for build failures
4. **Keep kind cluster resources** high for production simulation
5. **Use Helm values** for environment-specific configurations

## Security

- Images are built with non-root users
- Read-only root filesystems
- Dropped capabilities
- Network policies enabled
- Resource quotas configured

## Monitoring

- Health checks configured for both frontend and backend
- Liveness and readiness probes enabled
- Startup probes for slow-starting containers
- Horizontal Pod Autoscaler configured 
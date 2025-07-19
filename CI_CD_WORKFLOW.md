# MimirInsights CI/CD Workflow

This document describes the CI/CD workflow for MimirInsights, which builds multi-architecture Docker images on GitHub Actions and enables local deployment using pre-built images.

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Local Dev     │    │  GitHub Actions  │    │   Local Deploy  │
│                 │    │                  │    │                 │
│ • Code Changes  │───▶│ • Multi-arch     │───▶│ • Pull Images   │
│ • Git Push      │    │   Build          │    │ • Deploy to     │
│                 │    │ • Push to GHCR   │    │   Kind Cluster  │
│                 │    │ • Create Release │    │ • Port Forward  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## 🚀 GitHub Actions Workflow

### Workflow File: `.github/workflows/build-multiarch.yml`

The workflow automatically triggers on:
- **Push to main/develop branches** (when relevant files change)
- **Pull requests** (for testing)
- **Manual dispatch** (for releases)

### Features

1. **Multi-Architecture Builds**
   - Builds for `linux/amd64` and `linux/arm64`
   - Uses Docker Buildx for efficient cross-platform builds
   - GitHub Actions cache for faster builds

2. **Automatic Tagging**
   - Branch-based tags (e.g., `main-abc123`)
   - SHA-based tags (e.g., `main-abc123def456`)
   - Latest tag for main branch
   - Version tags for releases

3. **Image Testing**
   - Runs basic health checks on built images
   - Validates image functionality before release

4. **Release Management**
   - Creates GitHub releases with deployment instructions
   - Includes image URLs and deployment examples

## 📦 Container Registry

Images are pushed to **GitHub Container Registry (GHCR)**:
- `ghcr.io/akshaydubey29/mimir-insights-frontend:latest`
- `ghcr.io/akshaydubey29/mimir-insights-backend:latest`
- `ghcr.io/akshaydubey29/mimir-insights-frontend:v1.0.0`
- `ghcr.io/akshaydubey29/mimir-insights-backend:v1.0.0`

## 🛠️ Local Deployment

### Prerequisites

1. **Kubernetes Cluster**
   ```bash
   # Start kind cluster
   kind start cluster
   ```

2. **Required Tools**
   - `kubectl`
   - `helm`
   - `docker`
   - `curl`
   - `jq`

3. **GitHub Access**
   - Ensure you have access to the repository
   - Images are public, so no authentication required

### Deployment Script

Use the `deploy-from-ci.sh` script for easy local deployment:

```bash
# Deploy latest version
./deploy-from-ci.sh

# Deploy specific version
./deploy-from-ci.sh v1.0.0

# Deploy from specific branch
./deploy-from-ci.sh main-abc123
```

### What the Script Does

1. **Prerequisites Check**
   - Verifies kubectl, helm, and cluster availability
   - Checks Docker access

2. **Image Management**
   - Pulls images from GHCR
   - Verifies image architectures
   - Ensures images are available locally

3. **Deployment**
   - Creates namespace if needed
   - Generates Helm values file
   - Deploys using Helm chart
   - Sets up port forwarding

4. **Verification**
   - Waits for pods to be ready
   - Tests API endpoints
   - Validates frontend access

## 🔧 Manual Deployment

If you prefer manual deployment:

### 1. Pull Images
```bash
docker pull ghcr.io/akshaydubey29/mimir-insights-frontend:latest
docker pull ghcr.io/akshaydubey29/mimir-insights-backend:latest
```

### 2. Create Values File
```yaml
# values-manual.yaml
frontend:
  image:
    repository: akshaydubey29/mimir-insights-frontend
    tag: "latest"
    pullPolicy: Always

backend:
  image:
    repository: akshaydubey29/mimir-insights-backend
    tag: "latest"
    pullPolicy: Always
```

### 3. Deploy with Helm
```bash
helm upgrade --install mimir-insights ./deployments/helm-chart \
  --namespace mimir-insights \
  --values values-manual.yaml \
  --wait
```

### 4. Setup Port Forwarding
```bash
kubectl port-forward -n mimir-insights svc/mimir-insights-backend 8080:8080 &
kubectl port-forward -n mimir-insights svc/mimir-insights-frontend 8081:80 &
```

## 🎯 Workflow Benefits

### For Development
- **Resource Efficiency**: No local multi-arch builds
- **Consistency**: Same images across all environments
- **Speed**: Pre-built images ready for deployment
- **Reliability**: GitHub's infrastructure handles builds

### For Production
- **Multi-Platform**: Images work on any architecture
- **Versioning**: Proper semantic versioning
- **Security**: Built in controlled environment
- **Auditability**: Full build history and logs

## 📋 Available Versions

### Latest Images
- `ghcr.io/akshaydubey29/mimir-insights-frontend:latest`
- `ghcr.io/akshaydubey29/mimir-insights-backend:latest`

### Branch Images
- `ghcr.io/akshaydubey29/mimir-insights-frontend:main-<sha>`
- `ghcr.io/akshaydubey29/mimir-insights-backend:main-<sha>`

### Release Images
- `ghcr.io/akshaydubey29/mimir-insights-frontend:v1.0.0`
- `ghcr.io/akshaydubey29/mimir-insights-backend:v1.0.0`

## 🔍 Monitoring and Debugging

### Check Build Status
```bash
# View GitHub Actions runs
gh run list --workflow=build-multiarch.yml

# View specific run
gh run view <run-id>
```

### Check Image Availability
```bash
# List available tags
gh api /user/packages/container/mimir-insights-frontend/versions

# Check image details
docker inspect ghcr.io/akshaydubey29/mimir-insights-frontend:latest
```

### Debug Deployment
```bash
# Check pod status
kubectl get pods -n mimir-insights

# View logs
kubectl logs -f -l app.kubernetes.io/name=mimir-insights -n mimir-insights

# Check services
kubectl get services -n mimir-insights

# Check ingress
kubectl get ingress -n mimir-insights
```

## 🚨 Troubleshooting

### Common Issues

1. **Image Pull Failed**
   ```bash
   # Check if image exists
   docker pull ghcr.io/akshaydubey29/mimir-insights-frontend:latest
   
   # Check GitHub Actions build status
   gh run list --workflow=build-multiarch.yml
   ```

2. **Port Forwarding Issues**
   ```bash
   # Kill existing port forwards
   pkill -f "kubectl port-forward"
   
   # Check if ports are in use
   lsof -i :8080
   lsof -i :8081
   ```

3. **Helm Deployment Failed**
   ```bash
   # Check Helm status
   helm status mimir-insights -n mimir-insights
   
   # Check events
   kubectl get events -n mimir-insights --sort-by='.lastTimestamp'
   ```

4. **Cluster Issues**
   ```bash
   # Restart kind cluster
   kind delete cluster
   kind start cluster
   ```

## 🔄 Workflow Customization

### Adding New Triggers
Edit `.github/workflows/build-multiarch.yml`:
```yaml
on:
  push:
    branches: [ main, develop, feature/* ]
    paths:
      - 'cmd/**'
      - 'pkg/**'
      - 'web-ui/**'
```

### Adding New Platforms
```yaml
platforms: linux/amd64,linux/arm64,linux/386
```

### Custom Build Args
```yaml
build-args: |
  BUILDKIT_INLINE_CACHE=1
  BUILD_DATE=${{ env.BUILD_DATE }}
  VCS_REF=${{ github.sha }}
  VERSION=${{ github.event.inputs.version || 'v1.0.0' }}
  CUSTOM_ARG=value
```

## 📚 Related Documentation

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Docker Buildx](https://docs.docker.com/buildx/working-with-buildx/)
- [Helm Documentation](https://helm.sh/docs/)
- [Kind Documentation](https://kind.sigs.k8s.io/)

## 🤝 Contributing

To contribute to the CI/CD workflow:

1. **Fork the repository**
2. **Create a feature branch**
3. **Make your changes**
4. **Test locally using the deployment script**
5. **Submit a pull request**

The workflow will automatically test your changes and build new images for review. 
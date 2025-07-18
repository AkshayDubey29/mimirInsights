# Making GitHub Container Registry Images Public

This guide explains how to make your GitHub Container Registry (ghcr.io) images public so they can be pulled without authentication.

## Why Make Images Public?

- **No authentication required**: Eliminates the need for image pull secrets
- **Simpler deployment**: Reduces configuration complexity
- **Faster deployments**: No token management or secret rotation
- **Better for open source**: Allows community to easily use your images

## Making Images Public

### Option 1: Make Repository Public (Recommended)

1. **Go to your GitHub repository**:
   ```
   https://github.com/AkshayDubey29/mimirInsights
   ```

2. **Navigate to Settings**:
   - Click on "Settings" tab in your repository

3. **Go to General settings**:
   - Scroll down to "Danger Zone" section

4. **Change repository visibility**:
   - Click "Change repository visibility"
   - Select "Make public"
   - Confirm the change

5. **Verify the change**:
   - The repository should now show as "Public" at the top

### Option 2: Make Only the Package Public

1. **Go to your GitHub profile**:
   ```
   https://github.com/AkshayDubey29
   ```

2. **Click on "Packages" tab**:
   - Find your container packages

3. **For each package** (`mimir-insights-backend`, `mimir-insights-frontend`):
   - Click on the package name
   - Go to "Package settings"
   - Scroll down to "Danger Zone"
   - Click "Change package visibility"
   - Select "Public"
   - Confirm the change

## Verifying Public Access

### Test Image Pull Locally

```bash
# Test pulling the backend image
docker pull ghcr.io/akshaydubey29/mimir-insights-backend:v1.0.0-20250718-110355

# Test pulling the frontend image
docker pull ghcr.io/akshaydubey29/mimir-insights-frontend:v1.0.0-20250718-110355
```

### Test in Kubernetes

```bash
# Test pulling in a pod
kubectl run test-pull --image=ghcr.io/akshaydubey29/mimir-insights-backend:v1.0.0-20250718-110355 -n mimir-insights

# Check if it succeeds
kubectl get pod test-pull -n mimir-insights
```

## Updated Deployment Configuration

Once images are public, the deployment configuration has been simplified:

### Removed Components

- ❌ `imagePullSecrets` configuration
- ❌ GitHub PAT setup
- ❌ Docker registry secrets
- ❌ Authentication tokens

### Simplified Values Files

The production values files now use:

```yaml
# Backend Configuration
backend:
  image:
    repository: ghcr.io/akshaydubey29/mimir-insights-backend
    tag: "v1.0.0-20250718-110355"
    pullPolicy: Always
  # No imagePullSecrets needed

# Frontend Configuration
frontend:
  image:
    repository: ghcr.io/akshaydubey29/mimir-insights-frontend
    tag: "v1.0.0-20250718-110355"
    pullPolicy: Always
  # No imagePullSecrets needed
```

## Deployment Script

Use the new deployment script for public images:

```bash
# Make script executable
chmod +x deploy-production-public.sh

# Deploy with public images
./deploy-production-public.sh
```

## Benefits of Public Images

### For Development
- ✅ No authentication setup required
- ✅ Faster local development
- ✅ Easier CI/CD integration
- ✅ No token management

### For Production
- ✅ Simplified deployment
- ✅ No secret rotation needed
- ✅ Reduced configuration complexity
- ✅ Better reliability

### For Community
- ✅ Easy to try out your project
- ✅ No barriers to adoption
- ✅ Transparent and accessible
- ✅ Encourages contributions

## Security Considerations

### What's Public
- ✅ Container images
- ✅ Application code (if repository is public)
- ✅ Dockerfile and build context

### What's Still Private
- ✅ Application secrets and configuration
- ✅ Database credentials
- ✅ API keys
- ✅ Environment-specific settings

## Troubleshooting

### Image Still Not Accessible

1. **Check repository visibility**:
   ```bash
   # Check if repository is public
   curl -I https://github.com/AkshayDubey29/mimirInsights
   ```

2. **Verify package visibility**:
   - Go to your GitHub profile → Packages
   - Check if packages show as "Public"

3. **Test with different tag**:
   ```bash
   # Try with 'latest' tag
   docker pull ghcr.io/akshaydubey29/mimir-insights-backend:latest
   ```

### Deployment Issues

1. **Clear old secrets**:
   ```bash
   kubectl delete secret ghcr-secret -n mimir-insights
   ```

2. **Redeploy without secrets**:
   ```bash
   helm upgrade mimir-insights ./deployments/helm-chart \
       -f ./deployments/helm-chart/values-production-final.yaml \
       -n mimir-insights
   ```

3. **Check pod events**:
   ```bash
   kubectl get events -n mimir-insights --sort-by='.lastTimestamp'
   ```

## Migration Steps

### From Private to Public

1. **Make images public** (see steps above)

2. **Update deployment**:
   ```bash
   # Remove old deployment with secrets
   helm uninstall mimir-insights -n mimir-insights
   
   # Deploy with public images
   ./deploy-production-public.sh
   ```

3. **Clean up secrets**:
   ```bash
   kubectl delete secret ghcr-secret -n mimir-insights
   ```

4. **Verify deployment**:
   ```bash
   kubectl get pods -n mimir-insights
   kubectl logs deployment/mimir-insights-backend -n mimir-insights
   ```

## Next Steps

After making images public:

1. **Test the deployment**:
   ```bash
   ./deploy-production-public.sh
   ```

2. **Verify functionality**:
   ```bash
   kubectl port-forward service/mimir-insights-frontend 8081:80 -n mimir-insights
   kubectl port-forward service/mimir-insights-backend 8080:8080 -n mimir-insights
   ```

3. **Update documentation**:
   - Remove references to image pull secrets
   - Update deployment guides
   - Simplify setup instructions

4. **Share with community**:
   - Update README with public image references
   - Add quick start guide
   - Document usage examples

## References

- [GitHub Container Registry Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Making Packages Public](https://docs.github.com/en/packages/learn-github-packages/configuring-a-packages-access-control-and-visibility)
- [Kubernetes Image Pull Without Secrets](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/) 
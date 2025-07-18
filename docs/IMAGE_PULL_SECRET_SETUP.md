# GitHub Container Registry Image Pull Secret Setup

This guide explains how to set up image pull secrets for GitHub Container Registry (ghcr.io) to resolve unauthorized access issues when pulling images in Kubernetes.

## Problem

When deploying MimirInsights with images from GitHub Container Registry, you may encounter errors like:
- `ErrImagePull`
- `ImagePullBackOff`
- `unauthorized: authentication required`

This happens because Kubernetes needs authentication to pull images from private registries.

## Solution

Create a Kubernetes secret with your GitHub Personal Access Token (PAT) to authenticate with ghcr.io.

## Prerequisites

1. **GitHub Personal Access Token (PAT)** with `read:packages` scope
2. **kubectl** configured to access your cluster
3. **GitHub username**: `akshaydubey29`

## Step 1: Create GitHub Personal Access Token

1. Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Give it a descriptive name (e.g., "MimirInsights Image Pull")
4. Select scopes:
   - ✅ `read:packages` (required for pulling images)
5. Click "Generate token"
6. **Copy the token immediately** (you won't see it again)

## Step 2: Create Image Pull Secret

### Option A: Using the Automated Script

```bash
# Run the setup script
./create-image-pull-secret.sh
```

The script will prompt you for:
- GitHub Personal Access Token
- GitHub email address

### Option B: Manual Creation

```bash
# Replace YOUR_GITHUB_PAT and YOUR_EMAIL with actual values
kubectl create secret docker-registry ghcr-secret \
    --docker-server=ghcr.io \
    --docker-username=akshaydubey29 \
    --docker-password=YOUR_GITHUB_PAT \
    --docker-email=YOUR_EMAIL \
    -n mimir-insights
```

### Option C: Using YAML Manifest

Create a file `ghcr-secret.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ghcr-secret
  namespace: mimir-insights
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: <base64-encoded-docker-config>
```

Generate the base64 encoded config:

```bash
# Create Docker config JSON
echo -n '{"auths":{"ghcr.io":{"username":"akshaydubey29","password":"YOUR_GITHUB_PAT","email":"YOUR_EMAIL","auth":"'$(echo -n "akshaydubey29:YOUR_GITHUB_PAT" | base64)'"}}}' | base64
```

Apply the secret:

```bash
kubectl apply -f ghcr-secret.yaml
```

## Step 3: Verify Secret Creation

```bash
# Check if secret exists
kubectl get secret ghcr-secret -n mimir-insights

# View secret details (without sensitive data)
kubectl describe secret ghcr-secret -n mimir-insights
```

## Step 4: Update Helm Values

The production values files have been updated to include the image pull secret:

```yaml
# Backend configuration
backend:
  imagePullSecrets:
    - name: ghcr-secret

# Frontend configuration  
frontend:
  imagePullSecrets:
    - name: ghcr-secret
```

## Step 5: Redeploy Application

```bash
# For production deployment
helm upgrade mimir-insights ./deployments/helm-chart \
    -f ./deployments/helm-chart/values-production-final.yaml \
    -n mimir-insights

# For AWS ALB deployment
helm upgrade mimir-insights ./deployments/helm-chart \
    -f ./deployments/helm-chart/values-production-aws-alb.yaml \
    -n mimir-insights
```

## Step 6: Verify Deployment

```bash
# Check pod status
kubectl get pods -n mimir-insights

# Check for image pull errors
kubectl describe pod <pod-name> -n mimir-insights

# Check pod logs
kubectl logs <pod-name> -n mimir-insights
```

## Troubleshooting

### Common Issues

1. **Secret not found**
   ```bash
   # Verify secret exists
   kubectl get secret ghcr-secret -n mimir-insights
   ```

2. **Invalid credentials**
   - Ensure GitHub PAT has `read:packages` scope
   - Verify username is correct (`akshaydubey29`)
   - Check if PAT is expired

3. **Wrong namespace**
   ```bash
   # Ensure secret is in the correct namespace
   kubectl get secret ghcr-secret -n mimir-insights
   ```

4. **Image pull still failing**
   ```bash
   # Check pod events
   kubectl describe pod <pod-name> -n mimir-insights
   
   # Check if imagePullSecrets is configured
   kubectl get pod <pod-name> -n mimir-insights -o yaml | grep -A 5 imagePullSecrets
   ```

### Debugging Commands

```bash
# Test image pull manually
kubectl run test-pull --image=ghcr.io/akshaydubey29/mimir-insights-backend:v1.0.0-20250718-110355 \
    --image-pull-secret=ghcr-secret -n mimir-insights

# Check secret data (base64 encoded)
kubectl get secret ghcr-secret -n mimir-insights -o yaml

# Decode secret data
kubectl get secret ghcr-secret -n mimir-insights -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d
```

## Security Best Practices

1. **Use minimal scope**: Only grant `read:packages` scope to the PAT
2. **Rotate tokens regularly**: Update the secret when you rotate your GitHub PAT
3. **Namespace isolation**: Keep secrets in the specific namespace where they're needed
4. **Monitor usage**: Regularly check GitHub token usage and revoke unused tokens

## Alternative: Using Service Account

For production environments, consider using a service account with image pull secrets:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: mimir-insights-sa
  namespace: mimir-insights
imagePullSecrets:
- name: ghcr-secret
```

Then reference the service account in your deployment:

```yaml
spec:
  serviceAccountName: mimir-insights-sa
```

## Next Steps

After setting up the image pull secret:

1. Deploy MimirInsights using the updated values files
2. Verify all pods are running successfully
3. Test port-forwarding to access the application
4. Configure ingress for external access (if needed)

## References

- [GitHub Container Registry Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Kubernetes Image Pull Secrets](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)
- [GitHub Personal Access Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token) 
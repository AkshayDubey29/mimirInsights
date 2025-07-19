# Manual Deployment Guide

## Overview

This guide explains how to manually deploy MimirInsights after the CI/CD pipeline successfully builds and pushes containers to GHCR.

## Workflow

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Code Push     │    │  GitHub Actions  │    │  Manual Deploy  │
│                 │    │                  │    │                 │
│ • Git Push      │───▶│ • Build Images   │───▶│ • Update Values │
│ • Trigger CI    │    │ • Push to GHCR   │    │ • Deploy Helm   │
│                 │    │ • Generate Tags  │    │ • Verify Status │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Step 1: Monitor CI/CD Pipeline

1. **Check GitHub Actions**: https://github.com/AkshayDubey29/mimirInsights/actions
2. **Wait for Success**: Look for "Build and Push Multi-Architecture Docker Images" workflow
3. **Note Timestamp**: Copy the timestamp from the workflow output (e.g., `20250719-143000`)

## Step 2: Update Values File

Use the provided script to update the values file with the new timestamp:

```bash
# Replace TIMESTAMP with the actual timestamp from CI/CD
./update-values.sh TIMESTAMP

# Example:
./update-values.sh 20250719-143000
```

This will:
- Update `deployments/helm-chart/values-production-final.yaml`
- Create a backup of the previous values
- Show the updated image references

## Step 3: Deploy to Kind Cluster

Deploy using Helm:

```bash
helm upgrade --install mimir-insights ./deployments/helm-chart \
  --namespace mimir-insights \
  --create-namespace \
  --values ./deployments/helm-chart/values-production-final.yaml \
  --wait \
  --timeout 10m
```

## Step 4: Verify Deployment

Check the deployment status:

```bash
# Check pods
kubectl get pods -n mimir-insights

# Check services
kubectl get svc -n mimir-insights

# Check logs if needed
kubectl logs -n mimir-insights -l app.kubernetes.io/name=mimir-insights
```

## Step 5: Setup Port Forwarding

```bash
# Kill any existing port-forward processes
pkill -f "kubectl port-forward" || true

# Setup port forwarding
kubectl port-forward -n mimir-insights svc/mimir-insights-backend 8080:8080 &
kubectl port-forward -n mimir-insights svc/mimir-insights-frontend 8081:80 &
```

## Step 6: Access Application

- **Frontend**: http://localhost:8081
- **Backend API**: http://localhost:8080/api/tenants
- **Health Check**: http://localhost:8080/api/health

## Troubleshooting

### Image Pull Issues
If pods show `ImagePullBackOff`:
```bash
# Check if image exists
docker pull ghcr.io/akshaydubey29/mimir-insights-backend:TIMESTAMP
docker pull ghcr.io/akshaydubey29/mimir-insights-frontend:TIMESTAMP
```

### Port Forwarding Issues
If port forwarding fails:
```bash
# Kill existing processes
pkill -f "kubectl port-forward"

# Check if ports are in use
lsof -i :8080
lsof -i :8081
```

### Pod Issues
If pods are not ready:
```bash
# Check pod events
kubectl describe pods -n mimir-insights

# Check pod logs
kubectl logs -n mimir-insights <pod-name>
```

## Complete Deployment Script

For convenience, you can use the complete deployment script:

```bash
# This does everything: update values + deploy + port forward
./deploy-from-ci.sh TIMESTAMP
```

## Rollback

If deployment fails, you can rollback:

```bash
# Uninstall current release
helm uninstall mimir-insights -n mimir-insights

# Restore previous values from backup
cp deployments/helm-chart/values-production-final.yaml.backup.* deployments/helm-chart/values-production-final.yaml

# Redeploy with previous images
helm upgrade --install mimir-insights ./deployments/helm-chart \
  --namespace mimir-insights \
  --values ./deployments/helm-chart/values-production-final.yaml \
  --wait
``` 
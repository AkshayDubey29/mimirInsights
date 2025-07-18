# 🚀 MimirInsights Production Deployment Guide

This guide provides step-by-step instructions for deploying MimirInsights to your production Kubernetes cluster where Mimir and tenant namespaces are running.

## 📋 Prerequisites

- Access to production Kubernetes cluster
- `kubectl` configured for production cluster
- `helm` v3.x installed
- Docker registry access (if using private registry)
- Mimir deployment already running in the cluster
- Tenant namespaces already created

## 🏗️ Build and Push Images

### Option 1: Build Locally and Push to Registry

```bash
# Build both frontend and backend images
./build-fast.sh

# Tag images for your registry
docker tag mimir-insights-frontend:fast-$(date +%s) your-registry.com/mimir-insights-frontend:latest
docker tag mimir-insights-backend:fast-$(date +%s) your-registry.com/mimir-insights-backend:latest

# Push to registry
docker push your-registry.com/mimir-insights-frontend:latest
docker push your-registry.com/mimir-insights-backend:latest
```

### Option 2: Use GitHub Container Registry

```bash
# Build and push to GHCR
./build-fast.sh ghcr.io/your-username/mimir-insights-frontend ghcr.io/your-username/mimir-insights-backend latest
```

## 🔧 Production Configuration

### 1. Update Helm Values

Edit `deployments/helm-chart/values-production.yaml` with your production settings:

```yaml
# Update image repositories
frontend:
  image:
    repository: your-registry.com/mimir-insights-frontend
    tag: latest
    pullPolicy: Always

backend:
  image:
    repository: your-registry.com/mimir-insights-backend
    tag: latest
    pullPolicy: Always

# Update ingress configuration
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: mimir-insights.your-domain.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: mimir-insights-tls
      hosts:
        - mimir-insights.your-domain.com

# Resource limits for production
frontend:
  resources:
    requests:
      memory: "128Mi"
      cpu: "100m"
    limits:
      memory: "256Mi"
      cpu: "200m"

backend:
  resources:
    requests:
      memory: "512Mi"
      cpu: "250m"
    limits:
      memory: "1Gi"
      cpu: "500m"
```

### 2. Create Production Namespace

```bash
kubectl create namespace mimir-insights
```

### 3. Set Up RBAC (if not using Helm)

```bash
kubectl apply -f deployments/helm-chart/templates/rbac.yaml
```

## 🚀 Deploy to Production

### Option 1: Using Production Deployment Script

```bash
# Make script executable
chmod +x deploy-production.sh

# Deploy with production values
./deploy-production.sh
```

### Option 2: Manual Helm Deployment

```bash
# Add the chart repository (if using remote chart)
helm repo add mimir-insights https://your-chart-repo.com

# Deploy using production values
helm upgrade --install mimir-insights ./deployments/helm-chart \
  --namespace mimir-insights \
  --values deployments/helm-chart/values-production.yaml \
  --wait \
  --timeout=10m
```

### Option 3: Using Simple Deployment

```bash
# Apply the simple deployment
kubectl apply -f deploy-simple.yaml
```

## 🔍 Verification Steps

### 1. Check Pod Status

```bash
kubectl get pods -n mimir-insights
```

Expected output:
```
NAME                                       READY   STATUS    RESTARTS   AGE
mimir-insights-backend-xxx-xxx             1/1     Running   0          2m
mimir-insights-frontend-xxx-xxx            1/1     Running   0          2m
```

### 2. Check Services

```bash
kubectl get svc -n mimir-insights
```

### 3. Check Ingress

```bash
kubectl get ingress -n mimir-insights
```

### 4. Test Backend API

```bash
# Port forward to test locally
kubectl port-forward svc/mimir-insights-backend 8080:8080 -n mimir-insights

# Test health endpoint
curl http://localhost:8080/api/health
```

### 5. Test Frontend

```bash
# Port forward to test locally
kubectl port-forward svc/mimir-insights-frontend 8081:80 -n mimir-insights

# Open browser to http://localhost:8081
```

## 🔧 Configuration

### Environment Variables

The backend can be configured using environment variables:

```yaml
env:
  - name: MIMIR_NAMESPACE
    value: "mimir"  # Your Mimir namespace
  - name: MIMIR_API_URL
    value: "http://mimir-service:9009"  # Your Mimir service URL
  - name: LOG_LEVEL
    value: "info"
  - name: K8S_IN_CLUSTER
    value: "true"
  - name: MIMIR_AUTO_DISCOVER
    value: "true"
```

### Auto-Discovery Configuration

The system will automatically discover:
- Alloy workloads (Deployments, StatefulSets, DaemonSets)
- Tenant namespaces
- Mimir configurations
- Resource limits and requests

## 📊 Monitoring and Logs

### View Logs

```bash
# Backend logs
kubectl logs -f deployment/mimir-insights-backend -n mimir-insights

# Frontend logs
kubectl logs -f deployment/mimir-insights-frontend -n mimir-insights
```

### Check Health

```bash
# Backend health
kubectl exec -it deployment/mimir-insights-backend -n mimir-insights -- wget -qO- http://localhost:8080/api/health

# Frontend health
kubectl exec -it deployment/mimir-insights-frontend -n mimir-insights -- wget -qO- http://localhost:80
```

## 🔒 Security Considerations

### 1. Network Policies

Create network policies to restrict traffic:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: mimir-insights-network-policy
  namespace: mimir-insights
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 80
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: mimir
    ports:
    - protocol: TCP
      port: 9009
```

### 2. RBAC Configuration

Ensure proper RBAC is configured:

```bash
kubectl apply -f deployments/helm-chart/templates/rbac.yaml
```

### 3. Secrets Management

For production, use Kubernetes secrets for sensitive data:

```bash
kubectl create secret generic mimir-insights-secrets \
  --from-literal=api-key=your-api-key \
  --namespace mimir-insights
```

## 🔄 Updates and Rollbacks

### Update Deployment

```bash
# Build new images
./build-fast.sh

# Update deployment
helm upgrade mimir-insights ./deployments/helm-chart \
  --namespace mimir-insights \
  --values deployments/helm-chart/values-production.yaml
```

### Rollback

```bash
# List releases
helm list -n mimir-insights

# Rollback to previous version
helm rollback mimir-insights 1 -n mimir-insights
```

## 🚨 Troubleshooting

### Common Issues

1. **Image Pull Errors**
   ```bash
   # Check image pull policy
   kubectl describe pod -n mimir-insights
   
   # Verify image exists in registry
   docker pull your-registry.com/mimir-insights-backend:latest
   ```

2. **Port Forwarding Issues**
   ```bash
   # Kill existing port forwards
   pkill -f "port-forward"
   
   # Try different ports
   kubectl port-forward svc/mimir-insights-frontend 8082:80 -n mimir-insights
   ```

3. **Backend Connection Issues**
   ```bash
   # Check if Mimir is accessible
   kubectl exec -it deployment/mimir-insights-backend -n mimir-insights -- wget -qO- http://mimir-service:9009/api/v1/status/config
   ```

4. **Frontend Not Loading**
   ```bash
   # Check nginx configuration
   kubectl exec -it deployment/mimir-insights-frontend -n mimir-insights -- nginx -t
   
   # Check nginx logs
   kubectl logs deployment/mimir-insights-frontend -n mimir-insights
   ```

### Debug Commands

```bash
# Get detailed pod information
kubectl describe pod -n mimir-insights

# Check events
kubectl get events -n mimir-insights --sort-by='.lastTimestamp'

# Check resource usage
kubectl top pods -n mimir-insights
```

## 📞 Support

If you encounter issues:

1. Check the logs: `kubectl logs -n mimir-insights`
2. Verify configuration: `kubectl describe deployment -n mimir-insights`
3. Test connectivity: `kubectl exec -it -n mimir-insights -- wget -qO- http://localhost:8080/api/health`

## 🎯 Next Steps

After successful deployment:

1. Configure monitoring and alerting
2. Set up backup strategies
3. Implement CI/CD pipelines
4. Configure SSL certificates
5. Set up monitoring dashboards

---

**Note**: This deployment guide assumes you have a production Kubernetes cluster with Mimir already deployed. Adjust the configuration based on your specific environment and requirements. 
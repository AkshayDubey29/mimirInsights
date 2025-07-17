# MimirInsights Deployment Guide

This guide walks you through the complete deployment process for MimirInsights, from building containers to testing the application with mock data.

## 🎯 Overview

The deployment pipeline includes:
1. **Container Build & Push**: Build and push images to GitHub Container Registry
2. **Kind Cluster Setup**: Create/use the mimirInsights-test cluster
3. **Helm Deployment**: Deploy using Helm charts with proper values
4. **Mock Data Setup**: Create test environment with mock Mimir components
5. **Validation**: Test all features and endpoints

## 📋 Prerequisites

Ensure you have the following tools installed:
- Docker
- Kind
- kubectl
- Helm 3.x
- curl and jq (for testing)

## 🚀 Quick Start

### 1. Run the Complete Deployment Pipeline

```bash
# Make scripts executable
chmod +x deploy.sh mock-data-generator.sh

# Run the complete deployment pipeline
./deploy.sh
```

### 2. Set up Mock Data Environment

```bash
# Create mock Mimir environment for testing
./mock-data-generator.sh
```

### 3. Access the Application

```bash
# Start port forwarding
./port-forward.sh

# In another terminal, test the API
./test-api.sh
```

## 📦 Deployment Components

### Container Images

The deployment builds and pushes the following images to `ghcr.io/akshaydubey29`:

- **Backend**: `mimir-insights-backend:YYYYMMDD-HHMMSS`
- **Frontend**: `mimir-insights-frontend:YYYYMMDD-HHMMSS`

Both images are also tagged with `:latest` for convenience.

### Kubernetes Resources

The Helm chart deploys:

- **Namespace**: `mimir-insights`
- **Deployments**: Backend (2 replicas) and Frontend (2 replicas)
- **Services**: ClusterIP services for both components
- **Ingress**: NGINX ingress with routes for API and UI
- **RBAC**: Service account with read-only permissions
- **HPA**: Horizontal Pod Autoscaler for scaling

### Configuration

Default configuration in `values.yaml`:

```yaml
# Backend Configuration
backend:
  replicaCount: 2
  image:
    repository: mimir-insights-backend
    pullPolicy: Always
  env:
    - name: MIMIR_NAMESPACE
      value: mimir-test  # Updated for mock environment
    - name: MIMIR_API_URL
      value: http://mimir-distributor.mimir-test:9090

# Frontend Configuration
frontend:
  replicaCount: 2
  image:
    repository: mimir-insights-frontend
    pullPolicy: Always

# Ingress Configuration
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: mimir-insights.local
```

## 🧪 Testing and Validation

### Mock Environment

The mock data generator creates:

1. **Mock Mimir Cluster** (`mimir-test` namespace):
   - Distributor (2 replicas)
   - Ingester (3 replicas)
   - Querier (2 replicas)
   - Prometheus (for metrics simulation)

2. **Mock Tenant Namespaces**:
   - `team-a`, `team-b`, `team-c`
   - Each with Alloy agent deployment
   - Proper labels for tenant discovery

### API Testing

Use the generated `test-api.sh` script to validate:

```bash
./test-api.sh
```

Tests include:
- Health checks
- Discovery endpoints
- Tenant listing
- Metrics collection
- Limits analysis

### Access Points

After deployment, access the application via:

- **Frontend UI**: http://localhost:8080
- **Backend API**: http://localhost:8081/api
- **Ingress** (add to `/etc/hosts`): http://mimir-insights.local

## 🔧 Manual Deployment Steps

If you prefer manual steps or need to customize the deployment:

### 1. Build Containers

```bash
# Set timestamp
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

# Build backend
docker build -f Dockerfile.backend -t ghcr.io/akshaydubey29/mimir-insights-backend:$TIMESTAMP .

# Build frontend
docker build -f Dockerfile.frontend -t ghcr.io/akshaydubey29/mimir-insights-frontend:$TIMESTAMP .
```

### 2. Push to Registry

```bash
# Login to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u akshaydubey29 --password-stdin

# Push images
docker push ghcr.io/akshaydubey29/mimir-insights-backend:$TIMESTAMP
docker push ghcr.io/akshaydubey29/mimir-insights-frontend:$TIMESTAMP
```

### 3. Deploy with Helm

```bash
# Install/upgrade
helm upgrade --install mimir-insights ./deployments/helm-chart \
  --namespace mimir-insights \
  --create-namespace \
  --set backend.image.tag=$TIMESTAMP \
  --set frontend.image.tag=$TIMESTAMP \
  --set imageRegistry=ghcr.io/akshaydubey29 \
  --wait
```

## 🐛 Troubleshooting

### Common Issues

1. **Images not pulling**:
   ```bash
   # Check if images exist
   docker pull ghcr.io/akshaydubey29/mimir-insights-backend:latest
   
   # Verify registry access
   docker login ghcr.io -u akshaydubey29
   ```

2. **Pods not starting**:
   ```bash
   # Check pod status
   kubectl get pods -n mimir-insights
   
   # View logs
   kubectl logs -n mimir-insights deployment/mimir-insights-backend
   ```

3. **Ingress not working**:
   ```bash
   # Check ingress controller
   kubectl get pods -n ingress-nginx
   
   # Verify ingress rules
   kubectl describe ingress -n mimir-insights
   ```

### Useful Commands

```bash
# View all resources
kubectl get all -n mimir-insights

# Check Helm release
helm list -n mimir-insights

# Port forward manually
kubectl port-forward -n mimir-insights service/mimir-insights-frontend 8080:80
kubectl port-forward -n mimir-insights service/mimir-insights-backend 8081:8080

# View backend logs
kubectl logs -f -n mimir-insights deployment/mimir-insights-backend

# Check configuration
kubectl get configmap -n mimir-insights
kubectl describe configmap mimir-insights-config -n mimir-insights
```

## 🧹 Cleanup

To remove the deployment:

```bash
# Remove Helm release
helm uninstall mimir-insights -n mimir-insights

# Remove namespace
kubectl delete namespace mimir-insights

# Remove mock data
kubectl delete namespace mimir-test team-a team-b team-c

# Remove kind cluster (optional)
kind delete cluster --name mimirInsights-test
```

## 📝 Configuration Customization

### Environment Variables

Customize backend behavior via values.yaml:

```yaml
backend:
  env:
    - name: MIMIR_NAMESPACE
      value: "your-mimir-namespace"
    - name: MIMIR_API_URL
      value: "http://your-mimir-distributor:9090"
    - name: LOG_LEVEL
      value: "debug"
    - name: K8S_IN_CLUSTER
      value: "true"
```

### Resource Limits

Adjust resource allocation:

```yaml
backend:
  resources:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 1000m
      memory: 2Gi
```

### Scaling

Configure autoscaling:

```yaml
hpa:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
```

## 🔐 Security

The deployment includes:

- Non-root containers
- Read-only filesystem mounts
- RBAC with minimal permissions
- Security contexts
- Network policies (optional)

## 📊 Monitoring

The backend exposes Prometheus metrics at `/metrics`. Configure monitoring:

```yaml
monitoring:
  serviceMonitor:
    enabled: true
    interval: 30s
    path: /metrics
```

## 🎉 Success Criteria

A successful deployment should show:

1. ✅ All pods running and ready
2. ✅ Services accessible via port-forward
3. ✅ API endpoints responding
4. ✅ Frontend loading correctly
5. ✅ Mock data discovery working
6. ✅ Tenant analysis functioning

Ready to deploy? Run `./deploy.sh` and follow the output!
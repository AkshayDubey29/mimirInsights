# MimirInsights Production Deployment Guide

This guide provides step-by-step instructions for deploying MimirInsights to production with full security, monitoring, and high availability configurations.

## Prerequisites

1. **Kubernetes Cluster**: Production-ready cluster (EKS, GKE, AKS, or on-premises)
2. **kubectl**: Configured and authenticated to your cluster
3. **Helm**: Version 3.x installed
4. **GitHub Personal Access Token**: With `read:packages` scope for pulling images
5. **AWS ALB Ingress Controller**: If using AWS (already configured in values)

## Step 1: Create Image Pull Secret

Before deploying, you need to create a secret for pulling images from GitHub Container Registry:

```bash
# Run the image pull secret creation script
./create-image-pull-secret.sh
```

This script will:
- Create the `mimir-insights` namespace if it doesn't exist
- Create a `ghcr-secret` for authenticating with GitHub Container Registry
- Prompt for your GitHub Personal Access Token

## Step 2: Verify Production Values

Review the production values file: `deployments/helm-chart/values-production-final.yaml`

Key configurations included:
- **Security**: Non-root containers, read-only filesystems, dropped capabilities
- **High Availability**: Pod anti-affinity, multiple replicas, HPA
- **Resource Management**: Resource quotas, limits, and requests
- **Monitoring**: Health checks, liveness/readiness probes
- **Network**: Network policies, ingress with AWS ALB annotations

## Step 3: Deploy to Production

### Option A: Standard Production Deployment

```bash
# Deploy using the production values
helm upgrade --install mimir-insights \
  ./deployments/helm-chart \
  --namespace mimir-insights \
  --values deployments/helm-chart/values-production-final.yaml \
  --create-namespace \
  --wait \
  --timeout 10m
```

### Option B: AWS ALB Production Deployment

```bash
# Deploy using the AWS ALB specific values
helm upgrade --install mimir-insights \
  ./deployments/helm-chart \
  --namespace mimir-insights \
  --values deployments/helm-chart/values-production-aws-alb.yaml \
  --create-namespace \
  --wait \
  --timeout 10m
```

## Step 4: Verify Deployment

Check the deployment status:

```bash
# Check all resources
kubectl get all -n mimir-insights

# Check pod status
kubectl get pods -n mimir-insights

# Check services
kubectl get svc -n mimir-insights

# Check ingress
kubectl get ingress -n mimir-insights

# Check HPA
kubectl get hpa -n mimir-insights
```

## Step 5: Access the Application

### Local Access (for testing)

```bash
# Port forward to frontend
kubectl port-forward service/mimir-insights-frontend 8081:80 -n mimir-insights

# Port forward to backend
kubectl port-forward service/mimir-insights-backend 8080:8080 -n mimir-insights
```

Access the application at:
- Frontend: http://localhost:8081
- Backend API: http://localhost:8080

### Production Access

Once the ingress is configured and DNS is set up, access via:
- Frontend: https://mimir-insights.yourdomain.com
- Backend API: https://mimir-insights.yourdomain.com/api

## Step 6: Monitor and Troubleshoot

### Check Logs

```bash
# Backend logs
kubectl logs -f deployment/mimir-insights-backend -n mimir-insights

# Frontend logs
kubectl logs -f deployment/mimir-insights-frontend -n mimir-insights
```

### Check Events

```bash
# Pod events
kubectl get events -n mimir-insights --sort-by='.lastTimestamp'

# Specific pod events
kubectl describe pod <pod-name> -n mimir-insights
```

### Health Checks

```bash
# Check health endpoints
kubectl exec -it deployment/mimir-insights-backend -n mimir-insights -- curl -f http://localhost:8080/healthz
kubectl exec -it deployment/mimir-insights-frontend -n mimir-insights -- curl -f http://localhost:80/healthz
```

## Production Features

### Security Features
- **Non-root containers**: All containers run as user 1000
- **Read-only filesystems**: Containers cannot write to filesystem
- **Dropped capabilities**: All Linux capabilities dropped
- **Network policies**: Restricted ingress/egress traffic
- **Pod security contexts**: Enforced security policies

### High Availability
- **Multiple replicas**: 2 replicas for both frontend and backend
- **Pod anti-affinity**: Pods scheduled on different nodes
- **Horizontal Pod Autoscaler**: Automatic scaling based on CPU/memory
- **Health checks**: Liveness and readiness probes

### Resource Management
- **Resource quotas**: Limits namespace resource usage
- **Resource limits**: Prevents resource exhaustion
- **Priority classes**: High priority for MimirInsights pods

### Monitoring
- **Health endpoints**: `/healthz` and `/ready` endpoints
- **Probe configuration**: Proper health check intervals
- **Logging**: Structured logging with configurable levels

## Troubleshooting

### Common Issues

1. **Image Pull Errors**
   ```bash
   # Check if secret exists
   kubectl get secret ghcr-secret -n mimir-insights
   
   # Recreate secret if needed
   ./create-image-pull-secret.sh
   ```

2. **Pod CrashLoopBackOff**
   ```bash
   # Check pod logs
   kubectl logs <pod-name> -n mimir-insights
   
   # Check pod description
   kubectl describe pod <pod-name> -n mimir-insights
   ```

3. **Ingress Issues**
   ```bash
   # Check ingress status
   kubectl describe ingress -n mimir-insights
   
   # Check ALB controller logs
   kubectl logs -n kube-system deployment/aws-load-balancer-controller
   ```

4. **Resource Issues**
   ```bash
   # Check resource usage
   kubectl top pods -n mimir-insights
   
   # Check resource quotas
   kubectl describe resourcequota -n mimir-insights
   ```

### Performance Tuning

1. **Adjust HPA settings** in values file:
   ```yaml
   hpa:
     backend:
       targetCPUUtilizationPercentage: 70
       targetMemoryUtilizationPercentage: 80
   ```

2. **Scale replicas** for higher load:
   ```yaml
   backend:
     replicaCount: 3
   frontend:
     replicaCount: 3
   ```

3. **Increase resource limits** if needed:
   ```yaml
   backend:
     resources:
       limits:
         memory: "1Gi"
         cpu: "1000m"
   ```

## Backup and Recovery

### Backup Configuration
The production deployment includes backup configuration:
- **Schedule**: Daily at 2 AM
- **Retention**: 30 days
- **Storage**: Configured via backup section in values

### Recovery Process
1. Restore from backup
2. Update image tags if needed
3. Redeploy with `helm upgrade`

## Maintenance

### Updating Images
```bash
# Update image tags in values file
# Then redeploy
helm upgrade mimir-insights ./deployments/helm-chart \
  --namespace mimir-insights \
  --values deployments/helm-chart/values-production-final.yaml
```

### Scaling
```bash
# Scale manually if needed
kubectl scale deployment mimir-insights-backend --replicas=3 -n mimir-insights
kubectl scale deployment mimir-insights-frontend --replicas=3 -n mimir-insights
```

### Cleanup
```bash
# Uninstall completely
helm uninstall mimir-insights -n mimir-insights
kubectl delete namespace mimir-insights
```

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review pod logs and events
3. Verify configuration in values file
4. Check Kubernetes cluster health

## Security Notes

- All containers run as non-root users
- Network policies restrict traffic
- Secrets are properly managed
- Health checks prevent serving unhealthy pods
- Resource limits prevent DoS attacks

This production deployment provides enterprise-grade security, reliability, and scalability for MimirInsights. 